package main

import (
	"log/slog"
	"net/http"
	"simple-log-store/internal"
)

func main() {
	slog.Info("starting")
	app := internal.New()

	err := http.ListenAndServe(":3000", app.Router)
	slog.Error("exit", err)
}
