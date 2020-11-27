package scheduler_service

import (
	"context"
	"errors"
	"time"

	schedulerV1 "github.com/DoomSentinel/scheduler-api/gen/go/v1"
	ov "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/DoomSentinel/scheduler/services/grpc/validation_errors"
)

type (
	ValidatedService interface {
		schedulerV1.SchedulerServiceServer
	}
	Validator interface {
		ValidateScheduleRemoteTask(r *schedulerV1.ScheduleRemoteTaskRequest) error
		ValidateScheduleDummyTask(r *schedulerV1.ScheduleDummyTaskRequest) error
		ValidateScheduleCommandTask(r *schedulerV1.ScheduleCommandTaskRequest) error
	}
)

const (
	MaxRequestSize   = 1024 * 1024 //1MB
	MaxScheduleDelay = 4294967295  //ms

	MinValidTimeout = 1
)

type (
	validatedService struct {
		service   schedulerV1.SchedulerServiceServer
		validator Validator
	}
	validator struct{}
)

func NewValidatedService(service BaseService, validator Validator) ValidatedService {
	return &validatedService{
		service:   service,
		validator: validator,
	}
}

func NewValidator() Validator {
	return &validator{}
}

func (v validator) ValidateScheduleRemoteTask(r *schedulerV1.ScheduleRemoteTaskRequest) error {
	return validation_errors.FromValidation(ov.ValidateStruct(r,
		ov.Field(&r.Id, ov.Required, is.UUIDv4),
		ov.Field(&r.ScheduleTime, ov.By(func(value interface{}) error {
			scheduleDelay := (r.GetScheduleTime().GetSeconds() - time.Now().UTC().Unix()) * 1000
			if scheduleDelay > MaxScheduleDelay {
				return errors.New("max scheduling time exceeded")
			}

			return nil
		})),
		ov.Field(&r.Config, ov.Required, ov.By(func(value interface{}) error {
			return ov.ValidateStruct(r.Config,
				ov.Field(&r.Config.Method, ov.Required,
					ov.In(schedulerV1.ExecutionMethod_EXECUTION_METHOD_POST, schedulerV1.ExecutionMethod_EXECUTION_METHOD_GET),
				),
				ov.Field(&r.Config.Url, ov.Required, is.URL),
				ov.Field(&r.Config.Timeout, ov.Required, ov.Min(uint32(MinValidTimeout))),
				ov.Field(&r.Config.Body,
					ov.When(
						r.GetConfig().GetMethod() == schedulerV1.ExecutionMethod_EXECUTION_METHOD_GET,
						ov.Empty,
					).
						Else(ov.By(func(value interface{}) error {
							if len(r.GetConfig().GetBody()) > MaxRequestSize {
								return errors.New("request body should not exceed 1Mib")
							}
							return nil
						})),
				),
				ov.Field(&r.Config.ExpectedCodes, ov.Required),
			)
		})),
	))
}

func (v validator) ValidateScheduleDummyTask(r *schedulerV1.ScheduleDummyTaskRequest) error {
	return validation_errors.FromValidation(ov.ValidateStruct(r,
		ov.Field(&r.Id, ov.Required, is.UUIDv4),
		ov.Field(&r.ScheduleTime, ov.By(func(value interface{}) error {
			scheduleDelay := (r.GetScheduleTime().GetSeconds() - time.Now().UTC().Unix()) * 1000
			if scheduleDelay > MaxScheduleDelay {
				return errors.New("max scheduling time exceeded")
			}

			return nil
		})),
	))
}

func (v validator) ValidateScheduleCommandTask(r *schedulerV1.ScheduleCommandTaskRequest) error {
	return validation_errors.FromValidation(ov.ValidateStruct(r,
		ov.Field(&r.Id, ov.Required, is.UUIDv4),
		ov.Field(&r.ScheduleTime, ov.By(func(value interface{}) error {
			scheduleDelay := (r.GetScheduleTime().GetSeconds() - time.Now().UTC().Unix()) * 1000
			if scheduleDelay > MaxScheduleDelay {
				return errors.New("max scheduling time exceeded")
			}

			return nil
		})),
		ov.Field(&r.Config, ov.Required, ov.By(func(value interface{}) error {
			return ov.ValidateStruct(r.Config,
				ov.Field(&r.Config.Command, ov.Required),
				ov.Field(&r.Config.Timeout, ov.Required, ov.Min(uint32(MinValidTimeout))),
			)
		})),
	))
}

func (v validatedService) ScheduleRemoteTask(ctx context.Context, request *schedulerV1.ScheduleRemoteTaskRequest) (*schedulerV1.ScheduleRemoteTaskResponse, error) {
	if err := v.validator.ValidateScheduleRemoteTask(request); err != nil {
		return nil, err
	}

	return v.service.ScheduleRemoteTask(ctx, request)
}

func (v validatedService) ScheduleDummyTask(ctx context.Context, request *schedulerV1.ScheduleDummyTaskRequest) (*schedulerV1.ScheduleDummyTaskResponse, error) {
	if err := v.validator.ValidateScheduleDummyTask(request); err != nil {
		return nil, err
	}

	return v.service.ScheduleDummyTask(ctx, request)
}

func (v validatedService) ScheduleCommandTask(ctx context.Context, request *schedulerV1.ScheduleCommandTaskRequest) (*schedulerV1.ScheduleCommandTaskResponse, error) {
	if err := v.validator.ValidateScheduleCommandTask(request); err != nil {
		return nil, err
	}

	return v.service.ScheduleCommandTask(ctx, request)
}

func (v validatedService) ExecutionNotifications(request *schedulerV1.ExecutionNotificationsRequest, server schedulerV1.SchedulerService_ExecutionNotificationsServer) error {
	return v.service.ExecutionNotifications(request, server)
}
