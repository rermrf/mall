package grpcx

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// DefaultCallTimeout is the default per-call timeout for gRPC clients.
const DefaultCallTimeout = 5 * time.Second

// DefaultClientDialOptions returns common gRPC client dial options.
func DefaultClientDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(false),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}
}
