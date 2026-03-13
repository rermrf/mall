package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	cgrpc "github.com/rermrf/mall/cart/grpc"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
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

func InitGRPCServer(cartServer *cgrpc.CartGRPCServer, l logger.Logger) *grpcx.Server {
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
	cartServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "cart",
		L:         l,
	}
}
