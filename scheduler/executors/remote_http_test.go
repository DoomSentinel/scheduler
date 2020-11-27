package executors

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gentleMock "gopkg.in/h2non/gentleman-mock.v2"
	"gopkg.in/h2non/gentleman.v2"

	"github.com/DoomSentinel/scheduler/scheduler/types"
)

func TestNewRemoteHttp(t *testing.T) {
	executor := NewRemoteHttp(NewRemoteHTTPClient())
	require.NotNil(t, executor)
	require.Implements(t, new(types.Executor), executor)
}

func TestRemoteHTTP_Type(t *testing.T) {
	executor := NewRemoteHttp(NewRemoteHTTPClient())
	require.NotNil(t, executor)
	require.Equal(t, types.TaskTypeRemote, executor.Type())
}

func TestRemoteHTTP_Run_Success(t *testing.T) {
	expectedReply := []byte("all is good? or is it ?")

	task := &types.Task{
		RemoteConfig: &types.RemoteExecution{
			Method:        http.MethodPost,
			URL:           "http://testurl.com",
			Headers:       map[string]string{"test": "test"},
			Body:          []byte("anything"),
			Timeout:       10,
			ExpectedCodes: []uint32{http.StatusServiceUnavailable},
		},
		ID:                  "some id",
		Retries:             2,
		ScheduleOnTimestamp: time.Now().UTC().Unix(),
	}

	httpClient := gentleman.New()
	gentleMock.New(task.RemoteConfig.URL).Post("").Times(1).Reply(http.StatusServiceUnavailable).Body(bytes.NewBuffer(expectedReply))
	httpClient.Use(gentleMock.Plugin)

	executor := NewRemoteHttp(&RemoteHttpClient{
		Client: httpClient,
	})

	result, err := executor.Run(context.TODO(), task)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, types.Result(expectedReply), result)
}

func TestRemoteHTTP_Run_Fail(t *testing.T) {
	task := &types.Task{
		RemoteConfig: &types.RemoteExecution{
			Method:        http.MethodPost,
			URL:           "http://testurl.com",
			Headers:       map[string]string{"test": "test"},
			Body:          []byte("anything"),
			Timeout:       10,
			ExpectedCodes: []uint32{http.StatusOK},
		},
		ID:                  "some id",
		Retries:             2,
		ScheduleOnTimestamp: time.Now().UTC().Unix(),
	}

	expectedReply := []byte("all is good? or is it ?")
	replyError := errors.New("nope its not")

	httpClient := gentleman.New()
	gentleMock.New(task.RemoteConfig.URL).Post("").Times(3).Reply(http.StatusServiceUnavailable).Body(bytes.NewBuffer(expectedReply))
	gentleMock.New(task.RemoteConfig.URL).Post("").Times(3).ReplyError(replyError)
	httpClient.Use(gentleMock.Plugin)

	executor := NewRemoteHttp(&RemoteHttpClient{
		Client: httpClient,
	})

	result, err := executor.Run(context.TODO(), task)
	require.Error(t, err)
	require.Equal(t, ErrUnexpectedStatusCode, err)
	require.NotNil(t, result)
	require.Equal(t, types.Result(expectedReply), result)

	result, err = executor.Run(context.TODO(), task)
	require.Error(t, err)
	require.Equal(t, &url.Error{
		Op:  "Post",
		URL: task.RemoteConfig.URL,
		Err: replyError,
	}, err)
	require.Nil(t, result)
}
