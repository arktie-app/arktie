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
	"arktie.org/internal/data/client"
	"arktie.org/internal/lib/liblogs"
)

// Name is the name of the application.
var Name string

// Version is the version of the application.
var Version string

var cfgPath string

func main() {
	rootCmd := &cobra.Command{
		Use:     "arktie",
		Short:   "Arktie - community platform for artists",
		Version: Version,
	}

	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "configs/", "config dir path")

	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(databaseCmd())

	if err := rootCmd.Execute(); err != nil {
		slog.Error("command failed", liblogs.ErrAttr(err))
		os.Exit(1)
	}
}

func loadConfig() (*data.Config, error) {
	cfg, err := data.NewConfig(Name, Version, cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			app, err := newApp(cfg)
			if err != nil {
				return fmt.Errorf("failed to init app: %w", err)
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			return app.Run(ctx)
		},
	}
}

func databaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Database management commands",
	}

	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			db, err := client.NewDatabase(cfg)
			if err != nil {
				return fmt.Errorf("failed to init database client: %w", err)
			}

			slog.Info("running database migration")
			if err := db.Ent.Schema.Create(cmd.Context()); err != nil {
				return fmt.Errorf("failed to run migration: %w", err)
			}

			slog.Info("database migration completed")
			return nil
		},
	}

	cmd.AddCommand(migrateCmd)
	return cmd
}
