package ioc

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	"github.com/rermrf/mall/payment/service/channel"
)

func InitWechatConfig() *channel.WechatConfig {
	type Config struct {
		AppId          string `yaml:"appId"`
		MchId          string `yaml:"mchId"`
		MchApiV3Key    string `yaml:"mchApiV3Key"`
		PrivateKeyPath string `yaml:"privateKeyPath"`
		SerialNo       string `yaml:"serialNo"`
		NotifyUrl      string `yaml:"notifyUrl"`
	}
	var cfg Config
	if err := viper.UnmarshalKey("wechat", &cfg); err != nil {
		panic(fmt.Errorf("读取微信支付配置失败: %w", err))
	}
	if cfg.MchId == "" {
		return nil // wechat not configured, skip
	}
	return &channel.WechatConfig{
		AppId:          cfg.AppId,
		MchId:          cfg.MchId,
		MchApiV3Key:    cfg.MchApiV3Key,
		NotifyUrl:      cfg.NotifyUrl,
		PrivateKeyPath: cfg.PrivateKeyPath,
		SerialNo:       cfg.SerialNo,
	}
}

func InitWechatClient(cfg *channel.WechatConfig) *core.Client {
	if cfg == nil {
		return nil // wechat not configured, skip
	}

	privateKey, err := utils.LoadPrivateKeyWithPath(cfg.PrivateKeyPath)
	if err != nil {
		panic(fmt.Errorf("加载微信支付私钥失败: %w", err))
	}

	opts := []core.ClientOption{
		option.WithWechatPayAutoAuthCipher(cfg.MchId, cfg.SerialNo, privateKey, cfg.MchApiV3Key),
	}
	client, err := core.NewClient(context.Background(), opts...)
	if err != nil {
		panic(fmt.Errorf("初始化微信支付客户端失败: %w", err))
	}

	return client
}
