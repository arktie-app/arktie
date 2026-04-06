package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"

	"arktie.org/internal/data"
	"arktie.org/internal/data/client"
	"arktie.org/internal/lib/liblogs"
	"arktie.org/internal/service/post"
)

type App struct {
	Config   *data.Config
	Event    *client.Event
	Firehose *post.FirehoseService
}

func (a *App) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("starting watermill router")
		return a.Event.Router.Run(ctx)
	})

	g.Go(func() error {
		return a.subscribeFirehose(ctx)
	})

	<-ctx.Done()
	return g.Wait()
}

func (a *App) subscribeFirehose(ctx context.Context) error {
	const (
		baseDelay = 1 * time.Second
		maxDelay  = 60 * time.Second
	)

	var attempt int
	for {
		err := a.connectAndListen(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		attempt++

		slog.Error("firehose disconnected, reconnecting",
			liblogs.ErrAttr(err),
			slog.Duration("backoff", delay),
			slog.Int("attempt", attempt),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

func (a *App) connectAndListen(ctx context.Context) error {
	wsURL := a.Config.Service.Firehose.RelayURL.JoinPath("/xrpc/com.atproto.sync.subscribeRepos")

	slog.Info("connecting to firehose", slog.String("url", wsURL.String()))

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL.String(), http.Header{})
	if err != nil {
		return fmt.Errorf("failed to connect to firehose: %w", err)
	}
	defer conn.Close()

	slog.Info("connected to firehose, listening for app.arktie.post events")

	return a.Firehose.Subscribe(ctx, conn)
}
