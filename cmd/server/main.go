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
	app := internal.New()

	ctx, cancelFunc := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFunc()

	err := app.Start(ctx)
	if err != nil {
		slog.Error("error starting", httplog.ErrAttr(err))
	}
}
