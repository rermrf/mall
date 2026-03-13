package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/saramax"
)

// OrderPaidConsumer 消费 payment-svc 的支付成功事件
type OrderPaidConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt OrderPaidEvent) error
}

func NewOrderPaidConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	handler func(ctx context.Context, evt OrderPaidEvent) error,
) *OrderPaidConsumer {
	return &OrderPaidConsumer{client: client, l: l, handler: handler}
}

func (c *OrderPaidConsumer) Start() error {
	cg := c.client
	h := saramax.NewHandler[OrderPaidEvent](c.l, c.Consume)
	go func() {
		for {
			err := cg.Consume(context.Background(), []string{TopicOrderPaid}, h)
			if err != nil {
				c.l.Error("消费 order_paid 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderPaidConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderPaidEvent) error {
	c.l.Info("收到支付成功事件", logger.String("orderNo", evt.OrderNo))
	return c.handler(context.Background(), evt)
}

// OrderCloseDelayConsumer 消费 go-delay 投递的超时关单事件
type OrderCloseDelayConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, orderNo string) error
}

func NewOrderCloseDelayConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	handler func(ctx context.Context, orderNo string) error,
) *OrderCloseDelayConsumer {
	return &OrderCloseDelayConsumer{client: client, l: l, handler: handler}
}

func (c *OrderCloseDelayConsumer) Start() error {
	cg := c.client
	h := saramax.NewHandler[OrderCloseDelayEvent](c.l, c.Consume)
	go func() {
		for {
			err := cg.Consume(context.Background(), []string{TopicOrderCloseDelay}, h)
			if err != nil {
				c.l.Error("消费 order_close_delay 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderCloseDelayConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderCloseDelayEvent) error {
	c.l.Info("收到超时关单事件", logger.String("orderNo", evt.Key))
	return c.handler(context.Background(), evt.Key)
}

// SeckillSuccessConsumer 消费 marketing-svc 的秒杀成功事件
type SeckillSuccessConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt SeckillSuccessEvent) error
}

func NewSeckillSuccessConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	handler func(ctx context.Context, evt SeckillSuccessEvent) error,
) *SeckillSuccessConsumer {
	return &SeckillSuccessConsumer{client: client, l: l, handler: handler}
}

func (c *SeckillSuccessConsumer) Start() error {
	cg := c.client
	h := saramax.NewHandler[SeckillSuccessEvent](c.l, c.Consume)
	go func() {
		for {
			err := cg.Consume(context.Background(), []string{TopicSeckillSuccess}, h)
			if err != nil {
				c.l.Error("消费 seckill_success 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *SeckillSuccessConsumer) Consume(msg *sarama.ConsumerMessage, evt SeckillSuccessEvent) error {
	c.l.Info("收到秒杀成功事件",
		logger.Int64("userId", evt.UserId),
		logger.Int64("skuId", evt.SkuId))
	return c.handler(context.Background(), evt)
}
