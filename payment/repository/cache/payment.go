package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type PaymentCache interface {
	GetPayment(ctx context.Context, paymentNo string) ([]byte, error)
	SetPayment(ctx context.Context, paymentNo string, data []byte) error
	DeletePayment(ctx context.Context, paymentNo string) error
}

type RedisPaymentCache struct {
	client redis.Cmdable
}

func NewPaymentCache(client redis.Cmdable) PaymentCache {
	return &RedisPaymentCache{client: client}
}

func paymentKey(paymentNo string) string {
	return fmt.Sprintf("payment:info:%s", paymentNo)
}

func (c *RedisPaymentCache) GetPayment(ctx context.Context, paymentNo string) ([]byte, error) {
	return c.client.Get(ctx, paymentKey(paymentNo)).Bytes()
}

func (c *RedisPaymentCache) SetPayment(ctx context.Context, paymentNo string, data []byte) error {
	return c.client.Set(ctx, paymentKey(paymentNo), data, 15*time.Minute).Err()
}

func (c *RedisPaymentCache) DeletePayment(ctx context.Context, paymentNo string) error {
	return c.client.Del(ctx, paymentKey(paymentNo)).Err()
}
