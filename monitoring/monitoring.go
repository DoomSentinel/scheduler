package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/config"
)

var Module = fx.Provide(
	NewLogger,
	New,
	NewSchedulerMetrics,
)

const (
	Endpoint = "/metrics"

	EchoShutdownTimeout = 5 * time.Second
)

var (
	ErrServerNotSpecified   = fmt.Errorf("web server is not specified")
	ErrServerAlreadyStarted = fmt.Errorf("server already started")
)

type (
	Monitor interface {
		TrackMetrics(metrics ...MetricTracker) Monitor
		StartDetached(logger *zap.Logger)
		Start() error
		GracefulStop() error
		Error() error
	}

	monitor struct {
		config *config.MonitoringConfig
		echo   *echo.Echo

		isServerStarted bool

		err          error
		serverErrors chan error
		once         sync.Once
	}
)

func New(conf config.MonitoringConfig) Monitor {
	return &monitor{
		config:       &conf,
		serverErrors: make(chan error),
	}
}

func (m *monitor) TrackMetrics(metrics ...MetricTracker) Monitor {
	if m.err != nil {
		return m
	}

	for _, tracker := range metrics {
		err := tracker.SetUp()
		if err != nil {
			m.err = fmt.Errorf("track metrics: %v", err)
			return m
		}
	}

	return m
}

func (m *monitor) StartDetached(logger *zap.Logger) {
	go func(logger *zap.Logger) {
		if err := m.Start(); err != nil {
			logger.Warn(err.Error())
		}
	}(logger)
}

func (m *monitor) Start() error {
	if m.err != nil {
		return m.err
	}

	if m.isServerStarted {
		m.err = fmt.Errorf("server start: %v", ErrServerAlreadyStarted)
		return m.err
	}

	m.echo = echo.New()
	m.echo.Debug = false
	m.echo.HideBanner = true
	m.echo.HidePort = true
	m.echo.GET(Endpoint, echo.WrapHandler(promhttp.Handler()))

	return m.runHttpServer()
}

func (m *monitor) GracefulStop() error {
	defer m.once.Do(func() {
		close(m.serverErrors)
	})

	if m.echo == nil {
		return fmt.Errorf("stopping server: %v", ErrServerNotSpecified)
	}

	ctx, cancel := context.WithTimeout(context.Background(), EchoShutdownTimeout)
	defer cancel()
	if err := m.echo.Shutdown(ctx); err != nil {
		m.err = fmt.Errorf("monitoring shutdown failed: %v", err)
		return m.err
	}

	return nil
}

func (m *monitor) Error() error {
	return m.err
}

func (m *monitor) runHttpServer() error {
	m.isServerStarted = true

	go func(errorCh chan error) {
		if err := m.echo.Start(fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)); err != nil {
			if err != http.ErrServerClosed {
				errorCh <- fmt.Errorf("monitoring server failure: %v", err)
			}
		}
	}(m.serverErrors)

	for err := range m.serverErrors {
		m.isServerStarted = false
		return err
	}
	return nil
}
