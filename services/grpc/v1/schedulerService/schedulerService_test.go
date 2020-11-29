package schedulerService

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	schedulerV1 "github.com/DoomSentinel/scheduler-api/gen/go/v1"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DoomSentinel/scheduler/scheduler/types"
	"github.com/DoomSentinel/scheduler/services/grpc/v1/schedulerService/mocks"
	"github.com/DoomSentinel/scheduler/services/grpc/validation_errors"
)

//go:generate mockgen -destination=./mocks/scheduler.go -package=mocks . Scheduler
//go:generate mockgen -destination=./mocks/scheduler_grpc_stream.go -package=mocks github.com/DoomSentinel/scheduler-api/gen/go/v1 SchedulerService_ExecutionNotificationsServer

func TestSchedulerService_ScheduleRemoteTask_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	service, mock := setupService(ctrl)

	mock.scheduler.EXPECT().ScheduleTask(types.TaskTypeRemote, gomock.Any()).Return(nil).AnyTimes()

	response, err := service.ScheduleRemoteTask(context.TODO(), &schedulerV1.ScheduleRemoteTaskRequest{
		Id:      "not an UUID",
		Retries: 0,
		ScheduleTime: &timestamp.Timestamp{
			Seconds: time.Now().AddDate(0, 3, 0).UTC().Unix(),
		},
		Config: &schedulerV1.RemoteExecutionConfig{
			Method:        schedulerV1.ExecutionMethod_EXECUTION_METHOD_GET,
			Url:           "not valid url",
			Headers:       nil,
			Body:          []byte{'s'},
			Timeout:       0,
			ExpectedCodes: nil,
		},
	})
	require.Error(t, err)
	require.Nil(t, response)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Equal(t, map[string][]error{
		"id":                    {errors.New("must be a valid UUID v4")},
		"schedule_time":         {errors.New("max scheduling time exceeded")},
		"config.expected_codes": {errors.New("cannot be blank")},
		"config.url":            {errors.New("must be a valid URL")},
		"config.timeout":        {errors.New("cannot be blank")},
		"config.body":           {errors.New("must be blank")},
	}, validation_errors.ExtractFieldErrors(err))
}

func TestSchedulerService_ScheduleRemoteTask_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	service, mock := setupService(ctrl)

	request := &schedulerV1.ScheduleRemoteTaskRequest{
		Id:      uuid.New().String(),
		Retries: 3,
		ScheduleTime: &timestamp.Timestamp{
			Seconds: time.Now().AddDate(0, 0, 30).UTC().Unix(),
		},
		Config: &schedulerV1.RemoteExecutionConfig{
			Method:        schedulerV1.ExecutionMethod_EXECUTION_METHOD_GET,
			Url:           "http://validurl.com",
			Headers:       nil,
			Body:          nil,
			Timeout:       5,
			ExpectedCodes: []uint32{200},
		},
	}

	mock.scheduler.EXPECT().ScheduleTask(types.TaskTypeRemote, makeTaskFromRemoteTaskRequest(request)).Return(nil).Times(1)

	response, err := service.ScheduleRemoteTask(context.TODO(), request)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, codes.OK, status.Code(err))
}

func TestSchedulerService_ScheduleDummyTask_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	service, mock := setupService(ctrl)

	mock.scheduler.EXPECT().ScheduleTask(types.TaskTypeDummy, gomock.Any()).Return(nil).AnyTimes()

	response, err := service.ScheduleDummyTask(context.TODO(), &schedulerV1.ScheduleDummyTaskRequest{
		Id: "not an UUID",
		ScheduleTime: &timestamp.Timestamp{
			Seconds: time.Now().AddDate(0, 3, 0).UTC().Unix(),
		},
	})
	require.Error(t, err)
	require.Nil(t, response)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Equal(t, map[string][]error{
		"id":            {errors.New("must be a valid UUID v4")},
		"schedule_time": {errors.New("max scheduling time exceeded")},
	}, validation_errors.ExtractFieldErrors(err))
}

