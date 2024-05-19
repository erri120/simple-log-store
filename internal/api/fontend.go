package api

import (
	"errors"
	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
	"log/slog"
	"net/http"
	"simple-log-store/internal/logs"
	"simple-log-store/internal/redis"
	"simple-log-store/internal/storage"
	"simple-log-store/internal/utils"
	"simple-log-store/internal/views"
)

type frontendHandler struct {
	storageService *storage.Service
	redisService   *redis.Service
}

func registerFrontendHandler(r chi.Router, storageService *storage.Service, redisService *redis.Service) {
	h := &frontendHandler{
		storageService: storageService,
		redisService:   redisService,
	}

	r.Route("/view", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})

		r.Route("/bundle/{logBundleId}", func(r chi.Router) {
			r.Use(idCtx)
			r.Get("/", h.viewBundle)
		})
	})
}

func (h *frontendHandler) render(component templ.Component, w http.ResponseWriter, r *http.Request) {
	err := component.Render(r.Context(), w)
	if err != nil {
		oplog := httplog.LogEntry(r.Context())
		oplog.Error("error rendering templ component", utils.ErrAttr(err))
		writeInternalServerError(w)
	}
}

func (h *frontendHandler) viewBundle(w http.ResponseWriter, r *http.Request) {
	logBundleId := r.Context().Value("id").(logs.LogBundleId)

	logFileIds, err := h.redisService.GetLogBundle(r.Context(), logBundleId)
	if err != nil {
		if errors.Is(err, redis.ErrNotFound) {
			h.render(views.NotFound(logBundleId), w, r)
			return
		}

		oplog := httplog.LogEntry(r.Context())
		oplog.Error("unexpected error while getting log bundle from redis", slog.String("logBundleId", logBundleId.String()), utils.ErrAttr(err))
		writeInternalServerError(w)
		return
	}

	h.render(views.Bundle(logBundleId, logFileIds), w, r)
}
