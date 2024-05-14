package internal

import "net/http"

type App struct {
	Router http.Handler
}

func New() *App {
	app := &App{
		Router: loadRoutes(),
	}

	return app
}
