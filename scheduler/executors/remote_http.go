package executors

import (
	"bytes"
	"context"
	"time"

	"gopkg.in/h2non/gentleman.v2"
	"gopkg.in/h2non/gentleman.v2/plugins/timeout"

	"github.com/DoomSentinel/scheduler/scheduler/types"
)

type (
	RemoteHttpClient struct {
		*gentleman.Client
	}
	RemoteHTTP struct {
		client *RemoteHttpClient
	}
)

func NewRemoteHTTPClient() *RemoteHttpClient {
	return &RemoteHttpClient{
		Client: gentleman.New(),
	}
}

func NewRemoteHttp(client *RemoteHttpClient) types.Executor {
	return &RemoteHTTP{
		client: client,
	}
}

func (r RemoteHTTP) Type() types.TaskType {
	return types.TaskTypeRemote
}

func (r RemoteHTTP) Run(_ context.Context, task *types.Task) (types.Result, error) {
	//place for tracing
	return r.performHttpRequest(task.RemoteConfig, task.Retries)
}

func (r RemoteHTTP) performHttpRequest(config *types.RemoteExecution, retries int32) ([]byte, error) {
	response, err := r.client.Request().
		Use(timeout.Request(time.Duration(config.Timeout) * time.Second)).
		Use(NewRetry(
			NewConstantBackoff(int(retries), 100*time.Millisecond),
			ExceptCodes(config.ExpectedCodes),
		)).
		Method(config.Method).
		URL(config.URL).
		SetHeaders(config.Headers).
		Body(bytes.NewBuffer(config.Body)).
		Do()
	if err != nil {
		return nil, err
	}
	body := response.Bytes()
	if len(response.Bytes()) > types.MaxResponseSize {
		body = nil
	}
	for _, code := range config.ExpectedCodes {
		if response.StatusCode == int(code) {
			return body, nil
		}
	}

	return body, ErrUnexpectedStatusCode
}
