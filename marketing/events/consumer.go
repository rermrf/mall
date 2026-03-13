package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/saramax"
)

// OrderCancelledConsumer 消费 order_cancelled 事件，释放优惠券
type OrderCancelledConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt OrderCancelledEvent) error
}

func NewOrderCancelledConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	handler func(ctx context.Context, evt OrderCancelledEvent) error,
) *OrderCancelledConsumer {
	return &OrderCancelledConsumer{client: client, l: l, handler: handler}
}

func (c *OrderCancelledConsumer) Start() error {
	h := saramax.NewHandler[OrderCancelledEvent](c.l, c.Consume)
	go func() {
		for {
			err := c.client.Consume(context.Background(), []string{TopicOrderCancelled}, h)
			if err != nil {
				c.l.Error("消费 order_cancelled 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderCancelledConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderCancelledEvent) error {
	return c.handler(context.Background(), evt)
}
