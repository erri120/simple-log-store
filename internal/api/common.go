package api

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
	"log/slog"
	"net/http"
	"simple-log-store/internal/logs"
)

func writeInternalServerError(w http.ResponseWriter) {
	http.Error(w, "something went wrong", http.StatusInternalServerError)
}

func idCtx(next http.Handler) http.Handler {
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
