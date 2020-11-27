package executors

import (
	"bytes"
	"context"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/scheduler/types"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type (
	Dummy struct {
		log *zap.Logger
	}
)

func NewDummy(log *zap.Logger) types.Executor {
	return &Dummy{
		log: log,
	}
}

func (r Dummy) Type() types.TaskType {
	return types.TaskTypeDummy
}

func (r Dummy) Run(_ context.Context, task *types.Task) (types.Result, error) {
	output, err := json.MarshalIndent(task, "", " ")
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	schedulingTime := time.Now().UTC().Unix() - task.ScheduleOnTimestamp
	buf.WriteString(fmt.Sprintf("Scheduling delay (s): %d\n", schedulingTime))
	buf.WriteString(fmt.Sprintf("Received on : %s\n", time.Now().Format("02-Jan-2006 15:04:05")))
	buf.Write(output)

	return buf.Bytes(), nil
}
