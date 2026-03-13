package ioc

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"

	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	ijwt "github.com/rermrf/mall/admin-bff/handler/jwt"
)

func InitJWTHandler(userClient userv1.UserServiceClient, rdb redis.Cmdable) *ijwt.JWTHandler {
	var cfg struct {
		AccessSecret  string `yaml:"accessSecret"`
		RefreshSecret string `yaml:"refreshSecret"`
	}
	err := viper.UnmarshalKey("jwt", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 JWT 配置失败: %w", err))
	}
	return ijwt.NewJWTHandler(userClient, rdb, cfg.AccessSecret, cfg.RefreshSecret)
}
