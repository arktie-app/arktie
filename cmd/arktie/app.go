package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"golang.org/x/sync/errgroup"
)

type App struct {
	HTTP   *http.Server
	Router *message.Router
}

func (a *App) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("starting watermill router")
		return a.Router.Run(ctx)
	})

	g.Go(func() error {
		slog.Info("starting http server", slog.String("addr", a.HTTP.Addr))
		if err := a.HTTP.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// Listen for the interrupt signal
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := a.HTTP.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return g.Wait()
}
