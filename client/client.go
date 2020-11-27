package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	schedulerV1 "github.com/DoomSentinel/scheduler-api/gen/go/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

const (
	maxRetries      = 5
	defaultTimeout  = 10 * time.Second
	backoffDuration = 50 * time.Millisecond
	backoffJitter   = 0.1
	keepAliveTime   = 5 * time.Minute // defaultKeepalivePolicyMinTime

	MethodPost Method = schedulerV1.ExecutionMethod_EXECUTION_METHOD_POST
	MethodGet  Method = schedulerV1.ExecutionMethod_EXECUTION_METHOD_GET

	StatusSuccess Status = schedulerV1.ExecutionStatus_EXECUTION_STATUS_SUCCESS
	StatusFailed  Status = schedulerV1.ExecutionStatus_EXECUTION_STATUS_FAILED
)

type (
	Method = schedulerV1.ExecutionMethod
	Status = schedulerV1.ExecutionStatus

	Conn interface {
		grpc.ClientConnInterface
		io.Closer
	}
	SchedulerClient interface {
		ScheduleDummyTask(ctx context.Context, id string, time time.Time) error
		ScheduleCommandTask(
			ctx context.Context, id string, time time.Time, timeout time.Duration, command string, arguments ...string,
		) error
		ScheduleRemoteTask(ctx context.Context, id string, time time.Time, retries int32, config RemoteConfig) error
		ExecutionNotifications(ctx context.Context) (<-chan *ExecutionInfo, <-chan error, error)

		Close() error
	}
)

type (
	RemoteConfig struct {
		Method        Method
		URL           string
		Headers       map[string]string
		Body          []byte
		ExpectedCodes []uint32
		Timeout       time.Duration
	}
	schedulerClient struct {
		client schedulerV1.SchedulerServiceClient
		conn   Conn
	}
	GRPCError struct {
		err        error
		violations map[string][]string
	}
	ExecutionInfo struct {
		Duration time.Duration
		Output   []byte
		Status   Status
		Message  string
		Id       string
	}
)

func NewDefaultClient(host string, port int) (SchedulerClient, error) {
	return NewSchedulerClient(host, port, DefaultDialOptions())
}

func NewSchedulerClient(host string, port int, options []grpc.DialOption) (SchedulerClient, error) {
	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%d", host, port),
		options...,
	)
	if err != nil {
		return nil, err
	}

	return &schedulerClient{
		client: schedulerV1.NewSchedulerServiceClient(conn),
		conn:   conn,
	}, nil
}

func (s *schedulerClient) ScheduleDummyTask(ctx context.Context, id string, time time.Time) error {
	_, err := s.client.ScheduleDummyTask(ctx, &schedulerV1.ScheduleDummyTaskRequest{
		Id: id,
		ScheduleTime: &timestamp.Timestamp{
			Seconds: time.UTC().Unix(),
		},
	})
	if err != nil {
		return makeGRPCError(err)
	}
	return nil
}

func (s *schedulerClient) ScheduleCommandTask(
	ctx context.Context, id string, time time.Time, timeout time.Duration, command string, arguments ...string,
) error {
	_, err := s.client.ScheduleCommandTask(ctx, &schedulerV1.ScheduleCommandTaskRequest{
		Id: id,
		ScheduleTime: &timestamp.Timestamp{
			Seconds: time.UTC().Unix(),
		},
		Config: &schedulerV1.CommandTaskConfig{
			Command:   command,
			Arguments: arguments,
			Timeout:   uint32(math.Max(1, timeout.Seconds())),
		},
	})
	if err != nil {
		return makeGRPCError(err)
	}
	return nil
}

func (s *schedulerClient) ScheduleRemoteTask(
	ctx context.Context, id string, time time.Time, retries int32, config RemoteConfig,
) error {
	_, err := s.client.ScheduleRemoteTask(ctx, &schedulerV1.ScheduleRemoteTaskRequest{
		Id: id,
		ScheduleTime: &timestamp.Timestamp{
			Seconds: time.UTC().Unix(),
		},
		Retries: retries,
		Config: &schedulerV1.RemoteExecutionConfig{
			Method:        config.Method,
			Url:           config.URL,
			Headers:       config.Headers,
			Body:          config.Body,
			ExpectedCodes: config.ExpectedCodes,
			Timeout:       uint32(math.Max(1, config.Timeout.Seconds())),
		},
	})
	if err != nil {
		return makeGRPCError(err)
	}
	return nil
}

