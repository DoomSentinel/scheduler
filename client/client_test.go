package client

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	schedulerV1 "github.com/DoomSentinel/scheduler-api/gen/go/v1"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

//go:generate mockgen -destination=./service_mock.go -package=client github.com/DoomSentinel/scheduler-api/gen/go/v1 SchedulerServiceServer
//go:generate mockgen -destination=./stream_mock.go -package=client github.com/DoomSentinel/scheduler-api/gen/go/v1 SchedulerService_ExecutionNotificationsClient
//go:generate mockgen -destination=./clinet_mock.go -package=client github.com/DoomSentinel/scheduler-api/gen/go/v1 SchedulerServiceClient

const bufSize = 1024 * 1024

func TestSchedulerClient_ScheduleDummyTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client, mock, done := setupService(ctrl, t)
	defer close(done)

	ctx := context.Background()
	now := time.Now()

	mock.service.EXPECT().ScheduleDummyTask(gomock.Any(), &schedulerV1.ScheduleDummyTaskRequest{
		Id:           "1",
		ScheduleTime: &timestamp.Timestamp{Seconds: now.Unix()},
	}).Return(nil, errors.New("error")).Times(5) // Because default retry is 5

	err := client.ScheduleDummyTask(ctx, "1", now)
	require.Error(t, err)
	require.IsType(t, new(GRPCError), err)
	require.Contains(t, err.Error(), "error")

	mock.service.EXPECT().ScheduleDummyTask(gomock.Any(), &schedulerV1.ScheduleDummyTaskRequest{
		Id:           "2",
		ScheduleTime: &timestamp.Timestamp{Seconds: now.Unix()},
	}).Return(new(schedulerV1.ScheduleDummyTaskResponse), nil).Times(1)

	err = client.ScheduleDummyTask(ctx, "2", now)
	require.NoError(t, err)
}

func TestSchedulerClient_ScheduleRemoteTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client, mock, done := setupService(ctrl, t)
	defer close(done)

	ctx := context.Background()
	now := time.Now()

	errorResponse, _ := status.New(codes.InvalidArgument, codes.InvalidArgument.String()).WithDetails(&errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{
				Field:       "cookies",
				Description: "mm that tasty cookies",
			},
		},
	})

	mock.service.EXPECT().ScheduleRemoteTask(gomock.Any(), &schedulerV1.ScheduleRemoteTaskRequest{
		Id:           "1",
		Retries:      3,
		ScheduleTime: &timestamp.Timestamp{Seconds: now.Unix()},
		Config: &schedulerV1.RemoteExecutionConfig{
			Method:  schedulerV1.ExecutionMethod_EXECUTION_METHOD_POST,
			Url:     "something something - THE DARK SIDE",
			Timeout: 15,
		},
	}).Return(nil, errorResponse.Err()).Times(1) //No retries for validation

	err := client.ScheduleRemoteTask(ctx, "1", now, 3, RemoteConfig{
		Method:  MethodPost,
		URL:     "something something - THE DARK SIDE",
		Timeout: 15 * time.Second,
	})
	require.Error(t, err)
	require.IsType(t, new(GRPCError), err)
	require.Equal(t, err.Error(), "invalid input: field cookies: [mm that tasty cookies]")

	mock.service.EXPECT().ScheduleRemoteTask(gomock.Any(), &schedulerV1.ScheduleRemoteTaskRequest{
		Id:           "2",
		Retries:      3,
		ScheduleTime: &timestamp.Timestamp{Seconds: now.Unix()},
		Config: &schedulerV1.RemoteExecutionConfig{
			Method:  schedulerV1.ExecutionMethod_EXECUTION_METHOD_POST,
			Url:     "something something - THE DARK SIDE",
			Timeout: 15,
		},
	}).Return(new(schedulerV1.ScheduleRemoteTaskResponse), nil).Times(1)

	err = client.ScheduleRemoteTask(ctx, "2", now, 3, RemoteConfig{
		Method:  MethodPost,
		URL:     "something something - THE DARK SIDE",
		Timeout: 15 * time.Second,
	})
	require.NoError(t, err)
}

