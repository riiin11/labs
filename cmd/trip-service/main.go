package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/you/trip-service/internal/config"
	"github.com/you/trip-service/internal/container"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := container.New(ctx, cfg)
	if err != nil {
		slog.Error("container", "err", err)
		os.Exit(1)
	}
	defer app.Close()

	if err := app.Run(ctx); err != nil {
		slog.Error("run", "err", err)
		os.Exit(1)
	}
}