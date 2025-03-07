package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"simple-log-store/internal"
	"simple-log-store/internal/utils"
)

func main() {
	defaultWriter := os.Stdout
	defaultLogger := slog.New(slog.NewJSONHandler(defaultWriter, nil)).With(slog.String("service", "app"))
	slog.SetDefault(defaultLogger)

	app, err := internal.New(defaultLogger, defaultWriter)
	if err != nil {
		slog.Error("failed to create app", utils.ErrAttr(err))
		return
	}

	ctx, cancelFunc := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFunc()

	err = app.Start(ctx)
	if err != nil {
		slog.Error("error starting", utils.ErrAttr(err))
	}
}
