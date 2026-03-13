package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/saramax"
)

// ==================== UserRegisteredConsumer ====================

type UserRegisteredConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt UserRegisteredEvent) error
}

func NewUserRegisteredConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt UserRegisteredEvent) error) *UserRegisteredConsumer {
	return &UserRegisteredConsumer{client: client, l: l, handler: handler}
}

func (c *UserRegisteredConsumer) Start() error {
	h := saramax.NewHandler[UserRegisteredEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicUserRegistered}, h); err != nil {
				c.l.Error("消费 user_registered 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *UserRegisteredConsumer) Consume(msg *sarama.ConsumerMessage, evt UserRegisteredEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== OrderPaidConsumer ====================

type OrderPaidConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt OrderPaidEvent) error
}

func NewOrderPaidConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt OrderPaidEvent) error) *OrderPaidConsumer {
	return &OrderPaidConsumer{client: client, l: l, handler: handler}
}

func (c *OrderPaidConsumer) Start() error {
	h := saramax.NewHandler[OrderPaidEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicOrderPaid}, h); err != nil {
				c.l.Error("消费 order_paid 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderPaidConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderPaidEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== OrderShippedConsumer ====================

type OrderShippedConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt OrderShippedEvent) error
}

func NewOrderShippedConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt OrderShippedEvent) error) *OrderShippedConsumer {
	return &OrderShippedConsumer{client: client, l: l, handler: handler}
}

func (c *OrderShippedConsumer) Start() error {
	h := saramax.NewHandler[OrderShippedEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicOrderShipped}, h); err != nil {
				c.l.Error("消费 order_shipped 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderShippedConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderShippedEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== InventoryAlertConsumer ====================

type InventoryAlertConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt InventoryAlertEvent) error
}

func NewInventoryAlertConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt InventoryAlertEvent) error) *InventoryAlertConsumer {
	return &InventoryAlertConsumer{client: client, l: l, handler: handler}
}

func (c *InventoryAlertConsumer) Start() error {
	h := saramax.NewHandler[InventoryAlertEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicInventoryAlert}, h); err != nil {
				c.l.Error("消费 inventory_alert 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *InventoryAlertConsumer) Consume(msg *sarama.ConsumerMessage, evt InventoryAlertEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== TenantApprovedConsumer ====================

type TenantApprovedConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt TenantApprovedEvent) error
}

func NewTenantApprovedConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt TenantApprovedEvent) error) *TenantApprovedConsumer {
	return &TenantApprovedConsumer{client: client, l: l, handler: handler}
}

func (c *TenantApprovedConsumer) Start() error {
	h := saramax.NewHandler[TenantApprovedEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicTenantApproved}, h); err != nil {
				c.l.Error("消费 tenant_approved 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *TenantApprovedConsumer) Consume(msg *sarama.ConsumerMessage, evt TenantApprovedEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== TenantPlanChangedConsumer ====================

type TenantPlanChangedConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt TenantPlanChangedEvent) error
}

func NewTenantPlanChangedConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt TenantPlanChangedEvent) error) *TenantPlanChangedConsumer {
	return &TenantPlanChangedConsumer{client: client, l: l, handler: handler}
}

func (c *TenantPlanChangedConsumer) Start() error {
	h := saramax.NewHandler[TenantPlanChangedEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicTenantPlanChanged}, h); err != nil {
				c.l.Error("消费 tenant_plan_changed 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *TenantPlanChangedConsumer) Consume(msg *sarama.ConsumerMessage, evt TenantPlanChangedEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== OrderCompletedConsumer ====================

type OrderCompletedConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt OrderCompletedEvent) error
}

func NewOrderCompletedConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt OrderCompletedEvent) error) *OrderCompletedConsumer {
	return &OrderCompletedConsumer{client: client, l: l, handler: handler}
}

func (c *OrderCompletedConsumer) Start() error {
	h := saramax.NewHandler[OrderCompletedEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicOrderCompleted}, h); err != nil {
				c.l.Error("消费 order_completed 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderCompletedConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderCompletedEvent) error {
	return c.handler(context.Background(), evt)
}
