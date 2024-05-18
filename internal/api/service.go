package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"io"
	"log/slog"
	"net/http"
	"simple-log-store/internal/config"
	"simple-log-store/internal/redis"
	"simple-log-store/internal/storage"
	"time"
)

type Service struct {
	Handler http.Handler
}

func CreateService(appConfig *config.AppConfig, storageService *storage.Service, redisService *redis.Service, logWriter io.Writer) *Service {
	r := chi.NewRouter()
	service := &Service{
		Handler: r,
	}

	// https://github.com/go-chi/httplog/blob/master/options.go
	requestLogger := httplog.NewLogger("api", httplog.Options{
		JSON:             true,
		LogLevel:         slog.LevelDebug,
		RequestHeaders:   true,
		MessageFieldName: "msg",
		TimeFieldName:    "time",
		TimeFieldFormat:  time.RFC3339Nano,
		Writer:           logWriter,
	})

	r.Use(middleware.RequestID)
	// included in RequestLogger: r.Use(middleware.RealIP)
	r.Use(httplog.RequestLogger(requestLogger))
	//included in RequestLogger: r.Use(middleware.Recoverer)

	r.Use(middleware.Heartbeat("/ping"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	registerLogsHandler(r, appConfig, storageService, redisService)

	return service
}
