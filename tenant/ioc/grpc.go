package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/grpcx"
	igrpc "github.com/rermrf/mall/tenant/grpc"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func InitGRPCServer(tenantServer *igrpc.TenantGRPCServer, l logger.Logger) *grpcx.Server {
	type Config struct {
		Port      int      `yaml:"port"`
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 gRPC 配置失败: %w", err))
	}
	server := grpc.NewServer()
	tenantServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "tenant",
		L:         l,
	}
}
