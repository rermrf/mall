package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
	igrpc "github.com/rermrf/mall/product/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
)

func InitGRPCServer(productServer *igrpc.ProductGRPCServer, l logger.Logger) *grpcx.Server {
	type Config struct {
		Port      int      `yaml:"port"`
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 gRPC 配置失败: %w", err))
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(tenantx.GRPCUnaryServerInterceptor()))
	productServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "product",
		L:         l,
	}
}

func InitEtcdClient() *clientv3.Client {
	type Config struct {
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 etcd 配置失败: %w", err))
	}
	client, err := clientv3.New(clientv3.Config{Endpoints: cfg.EtcdAddrs})
	if err != nil {
		panic(fmt.Errorf("连接 etcd 失败: %w", err))
	}
	return client
}

func InitTenantClient(etcdClient *clientv3.Client) tenantv1.TenantServiceClient {
	etcdResolver, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(fmt.Errorf("创建 etcd resolver 失败: %w", err))
	}
	conn, err := grpc.NewClient(
		"etcd:///service/tenant",
		grpc.WithResolvers(etcdResolver),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(tenantx.GRPCUnaryClientInterceptor()),
	)
	if err != nil {
		panic(fmt.Errorf("连接 tenant 服务失败: %w", err))
	}
	return tenantv1.NewTenantServiceClient(conn)
}
