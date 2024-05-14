package internal

import (
	"context"
	"fmt"
	"github.com/go-chi/httplog/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sethvargo/go-envconfig"
	"log/slog"
	"net/http"
	"time"
)

type App struct {
	Config      AppConfig
	Router      http.Handler
	RedisClient *redis.Client
}

func New() (*App, error) {
	var config AppConfig
	err := envconfig.Process(context.Background(), &config)
	if err != nil {
		slog.Error("Failed to parse environment variables", httplog.ErrAttr(err))
		return nil, err
	}

	app := &App{
		Config:      config,
		RedisClient: nil,
	}

	app.loadRoutes()

	return app, nil
}

func (app *App) Start(ctx context.Context) error {
	port := app.Config.Port
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: app.Router,
	}

	if app.RedisClient != nil {
		err := app.RedisClient.Ping(ctx).Err()
		if err != nil {
			slog.Error("failed to connect to redis", httplog.ErrAttr(err))
			return err
		}

		defer func(redisClient *redis.Client) {
			if err := redisClient.Close(); err != nil {
				slog.Error("failed to close redis client", httplog.ErrAttr(err))
			}
		}(app.RedisClient)
	}

	slog.Info(fmt.Sprintf("Starting server with port %d", port))

	go func(server *http.Server) {
		err := server.ListenAndServe()
		if err != nil {
			slog.Error("failed to start server", httplog.ErrAttr(err))
		}
	}(server)

	select {
	case <-ctx.Done():
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		return server.Shutdown(timeout)
	}
}
