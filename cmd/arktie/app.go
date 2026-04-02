package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ThreeDotsLabs/watermill/message"
	"golang.org/x/sync/errgroup"
)

type App struct {
	HTTP   *http.Server
	Router *message.Router
}

func (a *App) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("starting watermill router")
		return a.Router.Run(ctx)
	})

	g.Go(func() error {
		slog.Info("starting http server", slog.String("addr", a.HTTP.Addr))
		return a.HTTP.ListenAndServe()
	})

	return g.Wait()
}
