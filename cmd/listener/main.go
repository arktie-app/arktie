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
	"os"
	"os/signal"
	"strings"
	"syscall"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/sequential"
	"github.com/bluesky-social/indigo/repo/carutil"
	"github.com/gorilla/websocket"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/spf13/cobra"

	"arktie.org/internal/data"
	"arktie.org/internal/lib/liblogs"
)

const collection = "app.arktie.post"

var cfgPath string

func main() {
	rootCmd := &cobra.Command{
		Use:   "listener",
		Short: "ATProto Firehose listener for syncing posts from other Arktie instances",
		RunE:  run,
	}

	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "configs/", "config dir path")

	if err := rootCmd.Execute(); err != nil {
		slog.Error("command failed", liblogs.ErrAttr(err))
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	cfg, err := data.NewConfig("", "", cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return subscribe(ctx, cfg)
}

func subscribe(ctx context.Context, cfg *data.Config) error {
	wsURL := cfg.Service.Firehose.RelayURL.JoinPath("/xrpc/com.atproto.sync.subscribeRepos")

	slog.Info("connecting to firehose", slog.String("url", wsURL.String()))

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL.String(), http.Header{})
	if err != nil {
		return fmt.Errorf("failed to connect to firehose: %w", err)
	}
	defer conn.Close()

	slog.Info("connected to firehose, listening for app.arktie.post events")

	callbacks := &events.RepoStreamCallbacks{
		RepoCommit: func(evt *comatproto.SyncSubscribeRepos_Commit) error {
			return handleCommit(evt)
		},
		Error: func(evt *events.ErrorFrame) error {
			slog.Error("firehose error", slog.String("error", evt.Error), slog.String("message", evt.Message))
			return nil
		},
	}

	sched := sequential.NewScheduler("arktie-listener", callbacks.EventHandler)

	return events.HandleRepoStream(ctx, conn, sched, slog.Default())
}

func handleCommit(evt *comatproto.SyncSubscribeRepos_Commit) error {
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

			jsonBytes, _ := json.MarshalIndent(record, "", "  ")
			log.Info("received post", slog.String("payload", string(jsonBytes)))

		case "delete":
			log.Info("received post deletion")
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
