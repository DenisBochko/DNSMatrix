package main

import (
	"context"
	"hackathon-agent/internal/app"
	"hackathon-agent/internal/config"
	"hackathon-agent/pkg/logger"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoadConfig()
	config.MustPrintConfig(cfg)

	loggerCfg := &logger.Config{
		Level:      "debug",
		FormatJSON: false,
		Rotation: logger.Rotation{
			File:       "./agent.log",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     7,
		},
	}

	log := logger.MustSetupLogger(loggerCfg)

	errors := make(chan error)

	application := app.MustNew(cfg, log)

	defer func() {
		close(errors)
	}()

	defer func() {
		if err := application.Shutdown(); err != nil {
			log.Error("Failed to shutdown application", zap.Error(err))
		}

		if err := log.Sync(); err != nil {
			log.Warn("Failed to sync logger", zap.Error(err))
		}

		log.Info("Application has shutdown")
	}()

	go func() { errors <- application.Run(ctx) }()

	select {
	case err := <-errors:
		if err != nil {
			log.Error("Server error, shutting down...", zap.Error(err))
		}
	case <-ctx.Done():
		log.Info("Received stop signal, shutting down...")
	}
}
