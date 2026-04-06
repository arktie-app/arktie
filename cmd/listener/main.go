package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"arktie.org/internal/data"
	"arktie.org/internal/lib/liblogs"
)

// Name is the name of the application.
var Name string

// Version is the version of the application.
var Version string

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

	cfg, err := data.NewConfig(Name, Version, cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	app, err := newApp(cfg)
	if err != nil {
		return fmt.Errorf("failed to init app: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return app.Run(ctx)
}
