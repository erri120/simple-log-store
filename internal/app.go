package internal

import (
	"context"
	"errors"
	"fmt"
	"github.com/sethvargo/go-envconfig"
	"io"
	"log/slog"
	"net/http"
	"simple-log-store/internal/api"
	"simple-log-store/internal/config"
	"simple-log-store/internal/redis"
	"simple-log-store/internal/storage"
	"simple-log-store/internal/utils"
	"time"
)

type App struct {
	Logger *slog.Logger

	Config         *config.AppConfig
	StorageService *storage.Service
	RedisService   *redis.Service
	ApiService     *api.Service
}

func New(logger *slog.Logger, logWriter io.Writer) (*App, error) {
	var appConfig config.AppConfig
	err := envconfig.Process(context.Background(), &appConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	storageService, err := storage.CreateService(&appConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}

	redisService, err := redis.CreateService(&appConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis service: %w", err)
	}

	apiService := api.CreateService(&appConfig, storageService, logWriter)

	app := &App{
		Logger:         logger,
		Config:         &appConfig,
		StorageService: storageService,
		RedisService:   redisService,
		ApiService:     apiService,
	}

	return app, nil
}

func (app *App) Start(ctx context.Context) error {
	port := app.Config.Port
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: app.ApiService.Handler,
	}

	if err := app.RedisService.Ping(); err != nil {
		return err
	}

	defer func(redisService *redis.Service) {
		redisService.Close()
	}(app.RedisService)

	app.Logger.Info("starting server", slog.Uint64("port", uint64(port)))

	go func(server *http.Server, logger *slog.Logger) {
		err := server.ListenAndServe()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				logger.Info("server closed")
			} else {
				slog.Error("server shutdown unexpectedly", utils.ErrAttr(err))
			}
		}
	}(server, app.Logger)

	select {
	case <-ctx.Done():
		app.Logger.Info("shutting down server")

		timeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		return server.Shutdown(timeout)
	}
}
