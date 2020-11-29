package scheduler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/json-iterator/go"
	"github.com/streadway/amqp"
	"go.uber.org/atomic"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/backends"
	"github.com/DoomSentinel/scheduler/monitoring"
	"github.com/DoomSentinel/scheduler/scheduler/executors"
	"github.com/DoomSentinel/scheduler/scheduler/types"
)

var Module = fx.Options(
	executors.Module,
	fx.Provide(
		//Wrap concrete type to a one we will use in package
		func(bus *backends.AMQP) AMQP {
			return bus
		},
		NewScheduler,
	),
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary

	ErrInvalidMessage = errors.New("invalid message format")
)

type (
	AMQP interface {
		ConsumeTasks() (<-chan amqp.Delivery, error)
		PublishNotification(body []byte) error
		ConsumeNotifications(ctx context.Context) (<-chan amqp.Delivery, error)
		PublishDelayed(body []byte, delay time.Duration, messageType string) error
	}
)

type (
	Params struct {
		fx.In
		Executors []types.Executor `group:"executors"` // group from const types.ExecutorsGroup

		Backend AMQP
		Log     *zap.Logger
		Metrics monitoring.SchedulerMetrics
	}
	Scheduler struct {
		bus     AMQP
		log     *zap.Logger
		metrics monitoring.SchedulerMetrics

		executors map[types.TaskType]types.Executor

		stopChan     chan struct{}
		shuttingDown *atomic.Bool
		running      *atomic.Bool
		wg           sync.WaitGroup
	}
)

func NewScheduler(p Params) *Scheduler {
	execs := make(map[string]types.Executor, len(p.Executors))
	for _, executor := range p.Executors {
		execs[executor.Type()] = executor
	}

	return &Scheduler{
		executors:    execs,
		bus:          p.Backend,
		log:          p.Log,
		stopChan:     make(chan struct{}),
		shuttingDown: atomic.NewBool(false),
		running:      atomic.NewBool(false),
		metrics:      p.Metrics,
	}
}

func (s *Scheduler) ScheduleTask(taskType types.TaskType, task *types.Task) error {
	body, err := json.Marshal(task)
	if err != nil || task == nil {
		return fmt.Errorf("unable to marshal task: %v", err)
	}

	err = s.bus.PublishDelayed(body,
		time.Duration(
			math.Max(0, float64(task.ScheduleOnTimestamp-time.Now().UTC().Unix())),
		)*time.Second, taskType)
	if err != nil {
		return err
	}
	s.metrics.AddScheduled(taskType)
	return nil
}

func (s *Scheduler) Start() error {
	if s.shuttingDown.Load() {
		return errors.New("unable to start scheduler: shutting down")
	}

	s.running.Store(true)
	go s.listen()

	return nil
}

//Subscribe for temporary notifications queue for usage in requested callbacks
func (s *Scheduler) ReceiveNotifications(ctx context.Context) (<-chan *types.ExecutionInfo, error) {
	receiver := make(chan *types.ExecutionInfo)
	messages, err := s.bus.ConsumeNotifications(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(receiver)

		for {
			select {
			case <-ctx.Done():
				return
			case message, ok := <-messages:
				if !ok {
					return
				}

				var info types.ExecutionInfo
				err := json.Unmarshal(message.Body, &info)
				if err != nil {
					s.log.Error("malformed notification message", zap.Error(err))
				}

				receiver <- &info
			case <-s.stopChan:
				return
			}
		}
	}()

	return receiver, nil
}

func (s *Scheduler) Running() bool {
	return s.running.Load()
}

func (s *Scheduler) Shutdown() error {
	if !s.shuttingDown.CAS(false, true) {
		return errors.New("scheduler is already shutting down")
	}
	s.running.Store(false)
	close(s.stopChan)
	s.wg.Wait()
	return nil
}

func (s *Scheduler) listen() {
	messages, err := s.bus.ConsumeTasks()
	if err != nil {
		s.log.Error("unable to listen for events", zap.Error(err))
		return
	}
	for {
		select {
		case message, ok := <-messages:
			if !ok {
				err := s.Shutdown()
				if err != nil {
					s.log.Error("trying to stop scheduler: ", zap.Error(err))
				}
				return
			}

			if executor, err := s.getExecutor(message.Type); err == nil {
				s.wg.Add(1)
				go s.processTask(executor, message.Body)
			} else {
				s.log.Error(err.Error())
			}

			err = message.Ack(false)
			if err != nil {
				s.log.Error("unable acknowledge message", zap.Error(err))
			}

		case <-s.stopChan:
			return
		}
	}
}

func (s *Scheduler) processTask(executor types.Executor, body []byte) {
	defer s.wg.Done()
	var task types.Task
	err := json.Unmarshal(body, &task)
	if err != nil {
		s.log.Error("invalid message", zap.Error(ErrInvalidMessage))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTime := time.Now()
	result, err := executor.Run(ctx, &task)
	finished := time.Since(startTime)

	s.metrics.AddExecuted(executor.Type())
	s.metrics.UpdateExecutionTime(executor.Type(), finished)

	taskInfo := types.ExecutionInfo{
		Duration: finished.Milliseconds(),
		Status:   types.ExecutionStatusSuccess,
		Output:   result,
		Id:       task.ID,
	}
	if err != nil {
		taskInfo.Message = err.Error()
		taskInfo.Status = types.ExecutionStatusFailed
		s.metrics.AddTaskFailed(executor.Type())
	} else {
		s.metrics.AddTaskSuccess(executor.Type())
	}

	err = s.notifyExecutionStatus(taskInfo)
	if err != nil {
		s.log.Error("unable to send notification", zap.Error(err))
	}
}

func (s *Scheduler) notifyExecutionStatus(info types.ExecutionInfo) error {
	bytes, err := json.Marshal(info)
	if err != nil {
		return err
	}

	return s.bus.PublishNotification(bytes)
}

func (s *Scheduler) getExecutor(messageType string) (types.Executor, error) {
	if executor, ok := s.executors[messageType]; ok {
		return executor, nil
	}

	return nil, fmt.Errorf("unable to find executor for type: %s", messageType)
}
