package ioc

import (
	"fmt"

	"github.com/rermrf/mall/notification/service/provider"
	"github.com/spf13/viper"
)

func InitSmsProvider() provider.SmsProvider {
	type Config struct {
		AccessKeyId     string `yaml:"accessKeyId"`
		AccessKeySecret string `yaml:"accessKeySecret"`
		Endpoint        string `yaml:"endpoint"`
	}
	var cfg Config
	err := viper.UnmarshalKey("sms", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 SMS 配置失败: %w", err))
	}
	return provider.NewAliyunSmsProvider(cfg.AccessKeyId, cfg.AccessKeySecret, cfg.Endpoint)
}

func InitEmailProvider() provider.EmailProvider {
	type Config struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		From     string `yaml:"from"`
	}
	var cfg Config
	err := viper.UnmarshalKey("email", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Email 配置失败: %w", err))
	}
	return provider.NewSmtpEmailProvider(cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.From)
}
