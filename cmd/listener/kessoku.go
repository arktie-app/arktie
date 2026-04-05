package main

//go:generate go tool kessoku $GOFILE

import (
	"github.com/mazrean/kessoku"

	"arktie.org/internal/data"
	dataclient "arktie.org/internal/data/client"
	"arktie.org/internal/server"
)

//nolint:unused
var _ = kessoku.Inject[*App](
	"newApp",
	kessoku.Provide(dataclient.NewEvent),

	kessoku.Provide(func() *FirehoseLogHandler { return &FirehoseLogHandler{} }),
	kessoku.Bind[server.FirehosePostHandler](
		kessoku.Provide(func(h *FirehoseLogHandler) server.FirehosePostHandler { return h }),
	),

	kessoku.Provide(server.NewFirehoseListener),

	kessoku.Provide(func(cfg *data.Config, event *dataclient.Event, _ *server.FirehoseListener) *App {
		return &App{
			Config: cfg,
			Event:  event,
		}
	}),
)
