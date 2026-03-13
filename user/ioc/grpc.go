package ioc

import (
	"fmt"

	igrpc "github.com/rermrf/mall/user/grpc"

	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/rermrf/emo/logger"
)

func InitGRPCServer(userServer *igrpc.UserGRPCServer, l logger.Logger) *grpcx.Server {
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
	userServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "user",
		L:         l,
	}
}
