package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type OrderCache interface {
	GetOrder(ctx context.Context, orderNo string) ([]byte, error)
	SetOrder(ctx context.Context, orderNo string, data []byte) error
	DeleteOrder(ctx context.Context, orderNo string) error
}

type RedisOrderCache struct {
	client redis.Cmdable
}

func NewOrderCache(client redis.Cmdable) OrderCache {
	return &RedisOrderCache{client: client}
}

func orderKey(orderNo string) string {
	return fmt.Sprintf("order:info:%s", orderNo)
}

func (c *RedisOrderCache) GetOrder(ctx context.Context, orderNo string) ([]byte, error) {
	data, err := c.client.Get(ctx, orderKey(orderNo)).Bytes()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *RedisOrderCache) SetOrder(ctx context.Context, orderNo string, data []byte) error {
	return c.client.Set(ctx, orderKey(orderNo), data, 15*time.Minute).Err()
}

func (c *RedisOrderCache) DeleteOrder(ctx context.Context, orderNo string) error {
	return c.client.Del(ctx, orderKey(orderNo)).Err()
}

// OrderCacheData 缓存数据结构（用于序列化）
type OrderCacheData struct {
	Order json.RawMessage `json:"order"`
	Items json.RawMessage `json:"items"`
}
