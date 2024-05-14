package internal

import (
	"context"
	"github.com/go-chi/httplog/v2"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"net/http"
	"time"
)

type App struct {
	Router      http.Handler
	RedisClient *redis.Client
}

func New() *App {
	app := &App{
		Router: loadRoutes(),
	}

	return app
}

func (app *App) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    ":3000",
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

	slog.Info("Starting server")

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
