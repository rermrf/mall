package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
	igrpc "github.com/rermrf/mall/inventory/grpc"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func InitGRPCServer(inventoryServer *igrpc.InventoryGRPCServer, l logger.Logger) *grpcx.Server {
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
	inventoryServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "inventory",
		L:         l,
	}
}