func TestSchedulerClient_ScheduleCommandTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client, mock, done := setupService(ctrl, t)
	defer close(done)

	ctx := context.Background()
	now := time.Now()

	errorResponse, _ := status.New(codes.InvalidArgument, codes.InvalidArgument.String()).WithDetails(&errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{
				Field:       "cookies",
				Description: "mm that tasty cookies",
			},
		},
	})

	mock.service.EXPECT().ScheduleCommandTask(gomock.Any(), &schedulerV1.ScheduleCommandTaskRequest{
		Id:           "1",
		ScheduleTime: &timestamp.Timestamp{Seconds: now.Unix()},
		Config: &schedulerV1.CommandTaskConfig{
			Command:   "echo",
			Arguments: []string{"text"},
			Timeout:   15,
		},
	}).Return(nil, errorResponse.Err()).Times(1) //No retries for validation

	err := client.ScheduleCommandTask(ctx, "1", now, 15*time.Second, "echo", "text")
	require.Error(t, err)
	require.IsType(t, new(GRPCError), err)
	require.Equal(t, map[string]interface{}{
		"cookies": []string{"mm that tasty cookies"},
	}, err.(*GRPCError).GetDetailsMap())
	require.Equal(t, err.Error(), "invalid input: field cookies: [mm that tasty cookies]")

	mock.service.EXPECT().ScheduleCommandTask(gomock.Any(), &schedulerV1.ScheduleCommandTaskRequest{
		Id:           "2",
		ScheduleTime: &timestamp.Timestamp{Seconds: now.Unix()},
		Config: &schedulerV1.CommandTaskConfig{
			Command:   "echo",
			Arguments: []string{"text"},
			Timeout:   15,
		},
	}).Return(new(schedulerV1.ScheduleCommandTaskResponse), nil).Times(1)

	err = client.ScheduleCommandTask(ctx, "2", now, 15*time.Second, "echo", "text")
	require.NoError(t, err)
}

func TestSchedulerClient_ExecutionNotifications(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := NewMockSchedulerServiceClient(ctrl)
	streamMock := NewMockSchedulerService_ExecutionNotificationsClient(ctrl)
	client := schedulerClient{
		client: mockClient,
	}
	ctx := context.Background()

	mockClient.EXPECT().ExecutionNotifications(ctx, new(schedulerV1.ExecutionNotificationsRequest), gomock.Any()).
		Return(streamMock, errors.New("error")).Times(1)

	_, _, err := client.ExecutionNotifications(ctx)
	require.Error(t, err)
	require.Error(t, err)
	require.IsType(t, new(GRPCError), err)

	mockClient.EXPECT().ExecutionNotifications(ctx, new(schedulerV1.ExecutionNotificationsRequest), gomock.Any()).
		Return(streamMock, nil).Times(1)
	messageChan, errChan, err := client.ExecutionNotifications(ctx)
	require.NoError(t, err)

	sendChan := make(chan *schedulerV1.ExecutionNotificationsResponse)

	streamMock.EXPECT().Recv().DoAndReturn(func() (*schedulerV1.ExecutionNotificationsResponse, error) {
		for msg := range sendChan {
			return msg, nil
		}
		return nil, errors.New("stream mega error")
	}).AnyTimes()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case notification, ok := <-messageChan:
				if !ok {
					return
				}
				require.Equal(t, &ExecutionInfo{
					Duration: 1500 * time.Millisecond,
					Status:   StatusSuccess,
					Message:  "nooooo",
					Id:       "1",
				}, notification)
			case err, ok := <-errChan:
				if !ok {
					return
				}
				require.Error(t, err)
				require.Contains(t, err.Error(), "stream mega error")
				return
			}
		}
	}()

	sendChan <- &schedulerV1.ExecutionNotificationsResponse{
		TaskId:   "1",
		Status:   schedulerV1.ExecutionStatus_EXECUTION_STATUS_SUCCESS,
		Duration: 1500,
		Message:  "nooooo",
	}

	close(sendChan)

	wg.Wait()
}

type testMocks struct {
	service *MockSchedulerServiceServer
}

func setupService(controller *gomock.Controller, t *testing.T) (SchedulerClient, testMocks, chan struct{}) {
	service := NewMockSchedulerServiceServer(controller)
	done := make(chan struct{})
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	schedulerV1.RegisterSchedulerServiceServer(s, service)
	go func() {
		if err := s.Serve(lis); err != nil {
			require.NoError(t, err)
		}
	}()

	go func() {
		<-done
		s.GracefulStop()
	}()

	client, err := NewSchedulerClient("test", 0, append([]grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}),
	}, DefaultDialOptions()...))
	require.NoError(t, err)

	return client, testMocks{
		service: service,
	}, done
}