func TestSchedulerService_ScheduleDummyTask_Validation_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	service, mock := setupService(ctrl)

	request := &schedulerV1.ScheduleDummyTaskRequest{
		Id: uuid.New().String(),
		ScheduleTime: &timestamp.Timestamp{
			Seconds: time.Now().UTC().Unix(),
		},
	}

	mock.scheduler.EXPECT().ScheduleTask(types.TaskTypeDummy, makeTaskFromDummyTaskRequest(request)).Return(nil).Times(1)

	response, err := service.ScheduleDummyTask(context.TODO(), request)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, codes.OK, status.Code(err))
}

func TestSchedulerService_ScheduleCommandTask_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	service, mock := setupService(ctrl)

	mock.scheduler.EXPECT().ScheduleTask(types.TaskTypeCommand, gomock.Any()).Return(nil).AnyTimes()

	response, err := service.ScheduleCommandTask(context.TODO(), &schedulerV1.ScheduleCommandTaskRequest{
		Id: "not an UUID",
		ScheduleTime: &timestamp.Timestamp{
			Seconds: time.Now().AddDate(0, 3, 0).UTC().Unix(),
		},
		Config: &schedulerV1.CommandTaskConfig{
			Command:   "",
			Arguments: nil,
			Timeout:   0,
		},
	})
	require.Error(t, err)
	require.Nil(t, response)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Equal(t, map[string][]error{
		"id":             {errors.New("must be a valid UUID v4")},
		"schedule_time":  {errors.New("max scheduling time exceeded")},
		"config.command": {errors.New("cannot be blank")},
		"config.timeout": {errors.New("cannot be blank")},
	}, validation_errors.ExtractFieldErrors(err))
}

func TestSchedulerService_ScheduleCommandTask_Validation_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	service, mock := setupService(ctrl)

	request := &schedulerV1.ScheduleCommandTaskRequest{
		Id: uuid.New().String(),
		ScheduleTime: &timestamp.Timestamp{
			Seconds: time.Now().UTC().Unix(),
		},
		Config: &schedulerV1.CommandTaskConfig{
			Command:   "echo",
			Arguments: []string{"some text"},
			Timeout:   5,
		},
	}

	mock.scheduler.EXPECT().ScheduleTask(types.TaskTypeCommand, makeTaskFromCommandTaskRequest(request)).Return(nil).Times(1)

	response, err := service.ScheduleCommandTask(context.TODO(), request)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, codes.OK, status.Code(err))
}

func TestSchedulerService_ExecutionNotifications(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	service, mock := setupService(ctrl)

	testContext := context.Background()
	testChannel := make(chan *types.ExecutionInfo)

	mock.notificationsStream.EXPECT().Context().Return(testContext).AnyTimes()
	mock.scheduler.EXPECT().ReceiveNotifications(testContext).Return(nil, errors.New("test_error")).Times(1)

	err := service.ExecutionNotifications(nil, mock.notificationsStream)
	require.Error(t, err)

	mock.scheduler.EXPECT().ReceiveNotifications(testContext).Return(testChannel, nil).Times(1)
	mock.notificationsStream.EXPECT().Send(&schedulerV1.ExecutionNotificationsResponse{
		TaskId:   "2",
		Status:   schedulerV1.ExecutionStatus_EXECUTION_STATUS_SUCCESS,
		Duration: 2,
		Output:   nil,
		Message:  "message",
	}).Return(nil).Times(1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := service.ExecutionNotifications(nil, mock.notificationsStream)
		require.NoError(t, err)
	}()
	testChannel <- &types.ExecutionInfo{
		Duration: 2,
		Output:   nil,
		Status:   types.ExecutionStatusSuccess,
		Message:  "message",
		Id:       "2",
	}
	close(testChannel)
	wg.Wait()
}

type testMocks struct {
	scheduler           *mocks.MockScheduler
	notificationsStream *mocks.MockSchedulerService_ExecutionNotificationsServer
}

func setupService(controller *gomock.Controller) (schedulerV1.SchedulerServiceServer, testMocks) {
	scheduler := mocks.NewMockScheduler(controller)
	notificationsStream := mocks.NewMockSchedulerService_ExecutionNotificationsServer(controller)

	return NewValidatedService(
			NewBaseService(scheduler),
			NewValidator(),
		), testMocks{
			scheduler:           scheduler,
			notificationsStream: notificationsStream,
		}
}
