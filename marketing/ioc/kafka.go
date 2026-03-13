package ioc

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/marketing/events"
	"github.com/rermrf/mall/marketing/service"
	"github.com/rermrf/mall/pkg/saramax"
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

func InitSyncProducer(client sarama.Client) sarama.SyncProducer {
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka SyncProducer 失败: %w", err))
	}
	return producer
}

func InitProducer(p sarama.SyncProducer) events.Producer {
	return events.NewSaramaProducer(p)
}

func InitConsumerGroup(client sarama.Client) sarama.ConsumerGroup {
	cg, err := sarama.NewConsumerGroupFromClient("marketing-svc", client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka ConsumerGroup 失败: %w", err))
	}
	return cg
}

func NewOrderCancelledConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.MarketingService,
	orderClient orderv1.OrderServiceClient,
) *events.OrderCancelledConsumer {
	return events.NewOrderCancelledConsumer(cg, l, func(ctx context.Context, evt events.OrderCancelledEvent) error {
		ctx = tenantx.WithTenantID(ctx, evt.TenantID)
		l.Info("收到订单取消事件",
			logger.String("orderNo", evt.OrderNo),
			logger.Int64("tenantId", evt.TenantID))
		orderResp, err := orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderNo: evt.OrderNo})
		if err != nil {
			l.Error("获取订单详情失败", logger.String("orderNo", evt.OrderNo), logger.Error(err))
			return nil
		}
		couponId := orderResp.GetOrder().GetCouponId()
		if couponId == 0 {
			l.Info("订单未使用优惠券，无需释放", logger.String("orderNo", evt.OrderNo))
			return nil
		}
		if err := svc.ReleaseCoupon(ctx, couponId); err != nil {
			l.Error("释放优惠券失败",
				logger.Int64("couponId", couponId),
				logger.String("orderNo", evt.OrderNo),
				logger.Error(err))
			return err
		}
		l.Info("优惠券释放成功",
			logger.Int64("couponId", couponId),
			logger.String("orderNo", evt.OrderNo))
		return nil
	})
}

func InitConsumers(
	cancelledConsumer *events.OrderCancelledConsumer,
) []saramax.Consumer {
	return []saramax.Consumer{cancelledConsumer}
}
