package ioc

import (
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/saramax"
	"github.com/rermrf/mall/search/events"
	"github.com/rermrf/mall/search/service"
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
	cg, err := sarama.NewConsumerGroupFromClient("search-svc", client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka ConsumerGroup 失败: %w", err))
	}
	return cg
}

func NewProductUpdatedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	productClient productv1.ProductServiceClient,
	svc service.SearchService,
) *events.ProductUpdatedConsumer {
	return events.NewProductUpdatedConsumer(cg, l, productClient, svc)
}

func NewProductStatusChangedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	productClient productv1.ProductServiceClient,
	svc service.SearchService,
) *events.ProductStatusChangedConsumer {
	return events.NewProductStatusChangedConsumer(cg, l, productClient, svc)
}

func InitConsumers(
	updatedConsumer *events.ProductUpdatedConsumer,
	statusChangedConsumer *events.ProductStatusChangedConsumer,
) []saramax.Consumer {
	return []saramax.Consumer{updatedConsumer, statusChangedConsumer}
}
