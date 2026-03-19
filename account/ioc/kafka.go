package ioc

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/account/events"
	"github.com/rermrf/mall/account/service"
	"github.com/rermrf/mall/pkg/tenantx"
	"github.com/spf13/viper"
)

func InitKafka() sarama.Client {
	type Config struct {
		Addrs []string `yaml:"addrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("kafka", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Kafka 配置失败: %w", err))
	}
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	client, err := sarama.NewClient(cfg.Addrs, saramaCfg)
	if err != nil {
		panic(fmt.Errorf("连接 Kafka 失败: %w", err))
	}
	return client
}

func InitConsumerGroup(client sarama.Client) sarama.ConsumerGroup {
	cg, err := sarama.NewConsumerGroupFromClient("account-svc", client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka ConsumerGroup 失败: %w", err))
	}
	return cg
}

func NewOrderCompletedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.AccountService,
) *events.OrderCompletedConsumer {
	return events.NewOrderCompletedConsumer(cg, l, func(ctx context.Context, evt events.OrderCompletedEvent) error {
		ctx = tenantx.WithTenantID(ctx, evt.TenantID)
		_, err := svc.CreateSettlement(ctx, evt.TenantID, evt.OrderNo, evt.PaymentNo, evt.Amount)
		return err
	})
}

func InitConsumers(
	completedConsumer *events.OrderCompletedConsumer,
) []events.Consumer {
	return []events.Consumer{completedConsumer}
}
