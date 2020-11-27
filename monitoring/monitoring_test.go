package monitoring

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"

	"github.com/DoomSentinel/scheduler/config"
)

//go:generate mockgen -destination=./monitoring_mocks/metrics.go -package=monitoring_mocks . SchedulerMetrics

func TestNewLogger(t *testing.T) {
	lc := fxtest.NewLifecycle(t)
	logger := NewLogger(lc, config.DebugConfig{Debug: true})
	require.NotNil(t, logger)
	lc.RequireStart()
	lc.RequireStop()

	require.True(t, LoggerValid(logger))
	require.False(t, LoggerValid(nil))
}
