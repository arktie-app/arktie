package post

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/sequential"
	"github.com/bluesky-social/indigo/repo/carutil"
	"github.com/gorilla/websocket"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"

	"arktie.org/internal/data"
	"arktie.org/internal/data/client"
	"arktie.org/internal/lib/liblogs"
)

const collection = "app.arktie.post"

type FirehoseService struct {
	cfg   *data.Config
	event *client.Event
}

func NewFirehoseService(cfg *data.Config, event *client.Event) *FirehoseService {
	return &FirehoseService{cfg: cfg, event: event}
}

func (s *FirehoseService) Subscribe(ctx context.Context, conn *websocket.Conn) error {
	callbacks := &events.RepoStreamCallbacks{
		RepoCommit: func(evt *comatproto.SyncSubscribeRepos_Commit) error {
			return s.handleCommit(evt)
		},
		Error: func(evt *events.ErrorFrame) error {
			slog.Error("firehose error", slog.String("error", evt.Error), slog.String("message", evt.Message))
			return nil
		},
	}

	sched := sequential.NewScheduler("arktie-listener", callbacks.EventHandler)

	return events.HandleRepoStream(ctx, conn, sched, slog.Default())
}

func (s *FirehoseService) handleCommit(evt *comatproto.SyncSubscribeRepos_Commit) error {
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

			from, _ := record["from"].(string)
			if shouldIgnore(from, s.cfg.App.URL) {
				continue
			}

			topic := "firehose.post." + op.Action
			payload, _ := json.Marshal(record)

			msg := message.NewMessage("", payload)
			msg.Metadata.Set("repo", evt.Repo)
			msg.Metadata.Set("path", op.Path)

			if err := s.event.Publisher.Publish(topic, msg); err != nil {
				log.Error("failed to publish event", liblogs.ErrAttr(err))
			}

		case "delete":
			topic := "firehose.post.delete"
			msg := message.NewMessage("", []byte(op.Path))
			msg.Metadata.Set("repo", evt.Repo)
			msg.Metadata.Set("path", op.Path)

			if err := s.event.Publisher.Publish(topic, msg); err != nil {
				log.Error("failed to publish event", liblogs.ErrAttr(err))
			}
		}
	}

	return nil
}

func shouldIgnore(from string, appURL *url.URL) bool {
	if from == "" {
		return true
	}

	u, err := url.Parse(from)
	if err != nil {
		return false
	}

	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}

	if appURL != nil && from == appURL.String() {
		return true
	}

	return false
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
