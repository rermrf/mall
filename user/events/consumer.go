package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/saramax"
)

// UserRegisteredConsumer 消费 user_registered 事件
type UserRegisteredConsumer struct {
	client sarama.ConsumerGroup
	l      logger.Logger
}

func NewUserRegisteredConsumer(client sarama.ConsumerGroup, l logger.Logger) *UserRegisteredConsumer {
	return &UserRegisteredConsumer{
		client: client,
		l:      l,
	}
}

// Start 实现 saramax.Consumer 接口
func (c *UserRegisteredConsumer) Start() error {
	cg := c.client
	handler := saramax.NewHandler[UserRegisteredEvent](c.l, c.Consume)
	go func() {
		for {
			err := cg.Consume(context.Background(), []string{TopicUserRegistered}, handler)
			if err != nil {
				c.l.Error("消费 user_registered 事件出错",
					logger.Error(err),
				)
				// 退避重试，避免日志刷屏
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

// Consume 处理单条 user_registered 事件
func (c *UserRegisteredConsumer) Consume(msg *sarama.ConsumerMessage, evt UserRegisteredEvent) error {
	c.l.Info("收到用户注册事件",
		logger.Int64("userId", evt.UserId),
		logger.Int64("tenantId", evt.TenantId),
		logger.String("phone", evt.Phone),
	)
	// user-svc 内部消费：预留用于初始化用户默认数据（如偏好设置等）
	// 主要消费者为 notification-svc（发送欢迎通知）
	return nil
}
