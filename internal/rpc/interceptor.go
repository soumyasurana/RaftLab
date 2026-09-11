package rpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerLoggingInterceptor returns a gRPC server unary interceptor that logs RPC duration and recovers from panics.
func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		start := time.Now()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("gRPC panic recovered: method=%s panic=%v", info.FullMethod, r)
				err = status.Errorf(codes.Internal, "internal error: %v", r)
			}
		}()

		resp, err = handler(ctx, req)
		duration := time.Since(start)

		if err != nil {
			log.Printf("gRPC call failed: method=%s duration=%v error=%v", info.FullMethod, duration, err)
		}

		return resp, err
	}
}
