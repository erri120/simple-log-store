package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
	"github.com/oklog/ulid/v2"
	"io"
	"log/slog"
	"net/http"
	"os"
	"simple-log-store/internal/config"
	"simple-log-store/internal/logs"
	"simple-log-store/internal/redis"
	"simple-log-store/internal/storage"
	"simple-log-store/internal/utils"
	"time"
)

// Handler for the `/logs` endpoint.
type logsHandler struct {
	singleFileLimit    uint64
	maxFileCount       uint16
	contentLengthLimit uint64

	storageService *storage.Service
	redisService   *redis.Service
}

func registerLogsHandler(r chi.Router, appConfig *config.AppConfig, storageService *storage.Service, redisService *redis.Service) {
	h := &logsHandler{
		singleFileLimit:    appConfig.SingleFileSizeLimit,
		maxFileCount:       appConfig.MaxFileCount,
		contentLengthLimit: appConfig.SingleFileSizeLimit * uint64(appConfig.MaxFileCount),
		storageService:     storageService,
		redisService:       redisService,
	}

	r.Route("/logs", func(r chi.Router) {
		r.Post("/", h.post)

		r.Route("/file/{logFileId}", func(r chi.Router) {
			r.Use(h.idCtx)
			r.Get("/", h.getFile)
		})

		r.Route("/bundle/{logBundleId}", func(r chi.Router) {
			r.Use(h.idCtx)
			r.Get("/", h.getBundle)
		})
	})
}

func (h *logsHandler) post(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength <= 0 {
		http.Error(w, fmt.Sprintf("Content-Length must be set to a positive non-zero value!"), http.StatusLengthRequired)
		return
	}

	if uint64(r.ContentLength) > h.contentLengthLimit {
		http.Error(w, fmt.Sprintf("Content-Length of %d is over the limit of %d bytes", r.ContentLength, h.contentLengthLimit), http.StatusRequestEntityTooLarge)
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		oplog := httplog.LogEntry(r.Context())
		oplog.Error("failed to parse multipart form", utils.ErrAttr(err))

		w.Header().Set("Accept-Post", "multipart/form-data")
		http.Error(w, "invalid multipart form", http.StatusUnsupportedMediaType)
		return
	}

	fileCount := 0
	logFileIds := make([]logs.LogFileId, h.maxFileCount)

	for {
		if uint16(fileCount) >= h.maxFileCount {
			http.Error(w, "you're not allowed to upload more than `%d` file(s)", http.StatusRequestEntityTooLarge)
			return
		}

		part, err := reader.NextRawPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			oplog := httplog.LogEntry(r.Context())
			oplog.Error("unexpected error, expected EOF", utils.ErrAttr(err))
			writeInternalServerError(w)
			return
		}

		logFileId := ulid.Make()
		logFileIds[fileCount] = logFileId
		fileCount += 1

		err = h.storageService.StageLogFile(logFileId, part, h.singleFileLimit)
		if err != nil {
			var fileTooLarge storage.FileTooLarge
			if errors.As(err, &fileTooLarge) {
				http.Error(w, fmt.Sprintf("`%d` bytes is over the single file limit of `%d` bytes", fileTooLarge.Actual, fileTooLarge.Actual), http.StatusRequestEntityTooLarge)
				return
			} else {
				oplog := httplog.LogEntry(r.Context())
				oplog.Error("unexpected error", utils.ErrAttr(err))
				writeInternalServerError(w)
				return
			}
		}

		err = h.redisService.StageLogFile(context.Background(), logFileId)
		if err != nil {
			writeInternalServerError(w)
			return
		}
	}

	logFileIds = logFileIds[:fileCount]
	logBundleId, err := h.redisService.CreateLogBundle(context.Background(), logFileIds)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	idString, err := logBundleId.MarshalText()
	if err != nil {
		oplog := httplog.LogEntry(r.Context())
		oplog.Error("unexpected error marshaling id to text", utils.ErrAttr(err))
		writeInternalServerError(w)
		return
	}

	_, err = w.Write(idString)
	if err != nil {
		oplog := httplog.LogEntry(r.Context())
		oplog.Error("unexpected error writing output", utils.ErrAttr(err))
		writeInternalServerError(w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *logsHandler) idCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var idInput string

		if logFileIdParam := chi.URLParam(r, "logFileId"); logFileIdParam != "" {
			idInput = logFileIdParam
		} else if logBundleIdParam := chi.URLParam(r, "logBundleId"); logBundleIdParam != "" {
			idInput = logBundleIdParam
		} else {
			http.NotFound(w, r)
			return
		}

		id, err := logs.ParseId(idInput)
		if err != nil {
			oplog := httplog.LogEntry(r.Context())
			oplog.Error("failed to parse id", slog.String("input", idInput))
			http.NotFound(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), "id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *logsHandler) getFile(w http.ResponseWriter, r *http.Request) {
	logFileId := r.Context().Value("id").(logs.LogFileId)

	// TODO: check with redis?

	file, err := h.storageService.OpenLogFile(logFileId)
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}

		writeInternalServerError(w)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, logFileId.String(), time.UnixMilli(0), file)
}

func (h *logsHandler) getBundle(w http.ResponseWriter, r *http.Request) {
	logBundleId := r.Context().Value("id").(logs.LogBundleId)

	logFileIds, err := h.redisService.GetLogBundle(r.Context(), logBundleId)
	if err != nil {
		if errors.Is(err, redis.ErrNotFound) {
			http.NotFound(w, r)
			return
		}

		oplog := httplog.LogEntry(r.Context())
		oplog.Error("unexpected error while getting log bundle from redis", slog.String("logBundleId", logBundleId.String()), utils.ErrAttr(err))
		writeInternalServerError(w)
		return
	}

	jsonBytes, err := json.Marshal(logFileIds)
	if err != nil {
		oplog := httplog.LogEntry(r.Context())
		oplog.Error("unexpected error while marshaling log file IDs of log bundle", slog.String("logBundleId", logBundleId.String()), utils.ErrAttr(err))
		writeInternalServerError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonBytes)
	w.WriteHeader(http.StatusOK)
}
