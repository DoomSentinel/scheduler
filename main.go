package main

import (
	"context"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/bus"
	"github.com/DoomSentinel/scheduler/config"
	"github.com/DoomSentinel/scheduler/monitoring"
	"github.com/DoomSentinel/scheduler/services"
)

func main() {
	var logger *zap.Logger
	c := make(chan struct{}, 1)
	defer func() {
		if monitoring.LoggerValid(logger) {
			_ = logger.Sync()
		}
	}()
	ctx := context.Background()
	app := fx.New(
		config.Module,
		monitoring.Module,
		fx.Populate(&logger),
		bus.Module,
		services.Module,
		fx.Invoke(services.Run),
	)
	err := app.Start(ctx)
	if err != nil {
		if monitoring.LoggerValid(logger) {
			logger.Error("unable to start application", zap.Error(err))
		}

		c <- struct{}{}
	}

	select {
	case <-app.Done():
		if monitoring.LoggerValid(logger) {
			_ = logger.Sync()
		}
	case <-c:
		os.Exit(1)
	}

	err = app.Stop(context.Background())
	if err != nil {
		logger.Fatal("unable to stop application", zap.Error(err))
	}
}
