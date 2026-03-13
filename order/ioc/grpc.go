package ioc

import (
	"fmt"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rermrf/emo/logger"
	inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"
	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	ogrpc "github.com/rermrf/mall/order/grpc"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
)

func InitEtcdClient() *clientv3.Client {
	var cfg struct {
		Addrs []string `yaml:"addrs"`
	}
	err := viper.UnmarshalKey("etcd", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 etcd 配置失败: %w", err))
	}
	client, err := clientv3.New(clientv3.Config{Endpoints: cfg.Addrs})
	if err != nil {
		panic(fmt.Errorf("连接 etcd 失败: %w", err))
	}
	return client
}

func initServiceConn(etcdClient *clientv3.Client, serviceName string) *grpc.ClientConn {
	etcdResolver, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(fmt.Errorf("创建 etcd resolver 失败: %w", err))
	}
	conn, err := grpc.NewClient(
		"etcd:///service/"+serviceName,
		grpc.WithResolvers(etcdResolver),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(tenantx.GRPCUnaryClientInterceptor()),
	)
	if err != nil {
		panic(fmt.Errorf("连接 gRPC 服务 %s 失败: %w", serviceName, err))
	}
	return conn
}

func InitProductClient(etcdClient *clientv3.Client) productv1.ProductServiceClient {
	return productv1.NewProductServiceClient(initServiceConn(etcdClient, "product"))
}

func InitInventoryClient(etcdClient *clientv3.Client) inventoryv1.InventoryServiceClient {
	return inventoryv1.NewInventoryServiceClient(initServiceConn(etcdClient, "inventory"))
}

func InitPaymentClient(etcdClient *clientv3.Client) paymentv1.PaymentServiceClient {
	return paymentv1.NewPaymentServiceClient(initServiceConn(etcdClient, "payment"))
}

func InitUserClient(etcdClient *clientv3.Client) userv1.UserServiceClient {
	return userv1.NewUserServiceClient(initServiceConn(etcdClient, "user"))
}

func InitGRPCServer(orderServer *ogrpc.OrderGRPCServer, l logger.Logger) *grpcx.Server {
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
	orderServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "order",
		L:         l,
	}
}
