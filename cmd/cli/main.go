package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"arktie.org/internal/data"
	"arktie.org/internal/lib/liblogs"
)

// Name is the name of the application.
var Name string

// Version is the version of the application.
var Version string

func main() {
	rootCmd := &cobra.Command{
		Use:     "arktie-cli",
		Short:   "Arktie CLI tools",
		Version: Version,
	}

	var cfgPath string
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "configs/", "config dir path")

	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

			cfg, err := data.NewConfig(Name, Version, cfgPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			client, err := data.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to init data client: %w", err)
			}

			slog.Info("running database migration")
			if err := client.Ent.Schema.Create(cmd.Context()); err != nil {
				return fmt.Errorf("failed to run migration: %w", err)
			}

			slog.Info("database migration completed")
			return nil
		},
	}

	rootCmd.AddCommand(migrateCmd)

	if err := rootCmd.Execute(); err != nil {
		slog.Error("command failed", liblogs.ErrAttr(err))
		os.Exit(1)
	}
}
