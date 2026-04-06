package main

//go:generate go tool kessoku $GOFILE

import (
	"github.com/mazrean/kessoku"

	"arktie.org/internal/data"
	dataclient "arktie.org/internal/data/client"
	"arktie.org/internal/server"
	"arktie.org/internal/service/post"
	ucpost "arktie.org/internal/usecase/post"
)

//nolint:unused
var _ = kessoku.Inject[*App](
	"newApp",
	kessoku.Provide(dataclient.NewDatabase),
	kessoku.Provide(dataclient.NewEvent),
	kessoku.Provide(dataclient.NewOAuth),

	kessoku.Provide(ucpost.NewPDSRecord),
	kessoku.Bind[post.PDSRecordAPI](
		kessoku.Provide(func(p *ucpost.PDSRecord) post.PDSRecordAPI { return p }),
	),

	kessoku.Provide(post.NewFirehoseService),
	kessoku.Provide(post.NewSyncService),
	kessoku.Bind[server.PostPuller](
		kessoku.Provide(func(s *post.SyncService) server.PostPuller { return s }),
	),

	kessoku.Provide(server.NewFirehoseListener),

	kessoku.Provide(func(cfg *data.Config, event *dataclient.Event, firehose *post.FirehoseService, _ *server.FirehoseListener) *App {
		return &App{
			Config:   cfg,
			Event:    event,
			Firehose: firehose,
		}
	}),
)
