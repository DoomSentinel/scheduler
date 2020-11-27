package services

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/monitoring"
	"github.com/DoomSentinel/scheduler/scheduler"
	"github.com/DoomSentinel/scheduler/services/grpc"
)

var Module = fx.Options(
	grpc.Module,
	scheduler.Module,
	fx.Invoke(grpc.ServeGRPC),
)

type ApplicationParams struct {
	fx.In

	Lc        fx.Lifecycle
	Scheduler *scheduler.Scheduler
	Monitor   monitoring.Monitor
	Log       *zap.Logger
}

func Run(in ApplicationParams) {
	err := in.Scheduler.Start()
	if err != nil {
		in.Log.Fatal("failed to start scheduler", zap.Error(err))
	}

	in.Lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			err := in.Monitor.Error()
			if err != nil {
				return err
			}
			in.Monitor.StartDetached(in.Log)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			_ = in.Monitor.GracefulStop()
			return in.Scheduler.Shutdown()
		},
	})
}