func (s *schedulerClient) ExecutionNotifications(ctx context.Context) (<-chan *ExecutionInfo, <-chan error, error) {
	receiver := make(chan *ExecutionInfo)
	errorsChan := make(chan error)
	str, err := s.client.ExecutionNotifications(ctx, new(schedulerV1.ExecutionNotificationsRequest))
	if err != nil {
		return nil, nil, err
	}

	go func() {
		defer close(receiver)
		defer close(errorsChan)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				data, err := str.Recv()
				if err != nil {
					if errors.Is(err, io.EOF) {
						return
					}
					errorsChan <- err
					continue
				}

				receiver <- &ExecutionInfo{
					Duration: time.Duration(data.GetDuration()) * time.Millisecond,
					Output:   data.GetOutput(),
					Status:   data.GetStatus(),
					Message:  data.GetMessage(),
					Id:       data.GetTaskId(),
				}
			}
		}
	}()

	return receiver, errorsChan, nil
}

func (s *schedulerClient) Close() error {
	return s.conn.Close()
}

func (e ExecutionInfo) String() string {
	buf := bytes.Buffer{}
	buf.WriteString("ID: " + e.Id + "\n")
	buf.WriteString("Status: " + e.Status.String() + "\n")
	buf.WriteString("Duration: " + strconv.Itoa(int(e.Duration.Milliseconds())) + "ms\n")
	if e.Status == StatusFailed {
		buf.WriteString("Message: " + e.Message + "\n")
	}
	if len(e.Output) > 0 {
		buf.WriteString("Output: \n")
		buf.WriteString(strings.Repeat("-", 20) + "\n")
		buf.WriteString(string(e.Output) + "\n")
		buf.WriteString(strings.Repeat("-", 20) + "\n")
	}
	return buf.String()
}

func DefaultDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                keepAliveTime,
			PermitWithoutStream: true,
		}),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			timeoutClientInterceptor(defaultTimeout),
			waitForReadyInterceptor,
			grpc_retry.UnaryClientInterceptor(retryOpts...),
		)),
	}
}

var retryOpts = []grpc_retry.CallOption{
	grpc_retry.WithBackoff(grpc_retry.BackoffLinearWithJitter(backoffDuration, backoffJitter)),
	grpc_retry.WithMax(maxRetries),
	grpc_retry.WithCodes(
		codes.ResourceExhausted,
		codes.Unavailable,
		codes.Aborted,
		codes.Internal,
		codes.Unknown,
	),
}

func timeoutClientInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		cont, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return invoker(cont, method, req, resp, cc, opts...)
	}
}

func waitForReadyInterceptor(
	ctx context.Context,
	method string,
	req, resp interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	opts = append(opts, grpc.WaitForReady(true))
	return invoker(ctx, method, req, resp, cc, opts...)
}

func (e *GRPCError) Error() string {
	if len(e.violations) > 0 {
		buf := bytes.Buffer{}
		buf.WriteString("invalid input: ")
		for field, violations := range e.violations {
			buf.WriteString("field " + field + "[" + strings.Join(violations, " | ") + "]")
		}
		return buf.String()
	}
	return e.err.Error()
}

func (e *GRPCError) GetDetailsMap() map[string]interface{} {
	res := make(map[string]interface{}, len(e.violations))
	for key, value := range e.violations {
		res[key] = value
	}
	return res
}

func makeGRPCError(err error) error {
	violations := make(map[string][]string)
	rpcError := status.Convert(err)
	switch rpcError.Code() {
	case codes.InvalidArgument:
		for _, detail := range rpcError.Details() {
			switch t := detail.(type) {
			case *errdetails.BadRequest:
				for _, violation := range t.GetFieldViolations() {
					violations[violation.GetField()] = append(violations[violation.GetField()], violation.GetDescription())
				}
			}
		}
	}

	grpcErr := new(GRPCError)
	grpcErr.err = err
	grpcErr.violations = violations
	return grpcErr
}
