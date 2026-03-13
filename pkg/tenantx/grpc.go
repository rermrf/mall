package tenantx

import (
	"context"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const metadataKey = "x-tenant-id"

// GRPCUnaryClientInterceptor 在 gRPC 客户端请求中自动携带 tenant_id
func GRPCUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		tid := GetTenantID(ctx)
		if tid > 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, metadataKey, strconv.FormatInt(tid, 10))
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// GRPCUnaryServerInterceptor 从 gRPC metadata 提取 tenant_id 注入 context
func GRPCUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			vals := md.Get(metadataKey)
			if len(vals) > 0 {
				tid, err := strconv.ParseInt(vals[0], 10, 64)
				if err == nil {
					ctx = WithTenantID(ctx, tid)
				}
			}
		}
		return handler(ctx, req)
	}
}
