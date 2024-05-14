package internal

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"io"
	"log/slog"
	"net/http"
	"time"
)

func loadRoutes() http.Handler {
	r := chi.NewRouter()

	// https://github.com/go-chi/httplog/blob/master/options.go
	logger := httplog.NewLogger("simple-log-store", httplog.Options{
		JSON:            true,
		LogLevel:        slog.LevelDebug,
		RequestHeaders:  true,
		TimeFieldFormat: time.RFC3339,
	})

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(httplog.RequestLogger(logger))
	r.Use(middleware.Recoverer)

	r.Use(middleware.Heartbeat("/ping"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	r.Route("/logs", func(r chi.Router) {
		r.Post("/", uploadLogs)
	})

	return r
}

var expectedPrefix = []byte("############ Nexus Mods App log file")

func uploadLogs(w http.ResponseWriter, r *http.Request) {

	// TODO: move values to config
	const singleFileLimit int64 = 2 << 19
	const maxFileCount int64 = 5
	const contentLengthLimit = singleFileLimit * maxFileCount

	if r.ContentLength > contentLengthLimit {
		http.Error(w, fmt.Sprintf("%d is over the limit of %d bytes", r.ContentLength, contentLengthLimit), http.StatusRequestEntityTooLarge)
		return
	}

	if r.ContentLength <= 0 {
		http.Error(w, fmt.Sprintf("Content-Length must be set to a positive non-zero value!"), http.StatusLengthRequired)
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		oplog := httplog.LogEntry(r.Context())
		oplog.Error("failed to parse multipart form", httplog.ErrAttr(err))

		w.Header().Set("Accept-Post", "multipart/form-data")
		http.Error(w, "invalid multipart form", http.StatusUnsupportedMediaType)
		return
	}

	// TODO: can probably be optimized
	buffer := make([]byte, singleFileLimit)
	eofCheck := make([]byte, 1)

	for {
		part, err := reader.NextRawPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			oplog := httplog.LogEntry(r.Context())
			oplog.Error("expected EOF", httplog.ErrAttr(err))
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		fileName := part.FileName()

		oplog := httplog.LogEntry(r.Context())
		oplog.With(slog.Group("details", slog.String("file name", fileName))).Info("multiform part info")

		bytesRead, err := part.Read(buffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				oplog := httplog.LogEntry(r.Context())
				oplog.Error("error while reading form part", httplog.ErrAttr(err))
				http.Error(w, "something went wrong", http.StatusInternalServerError)
				return
			}
		}

		if bytesRead == len(buffer) {
			_, err = part.Read(eofCheck)
			if !errors.Is(err, io.EOF) {
				oplog := httplog.LogEntry(r.Context())
				oplog.Error("expected EOF, client send more than the allowed max")
				http.Error(w, fmt.Sprintf("the max file size allowed is %d bytes", singleFileLimit), http.StatusRequestEntityTooLarge)
				return
			}
		}

		contentSlice := buffer[:bytesRead]
		if !bytes.HasPrefix(contentSlice, expectedPrefix) {
			oplog := httplog.LogEntry(r.Context())
			oplog.Error("uploaded file isn't a log file after checking the header")
			http.Error(w, "missing header", http.StatusUnsupportedMediaType)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
