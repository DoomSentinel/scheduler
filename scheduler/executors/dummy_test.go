package executors

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/scheduler/types"
)

func TestNewDummy(t *testing.T) {
	executor := NewDummy(zap.NewNop())
	require.NotNil(t, executor)
	require.Implements(t, new(types.Executor), executor)
}

func TestDummy_Type(t *testing.T) {
	executor := NewDummy(zap.NewNop())
	require.NotNil(t, executor)
	require.Equal(t, types.TaskTypeDummy, executor.Type())
}

func TestDummy_Run(t *testing.T) {
	task := &types.Task{
		ID:                  "some id",
		Retries:             2,
		ScheduleOnTimestamp: time.Now().UTC().Unix(),
	}

	executor := NewDummy(zap.NewNop())

	result, err := executor.Run(context.TODO(), task)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = executor.Run(context.TODO(), nil)
	require.Error(t, err)
	require.Nil(t, result)
}
