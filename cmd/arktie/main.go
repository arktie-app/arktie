package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"arktie.org/internal/data"
	"arktie.org/internal/lib/liblogs"
)

// Name is the name of the application.
var Name string

// Version is the version of the application.
var Version string

var (
	flagCfgPath string
)

func init() {
	flag.StringVar(&flagCfgPath, "config", "configs/", "config dir path")
}

func main() {
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	cfg, err := data.NewConfig(Name, Version, flagCfgPath)
	if err != nil {
		slog.Error("failed to load config", liblogs.ErrAttr(err))
		return
	}

	app, err := newApp(cfg)
	if err != nil {
		slog.Error("failed to init app", liblogs.ErrAttr(err))
		return
	}

	if err := app.Run(context.Background()); err != nil {
		slog.Error("app stopped", liblogs.ErrAttr(err))
	}
}
