package events

import (
	"context"
	"strconv"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/saramax"
)

type DeductExpireConsumer struct {
	client     sarama.ConsumerGroup
	l          logger.Logger
	rollbackFn func(ctx context.Context, orderId int64) error
}

func NewDeductExpireConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	rollbackFn func(ctx context.Context, orderId int64) error,
) *DeductExpireConsumer {
	return &DeductExpireConsumer{
		client:     client,
		l:          l,
		rollbackFn: rollbackFn,
	}
}

func (c *DeductExpireConsumer) Start() error {
	cg := c.client
	handler := saramax.NewHandler[DeductExpireEvent](c.l, c.Consume)
	go func() {
		for {
			err := cg.Consume(context.Background(), []string{TopicDeductExpire}, handler)
			if err != nil {
				c.l.Error("消费 inventory_deduct_expire 事件出错",
					logger.Error(err),
				)
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *DeductExpireConsumer) Consume(msg *sarama.ConsumerMessage, evt DeductExpireEvent) error {
	orderId, err := strconv.ParseInt(evt.Key, 10, 64)
	if err != nil {
		c.l.Error("解析 orderId 失败",
			logger.Error(err),
			logger.String("key", evt.Key),
		)
		return err
	}
	c.l.Info("收到库存预扣超时事件",
		logger.Int64("orderId", orderId),
	)
	return c.rollbackFn(context.Background(), orderId)
}
