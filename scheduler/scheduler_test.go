package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/monitoring/monitoring_mocks"
	"github.com/DoomSentinel/scheduler/scheduler/executors"
	"github.com/DoomSentinel/scheduler/scheduler/mocks"
	"github.com/DoomSentinel/scheduler/scheduler/types"
)

//go:generate mockgen -destination=./mocks/amqp.go -package=mocks . AMQP
//go:generate mockgen -destination=./mocks/ack.go -package=mocks github.com/streadway/amqp Acknowledger

func TestNewScheduler(t *testing.T) {
	scheduler := NewScheduler(Params{
		Executors: nil,
		Bus:       nil,
		Log:       nil,
		Metrics:   nil,
	})
	require.NotNil(t, scheduler)
}

func TestScheduler_Start_Shutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	scheduler, mock := setup(ctrl)

	mock.amqp.EXPECT().ConsumeTasks().Return(make(chan amqp.Delivery), nil).AnyTimes()

	err := scheduler.Start()
	require.NoError(t, err)

	for !scheduler.Running() {
		time.Sleep(500 * time.Millisecond)
	}

	err = scheduler.Shutdown()
	require.NoError(t, err)
	err = scheduler.Shutdown()
	require.Error(t, err)
	require.Equal(t, errors.New("scheduler is already shutting down"), err)
}

func TestScheduler_ScheduleTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	scheduler, mock := setup(ctrl)

	err := scheduler.ScheduleTask(types.TaskTypeDummy, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to marshal task")

	testErr := errors.New("unable to schedule")

	mock.amqp.EXPECT().PublishDelayed(gomock.Any(), 0*time.Second, map[string]interface{}{
		taskTypeHeader: types.TaskTypeDummy,
	}).Return(testErr).Times(1)

	err = scheduler.ScheduleTask(types.TaskTypeDummy, new(types.Task))
	require.Error(t, err)
	require.Equal(t, testErr, err)

	mock.amqp.EXPECT().PublishDelayed(gomock.Any(), 0*time.Second, map[string]interface{}{
		taskTypeHeader: types.TaskTypeCommand,
	}).Return(nil).Times(1)
	mock.metrics.EXPECT().AddScheduled(types.TaskTypeCommand)

	err = scheduler.ScheduleTask(types.TaskTypeCommand, new(types.Task))
	require.NoError(t, err)
}

func TestScheduler_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	scheduler, mock := setup(ctrl)

	deliveryChan := make(chan amqp.Delivery)

	mock.amqp.EXPECT().ConsumeTasks().Return(deliveryChan, nil).Times(1)
	mock.ack.EXPECT().Ack(gomock.Any(), false).Return(nil).Times(4)
	mock.amqp.EXPECT().PublishNotification(gomock.Any()).Return(nil).Times(1)
	mock.metrics.EXPECT().AddExecuted(types.TaskTypeDummy).Times(1)
	mock.metrics.EXPECT().UpdateExecutionTime(types.TaskTypeDummy, gomock.Any()).Times(1)
	mock.metrics.EXPECT().AddTaskSuccess(types.TaskTypeDummy).Times(1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scheduler.listen()
	}()

	deliveryChan <- delivery(mock.ack, new(types.Task), map[string]interface{}{
		taskTypeHeader: types.TaskTypeDummy,
	})

	deliveryChan <- delivery(mock.ack, new(types.Task), map[string]interface{}{
		taskTypeHeader: types.TaskTypeCommand,
	})

	deliveryChan <- delivery(mock.ack, nil, map[string]interface{}{
		taskTypeHeader: types.TaskTypeDummy,
	})

	deliveryChan <- delivery(mock.ack, nil, nil)

	close(deliveryChan)
	wg.Wait()
	err := scheduler.Shutdown()
	require.Error(t, err)
}

func TestScheduler_ReceiveNotifications(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	scheduler, mock := setup(ctrl)

	deliveryChan := make(chan amqp.Delivery)

	mock.amqp.EXPECT().ConsumeNotifications().Return(deliveryChan, nil).Times(1)

	testMessage := &types.ExecutionInfo{
		Duration: 5,
		Output:   []byte("123"),
		Status:   "SUCCESS",
		Message:  "hop",
		Id:       "some id",
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		channel, err := scheduler.ReceiveNotifications(context.Background())
		for msg := range channel {
			require.Equal(t, testMessage, msg)
		}
		require.NoError(t, err)
	}()

	body, _ := json.Marshal(testMessage)
	deliveryChan <- amqp.Delivery{
		Body: body,
	}

	close(deliveryChan)
	wg.Wait()
}

type testMocks struct {
	amqp    *mocks.MockAMQP
	ack     *mocks.MockAcknowledger
	metrics *monitoring_mocks.MockSchedulerMetrics
}

func setup(controller *gomock.Controller) (*Scheduler, testMocks) {
	Amqp := mocks.NewMockAMQP(controller)
	ack := mocks.NewMockAcknowledger(controller)
	metrics := monitoring_mocks.NewMockSchedulerMetrics(controller)

	return NewScheduler(Params{
			Executors: []types.Executor{executors.NewDummy(zap.NewNop())},
			Bus:       Amqp,
			Log:       zap.NewNop(),
			Metrics:   metrics,
		}), testMocks{
			amqp:    Amqp,
			ack:     ack,
			metrics: metrics,
		}
}

func delivery(ack amqp.Acknowledger, task *types.Task, headers map[string]interface{}) amqp.Delivery {
	var bytes []byte
	if task != nil {
		bytes, _ = json.Marshal(task)
	}

	return amqp.Delivery{
		Acknowledger: ack,
		Headers:      headers,
		Timestamp:    time.Time{},
		Body:         bytes,
	}
}
