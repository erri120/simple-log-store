package main

import (
	"context"
	"github.com/go-chi/httplog/v2"
	"log/slog"
	"os"
	"os/signal"
	"simple-log-store/internal"
)

func main() {
	app, err := internal.New()
	if err != nil {
		slog.Error("failed to create app", httplog.ErrAttr(err))
		return
	}

	ctx, cancelFunc := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFunc()

	err = app.Start(ctx)
	if err != nil {
		slog.Error("error starting", httplog.ErrAttr(err))
	}
}
