package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/sequential"
	"github.com/bluesky-social/indigo/repo/carutil"
	"github.com/gorilla/websocket"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"golang.org/x/sync/errgroup"

	"arktie.org/internal/data"
	"arktie.org/internal/data/client"
	"arktie.org/internal/lib/liblogs"
)

const collection = "app.arktie.post"

type App struct {
	Config *data.Config
	Event  *client.Event
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
	wsURL := a.Config.Service.Firehose.RelayURL.JoinPath("/xrpc/com.atproto.sync.subscribeRepos")

	slog.Info("connecting to firehose", slog.String("url", wsURL.String()))

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL.String(), http.Header{})
	if err != nil {
		return fmt.Errorf("failed to connect to firehose: %w", err)
	}
	defer conn.Close()

	slog.Info("connected to firehose, listening for app.arktie.post events")

	callbacks := &events.RepoStreamCallbacks{
		RepoCommit: func(evt *comatproto.SyncSubscribeRepos_Commit) error {
			return a.handleCommit(evt)
		},
		Error: func(evt *events.ErrorFrame) error {
			slog.Error("firehose error", slog.String("error", evt.Error), slog.String("message", evt.Message))
			return nil
		},
	}

	sched := sequential.NewScheduler("arktie-listener", callbacks.EventHandler)

	return events.HandleRepoStream(ctx, conn, sched, slog.Default())
}

func (a *App) handleCommit(evt *comatproto.SyncSubscribeRepos_Commit) error {
	for _, op := range evt.Ops {
		if !strings.HasPrefix(op.Path, collection+"/") {
			continue
		}

		log := slog.With(
			slog.String("repo", evt.Repo),
			slog.String("action", op.Action),
			slog.String("path", op.Path),
		)

		switch op.Action {
		case "create", "update":
			record, err := extractRecord(evt, op)
			if err != nil {
				log.Error("failed to extract record", liblogs.ErrAttr(err))
				continue
			}
			if record == nil {
				continue
			}

			topic := "firehose.post." + op.Action
			payload, _ := json.Marshal(record)

			msg := message.NewMessage("", payload)
			msg.Metadata.Set("repo", evt.Repo)
			msg.Metadata.Set("path", op.Path)

			if err := a.Event.Publisher.Publish(topic, msg); err != nil {
				log.Error("failed to publish event", liblogs.ErrAttr(err))
			}

		case "delete":
			topic := "firehose.post.delete"
			msg := message.NewMessage("", []byte(op.Path))
			msg.Metadata.Set("repo", evt.Repo)
			msg.Metadata.Set("path", op.Path)

			if err := a.Event.Publisher.Publish(topic, msg); err != nil {
				log.Error("failed to publish event", liblogs.ErrAttr(err))
			}
		}
	}

	return nil
}

func extractRecord(evt *comatproto.SyncSubscribeRepos_Commit, op *comatproto.SyncSubscribeRepos_RepoOp) (map[string]any, error) {
	if len(evt.Blocks) == 0 || op.Cid == nil {
		return nil, nil
	}

	target := cid.Cid(*op.Cid)

	reader, _, err := carutil.NewReader(bufio.NewReader(bytes.NewReader(evt.Blocks)))
	if err != nil {
		return nil, fmt.Errorf("failed to open CAR reader: %w", err)
	}

	for {
		blk, err := reader.NextBlock()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read CAR block: %w", err)
		}

		if blk.Cid() != target {
			continue
		}

		var rec map[string]any
		if err := cbornode.DecodeInto(blk.RawData(), &rec); err != nil {
			return nil, fmt.Errorf("failed to decode CBOR record: %w", err)
		}

		return rec, nil
	}

	return nil, nil
}
