package ioc

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/order/events"
	"github.com/rermrf/mall/order/service"
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
	cg, err := sarama.NewConsumerGroupFromClient("order-svc", client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka ConsumerGroup 失败: %w", err))
	}
	return cg
}

func NewOrderPaidConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.OrderService,
) *events.OrderPaidConsumer {
	return events.NewOrderPaidConsumer(cg, l, func(ctx context.Context, evt events.OrderPaidEvent) error {
		return svc.HandleOrderPaid(ctx, evt)
	})
}

func NewOrderCloseDelayConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.OrderService,
) *events.OrderCloseDelayConsumer {
	return events.NewOrderCloseDelayConsumer(cg, l, func(ctx context.Context, orderNo string) error {
		return svc.HandleOrderCloseDelay(ctx, orderNo)
	})
}

func NewSeckillSuccessConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.OrderService,
	userClient userv1.UserServiceClient,
) *events.SeckillSuccessConsumer {
	return events.NewSeckillSuccessConsumer(cg, l, func(ctx context.Context, evt events.SeckillSuccessEvent) error {
		ctx = tenantx.WithTenantID(ctx, evt.TenantId)
		l.Info("收到秒杀成功事件，准备创建秒杀订单",
			logger.Int64("userId", evt.UserId),
			logger.Int64("skuId", evt.SkuId),
			logger.Int64("seckillPrice", evt.SeckillPrice))

		// 1. 获取用户默认收货地址
		addrResp, err := userClient.ListAddresses(ctx, &userv1.ListAddressesRequest{UserId: evt.UserId})
		if err != nil {
			l.Error("获取用户收货地址失败",
				logger.Int64("userId", evt.UserId),
				logger.Error(err))
			return err
		}
		addresses := addrResp.GetAddresses()
		if len(addresses) == 0 {
			l.Error("用户无收货地址，无法创建秒杀订单",
				logger.Int64("userId", evt.UserId))
			return nil
		}
		// 使用第一个地址作为默认地址
		defaultAddr := addresses[0]

		// 2. 创建秒杀订单
		orderNo, payAmount, err := svc.CreateOrder(ctx, service.CreateOrderReq{
			BuyerID:  evt.UserId,
			TenantID: evt.TenantId,
			Items: []service.CreateOrderItemReq{
				{SKUID: evt.SkuId, Quantity: 1},
			},
			AddressID:    defaultAddr.GetId(),
			CouponID:     0, // 秒杀不使用优惠券
			Remark:       "秒杀订单",
			Channel:      "mock",
			IsSeckill:    true,
			SeckillPrice: evt.SeckillPrice,
		})
		if err != nil {
			l.Error("创建秒杀订单失败",
				logger.Int64("userId", evt.UserId),
				logger.Int64("skuId", evt.SkuId),
				logger.Error(err))
			return err
		}
		l.Info("秒杀订单创建成功",
			logger.String("orderNo", orderNo),
			logger.Int64("payAmount", payAmount),
			logger.Int64("userId", evt.UserId))
		return nil
	})
}

func InitConsumers(
	paidConsumer *events.OrderPaidConsumer,
	closeDelayConsumer *events.OrderCloseDelayConsumer,
	seckillConsumer *events.SeckillSuccessConsumer,
) []saramax.Consumer {
	return []saramax.Consumer{paidConsumer, closeDelayConsumer, seckillConsumer}
}
