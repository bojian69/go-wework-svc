package main

import (
	"flag"
	"log/slog"
	"os"

	"go-wework-svc/internal/bootstrap"
	"go-wework-svc/internal/shared"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := shared.LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	app, err := bootstrap.NewApp(cfg)
	if err != nil {
		slog.Error("failed to init app", "error", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		slog.Error("app exited with error", "error", err)
		os.Exit(1)
	}
}
