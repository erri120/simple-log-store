package api

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
	"github.com/oklog/ulid/v2"
	"io"
	"net/http"
	"simple-log-store/internal/config"
	"simple-log-store/internal/logs"
	"simple-log-store/internal/storage"
	"simple-log-store/internal/utils"
)

// Handler for the `/logs` endpoint.
type logsHandler struct {
	singleFileLimit    uint64
	maxFileCount       uint16
	contentLengthLimit uint64

	storage *storage.Service
}

func registerLogsHandler(r chi.Router, appConfig *config.AppConfig, storage *storage.Service) {
	h := &logsHandler{
		singleFileLimit:    appConfig.SingleFileSizeLimit,
		maxFileCount:       appConfig.MaxFileCount,
		contentLengthLimit: appConfig.SingleFileSizeLimit * uint64(appConfig.MaxFileCount),
		storage:            storage,
	}

	r.Route("/logs", func(r chi.Router) {
		r.Post("/", h.post)
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
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		logFileId := ulid.Make()
		logFileIds[fileCount] = logFileId
		fileCount += 1

		err = h.storage.StageLogFile(logFileId, part, h.singleFileLimit)
		if err != nil {
			var fileTooLarge storage.FileTooLarge
			if errors.As(err, &fileTooLarge) {
				http.Error(w, fmt.Sprintf("`%d` bytes is over the single file limit of `%d` bytes", fileTooLarge.Actual, fileTooLarge.Actual), http.StatusRequestEntityTooLarge)
				return
			} else {
				oplog := httplog.LogEntry(r.Context())
				oplog.Error("unexpected error", utils.ErrAttr(err))
				http.Error(w, "something went wrong", http.StatusInternalServerError)
				return
			}
		}
	}

	logFileIds = logFileIds[:fileCount]
	// TODO: make log bundle

	w.WriteHeader(http.StatusOK)
}
