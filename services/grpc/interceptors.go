package grpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	GRPCRecovery interface {
		Handle(p interface{}) (err error)
	}
	gRPCRecovery struct{}
)

func NewGRPCRecovery() GRPCRecovery {
	return &gRPCRecovery{}
}

func (r *gRPCRecovery) Handle(p interface{}) (err error) {
	return status.Errorf(codes.Internal, "panic recovered: %s", p)
}

func UnaryLogAndCaptureErrors(log *zap.Logger, debug bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			s := status.Convert(err)
			if s != nil {
				switch s.Code() {
				case codes.Internal, codes.ResourceExhausted, codes.Unimplemented, codes.Unavailable, codes.Unknown:
					log.Error(s.Message(), zap.Error(s.Err()))
					if !debug {
						proto := s.Proto()
						if proto != nil {
							proto.Message = s.Code().String()
							return resp, status.FromProto(proto).Err()
						}
					}
				default:
					return resp, err
				}
			}
		}
		return resp, err
	}
}

func StreamLogAndCaptureErrors(log *zap.Logger, debug bool) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		if err != nil {
			s := status.Convert(err)
			if s != nil {
				switch s.Code() {
				case codes.Internal, codes.ResourceExhausted, codes.Unimplemented, codes.Unavailable, codes.Unknown:
					log.Error(s.Message(), zap.Error(s.Err()))
					if !debug {
						proto := s.Proto()
						if proto != nil {
							proto.Message = s.Code().String()
							return status.FromProto(proto).Err()
						}
					}
				default:
					return err
				}
			}
		}
		return err
	}
}
