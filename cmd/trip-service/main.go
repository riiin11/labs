package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/riiin11/labs/internal/config"
	"github.com/riiin11/labs/internal/container"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := container.New(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize app: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	fmt.Println("Starting server on", cfg.HTTPAddr)
	if err := app.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "server stopped with error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server stopped gracefully")
}