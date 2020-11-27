package types

import (
	"context"
)

type TaskType = string
type ExecutionStatus = string

const (
	ExecutorsGroup = "executors"

	MaxResponseSize = 1024 * 1024 //1MB

	TaskTypeRemote  TaskType = "task/remote"
	TaskTypeDummy   TaskType = "task/dummy"
	TaskTypeCommand TaskType = "task/command"

	ExecutionStatusSuccess = "SUCCESS"
	ExecutionStatusFailed  = "FAILED"
)

type Result []byte

type (
	RemoteExecution struct {
		Method        string            `json:"method,omitempty"`
		URL           string            `json:"url,omitempty"`
		Headers       map[string]string `json:"headers,omitempty"`
		Body          []byte            `json:"body,omitempty"`
		Timeout       uint32            `json:"timeout,omitempty"`
		ExpectedCodes []uint32          `json:"expected_codes,omitempty"`
	}

	LocalCommand struct {
		Command   string   `json:"command,omitempty"`
		Arguments []string `json:"arguments,omitempty"`
		Timeout   uint32   `json:"timeout,omitempty"`
	}

	Task struct {
		RemoteConfig        *RemoteExecution `json:"remote_config,omitempty"`
		Command             *LocalCommand    `json:"command,omitempty"`
		ID                  string           `json:"id,omitempty"`
		Retries             int32            `json:"retries,omitempty"`
		ScheduleOnTimestamp int64            `json:"schedule_on_timestamp,omitempty"`
	}

	ExecutionInfo struct {
		Duration int64           `json:"duration"`
		Output   []byte          `json:"output"`
		Status   ExecutionStatus `json:"status"`
		Message  string          `json:"message,omitempty"`
		Id       string          `json:"id"`
	}

	Executor interface {
		Type() TaskType
		Run(ctx context.Context, task *Task) (Result, error)
	}
)
