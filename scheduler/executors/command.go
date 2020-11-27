package executors

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"

	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/scheduler/types"
)

type (
	Command struct {
		log *zap.Logger
	}
)

func NewCommand(log *zap.Logger) types.Executor {
	return &Command{
		log: log,
	}
}

func (r Command) Type() types.TaskType {
	return types.TaskTypeCommand
}

func (r Command) Run(ctx context.Context, task *types.Task) (types.Result, error) {
	if task.Command == nil {
		return nil, errors.New("invalid command")
	}

	c, cancel := context.WithTimeout(ctx, time.Duration(task.Command.Timeout)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, task.Command.Command, task.Command.Arguments...)
	var buf bytes.Buffer
	cmd.Stdout = &buf

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		buf.WriteString(err.Error())
	}
	return buf.Bytes(), nil
}
