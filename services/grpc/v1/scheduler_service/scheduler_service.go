package scheduler_service

import (
	"context"
	"net/http"

	schedulerV1 "github.com/DoomSentinel/scheduler-api/gen/go/v1"
	"go.uber.org/fx"

	"github.com/DoomSentinel/scheduler/scheduler"
	"github.com/DoomSentinel/scheduler/scheduler/types"
)

var Module = fx.Provide(
	func(scheduler *scheduler.Scheduler) Scheduler {
		return scheduler
	},
	NewValidator,
	NewBaseService,
	NewValidatedService,
	NewService,
)

type (
	Scheduler interface {
		ScheduleTask(taskType types.TaskType, task *types.Task) error
		ReceiveNotifications(ctx context.Context) (<-chan *types.ExecutionInfo, error)
	}
	BaseService interface {
		schedulerV1.SchedulerServiceServer
	}
	schedulerService struct {
		scheduler Scheduler
	}
)

func NewService(service ValidatedService) schedulerV1.SchedulerServiceServer {
	return service
}

func NewBaseService(scheduler Scheduler) BaseService {
	return &schedulerService{
		scheduler: scheduler,
	}
}

func (s schedulerService) ScheduleRemoteTask(_ context.Context, r *schedulerV1.ScheduleRemoteTaskRequest) (*schedulerV1.ScheduleRemoteTaskResponse, error) {
	err := s.scheduler.ScheduleTask(types.TaskTypeRemote, makeTaskFromRemoteTaskRequest(r))
	if err != nil {
		return nil, err
	}

	return new(schedulerV1.ScheduleRemoteTaskResponse), nil
}

func (s schedulerService) ScheduleDummyTask(_ context.Context, r *schedulerV1.ScheduleDummyTaskRequest) (*schedulerV1.ScheduleDummyTaskResponse, error) {
	err := s.scheduler.ScheduleTask(types.TaskTypeDummy, makeTaskFromDummyTaskRequest(r))
	if err != nil {
		return nil, err
	}

	return new(schedulerV1.ScheduleDummyTaskResponse), nil
}

func (s schedulerService) ScheduleCommandTask(_ context.Context, r *schedulerV1.ScheduleCommandTaskRequest) (*schedulerV1.ScheduleCommandTaskResponse, error) {
	err := s.scheduler.ScheduleTask(types.TaskTypeCommand, makeTaskFromCommandTaskRequest(r))
	if err != nil {
		return nil, err
	}

	return new(schedulerV1.ScheduleCommandTaskResponse), nil
}

func (s schedulerService) ExecutionNotifications(_ *schedulerV1.ExecutionNotificationsRequest, server schedulerV1.SchedulerService_ExecutionNotificationsServer) error {
	receiver, err := s.scheduler.ReceiveNotifications(server.Context())
	if err != nil {
		return err
	}
	for info := range receiver {
		status := schedulerV1.ExecutionStatus_EXECUTION_STATUS_INVALID
		switch info.Status {
		case types.ExecutionStatusSuccess:
			status = schedulerV1.ExecutionStatus_EXECUTION_STATUS_SUCCESS
		case types.ExecutionStatusFailed:
			status = schedulerV1.ExecutionStatus_EXECUTION_STATUS_FAILED
		}

		err := server.Send(&schedulerV1.ExecutionNotificationsResponse{
			TaskId:   info.Id,
			Status:   status,
			Duration: info.Duration,
			Output:   info.Output,
			Message:  info.Message,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func makeTaskFromCommandTaskRequest(r *schedulerV1.ScheduleCommandTaskRequest) *types.Task {
	return &types.Task{
		Command: &types.LocalCommand{
			Command:   r.GetConfig().GetCommand(),
			Arguments: r.GetConfig().GetArguments(),
			Timeout:   r.GetConfig().GetTimeout(),
		},
		ID:                  r.GetId(),
		Retries:             0,
		ScheduleOnTimestamp: r.GetScheduleTime().GetSeconds(),
	}
}

func makeTaskFromDummyTaskRequest(r *schedulerV1.ScheduleDummyTaskRequest) *types.Task {
	return &types.Task{
		ID:                  r.GetId(),
		Retries:             0,
		ScheduleOnTimestamp: r.GetScheduleTime().GetSeconds(),
	}
}

func makeTaskFromRemoteTaskRequest(r *schedulerV1.ScheduleRemoteTaskRequest) *types.Task {
	return &types.Task{
		RemoteConfig: &types.RemoteExecution{
			Method:        executionMethodToHttp(r.GetConfig().GetMethod()),
			URL:           r.GetConfig().GetUrl(),
			Headers:       r.GetConfig().GetHeaders(),
			Body:          r.GetConfig().GetBody(),
			Timeout:       r.GetConfig().GetTimeout(),
			ExpectedCodes: r.GetConfig().GetExpectedCodes(),
		},
		ID:                  r.GetId(),
		Retries:             r.GetRetries(),
		ScheduleOnTimestamp: r.GetScheduleTime().GetSeconds(),
	}
}

func executionMethodToHttp(enum schedulerV1.ExecutionMethod) string {
	switch enum {
	case schedulerV1.ExecutionMethod_EXECUTION_METHOD_GET:
		return http.MethodGet
	case schedulerV1.ExecutionMethod_EXECUTION_METHOD_POST:
		return http.MethodPost
	}

	return ""
}
