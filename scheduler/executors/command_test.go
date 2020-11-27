package executors

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/scheduler/types"
)

func TestNewCommand(t *testing.T) {
	executor := NewCommand(zap.NewNop())
	require.NotNil(t, executor)
	require.Implements(t, new(types.Executor), executor)
}

func TestCommand_Type(t *testing.T) {
	executor := NewCommand(zap.NewNop())
	require.NotNil(t, executor)
	require.Equal(t, types.TaskTypeCommand, executor.Type())
}

func TestCommand_Run(t *testing.T) {
	task := &types.Task{
		ID:                  "some id",
		ScheduleOnTimestamp: time.Now().UTC().Unix(),
		Command: &types.LocalCommand{
			Command:   "echo",
			Arguments: []string{"test"},
			Timeout:   15,
		},
	}

	executor := NewCommand(zap.NewNop())

	result, err := executor.Run(context.TODO(), nil)
	require.Error(t, err)
	require.Nil(t, result)

	result, err = executor.Run(context.TODO(), task)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, string(result), "test")

	task.Command.Command = "las$#%SDFG"
	result, err = executor.Run(context.TODO(), task)
	require.Error(t, err)
	require.Nil(t, result)
}
