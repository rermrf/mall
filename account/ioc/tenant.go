package ioc

import (
	"fmt"

	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/account/integration"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitTenantClient(etcdClient *clientv3.Client) tenantv1.TenantServiceClient {
	r, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(fmt.Errorf("创建 etcd resolver 失败: %w", err))
	}
	conn, err := grpc.NewClient(
		"etcd:///service/tenant",
		grpc.WithResolvers(r),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Errorf("连接 tenant gRPC 服务失败: %w", err))
	}
	return tenantv1.NewTenantServiceClient(conn)
}

func InitTenantIntegration(client tenantv1.TenantServiceClient) integration.TenantIntegration {
	return integration.NewTenantIntegration(client)
}
