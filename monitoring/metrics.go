package monitoring

import (
	"fmt"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

var (
	ErrServerIsNotSet = fmt.Errorf("traking object is not set")
)

const (
	subsystem     = "scheduler"
	labelTaskType = "task_type"
)

type (
	MetricTracker interface {
		SetUp() error
	}
	SchedulerMetrics interface {
		AddScheduled(taskType string) SchedulerMetrics
		AddExecuted(taskType string) SchedulerMetrics
		UpdateExecutionTime(taskType string, duration time.Duration) SchedulerMetrics
		AddTaskSuccess(taskType string) SchedulerMetrics
		AddTaskFailed(taskType string) SchedulerMetrics
	}
)

type (
	schedulerMetrics struct {
		scheduled     *prometheus.CounterVec
		executed      *prometheus.CounterVec
		executionTime *prometheus.GaugeVec
		taskSuccess   *prometheus.CounterVec
		taskFailed    *prometheus.CounterVec
	}
	grpcMetricTracker struct {
		server *grpc.Server
	}
)

func NewSchedulerMetrics() SchedulerMetrics {
	scheduled := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: subsystem,
		Name:      "scheduled_total",
		Help:      "Total tasks scheduled",
	}, []string{labelTaskType})

	executed := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: subsystem,
		Name:      "processed_total",
		Help:      "Total tasks executed",
	}, []string{labelTaskType})

	executionTime := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: subsystem,
		Name:      "execution_time_seconds",
		Help:      "Execution time of tasks",
	}, []string{labelTaskType})

	taskSuccess := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: subsystem,
		Name:      "success_total",
		Help:      "Total success tasks",
	}, []string{labelTaskType})

	taskFailed := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: subsystem,
		Name:      "failed_total",
		Help:      "Total failed tasks",
	}, []string{labelTaskType})

	prometheus.DefaultRegisterer.MustRegister(
		scheduled, executed, executionTime, taskSuccess, taskFailed,
	)

	return &schedulerMetrics{
		scheduled:     scheduled,
		executed:      executed,
		executionTime: executionTime,
		taskSuccess:   taskSuccess,
		taskFailed:    taskFailed,
	}
}

func NewGRPCTracker(server *grpc.Server) MetricTracker {
	return &grpcMetricTracker{
		server: server,
	}
}

func (s *schedulerMetrics) AddScheduled(taskType string) SchedulerMetrics {
	s.scheduled.With(map[string]string{
		labelTaskType: taskType,
	}).Inc()
	return s
}

func (s *schedulerMetrics) AddExecuted(taskType string) SchedulerMetrics {
	s.executed.With(map[string]string{
		labelTaskType: taskType,
	}).Inc()
	return s
}

func (s *schedulerMetrics) UpdateExecutionTime(taskType string, duration time.Duration) SchedulerMetrics {
	s.executionTime.With(map[string]string{
		labelTaskType: taskType,
	}).Set(duration.Seconds())
	return s
}

func (s *schedulerMetrics) AddTaskSuccess(taskType string) SchedulerMetrics {
	s.taskSuccess.With(map[string]string{
		labelTaskType: taskType,
	}).Inc()
	return s
}

func (s *schedulerMetrics) AddTaskFailed(taskType string) SchedulerMetrics {
	s.taskFailed.With(map[string]string{
		labelTaskType: taskType,
	}).Inc()
	return s
}

func (g grpcMetricTracker) SetUp() error {
	if g.server == nil {
		return errors.Wrap(ErrServerIsNotSet, "setup failure")
	}

	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.Register(g.server)
	return nil
}
