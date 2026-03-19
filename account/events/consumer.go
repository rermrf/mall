package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/saramax"
)

type Consumer interface {
	Start() error
}

// OrderCompletedConsumer 消费 order-svc 的订单完成事件
type OrderCompletedConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt OrderCompletedEvent) error
}

func NewOrderCompletedConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	handler func(ctx context.Context, evt OrderCompletedEvent) error,
) *OrderCompletedConsumer {
	return &OrderCompletedConsumer{client: client, l: l, handler: handler}
}

func (c *OrderCompletedConsumer) Start() error {
	cg := c.client
	h := saramax.NewHandler[OrderCompletedEvent](c.l, c.Consume)
	go func() {
		for {
			err := cg.Consume(context.Background(), []string{TopicOrderCompleted}, h)
			if err != nil {
				c.l.Error("消费 order_completed 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderCompletedConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderCompletedEvent) error {
	c.l.Info("收到订单完成事件", logger.String("orderNo", evt.OrderNo))
	return c.handler(context.Background(), evt)
}
