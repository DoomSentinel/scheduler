package grpc

import (
	"context"
	"fmt"
	"net"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/DoomSentinel/scheduler/config"
	"github.com/DoomSentinel/scheduler/monitoring"
	v1 "github.com/DoomSentinel/scheduler/services/grpc/v1"
)

var Module = fx.Options(
	v1.Module,
	fx.Provide(
		NewGRPCRecovery,
		NewListener,
		NewServer,
	),
)

type (
	ListenerResult struct {
		fx.Out
		Listener net.Listener `name:"grpc-listener"`
	}
	ServerResult struct {
		fx.Out
		Server *grpc.Server `name:"server-grpc"`
	}
	ServeGrpcParams struct {
		fx.In

		Lc       fx.Lifecycle
		Listener net.Listener `name:"grpc-listener"`
		Server   *grpc.Server `name:"server-grpc"`
		Monitor  monitoring.Monitor
		Logger   *zap.Logger
	}
	ServerParams struct {
		fx.In
		Logger   *zap.Logger
		Recovery GRPCRecovery
		Debug    config.DebugConfig
	}
)

func NewListener(grpcConfig config.GRPCConfig) (ListenerResult, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", grpcConfig.Host, grpcConfig.Port))
	if err != nil {
		return ListenerResult{}, fmt.Errorf("failed to listen network address: %v", err)
	}

	return ListenerResult{
		Listener: listener,
	}, err
}

func NewServer(params ServerParams) ServerResult {
	server := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_recovery.UnaryServerInterceptor(grpc_recovery.WithRecoveryHandler(params.Recovery.Handle)),
			UnaryLogAndCaptureErrors(params.Logger, params.Debug.Debug),
			grpc_prometheus.DefaultServerMetrics.UnaryServerInterceptor(),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_recovery.StreamServerInterceptor(grpc_recovery.WithRecoveryHandler(params.Recovery.Handle)),
			StreamLogAndCaptureErrors(params.Logger, params.Debug.Debug),
			grpc_prometheus.DefaultServerMetrics.StreamServerInterceptor(),
		),
	)

	return ServerResult{
		Server: server,
	}
}

func ServeGRPC(params ServeGrpcParams) {
	go func() {
		err := params.Server.Serve(params.Listener)
		if err != nil {
			params.Logger.Fatal("failed to serve gRPC", zap.Error(err))
		}
	}()

	params.Monitor.TrackMetrics(
		monitoring.NewGRPCTracker(params.Server),
	)

	params.Lc.Append(fx.Hook{OnStop: func(ctx context.Context) error {
		params.Server.GracefulStop()
		return nil
	}})
}
