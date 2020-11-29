package v1

import (
	schedulerV1 "github.com/DoomSentinel/scheduler-api/gen/go/v1"
	"go.uber.org/fx"
	"google.golang.org/grpc"

	"github.com/DoomSentinel/scheduler/services/grpc/v1/schedulerService"
)

var Module = fx.Options(
	schedulerService.Module,
	fx.Invoke(RegisterGRPCServices),
)

type Services struct {
	fx.In

	SchedulerService schedulerV1.SchedulerServiceServer

	Server *grpc.Server `name:"server-grpc"`
}

func RegisterGRPCServices(params Services) {
	schedulerV1.RegisterSchedulerServiceServer(params.Server, params.SchedulerService)
}
