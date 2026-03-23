package ioc

import (
	"fmt"

	"github.com/smartwalle/alipay/v3"
	"github.com/spf13/viper"
)

func InitAlipayClient() *alipay.Client {
	type Config struct {
		AppId           string `yaml:"appId"`
		PrivateKey      string `yaml:"privateKey"`
		AlipayPublicKey string `yaml:"alipayPublicKey"`
		IsProd          bool   `yaml:"isProd"`
	}
	var cfg Config
	if err := viper.UnmarshalKey("alipay", &cfg); err != nil {
		panic(fmt.Errorf("读取支付宝配置失败: %w", err))
	}
	if cfg.AppId == "" {
		return nil // alipay not configured, skip
	}
	client, err := alipay.New(cfg.AppId, cfg.PrivateKey, cfg.IsProd)
	if err != nil {
		panic(fmt.Errorf("初始化支付宝客户端失败: %w", err))
	}
	if err = client.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
		panic(fmt.Errorf("加载支付宝公钥失败: %w", err))
	}
	return client
}
