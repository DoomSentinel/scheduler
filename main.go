package main

import (
	"context"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/backends"
	"github.com/DoomSentinel/scheduler/config"
	"github.com/DoomSentinel/scheduler/monitoring"
	"github.com/DoomSentinel/scheduler/services"
)

var (
	logger *zap.Logger
	app    = fx.Options(
		config.Module,
		monitoring.Module,
		fx.Populate(&logger),
		backends.Module,
		services.Module,
		fx.Invoke(services.Run),
	)
)

func main() {
	defer func() {
		if monitoring.LoggerValid(logger) {
			_ = logger.Sync()
		}
	}()
	ctx := context.Background()
	app := fx.New(
		app,
	)
	err := app.Start(ctx)
	if err != nil {
		if monitoring.LoggerValid(logger) {
			logger.Error("unable to start application", zap.Error(err))
			_ = logger.Sync()
		}

		os.Exit(1)
	}

	<-app.Done()
	if monitoring.LoggerValid(logger) {
		_ = logger.Sync()
	}

	err = app.Stop(context.Background())
	if err != nil {
		logger.Fatal("unable to stop application", zap.Error(err))
	}
}
