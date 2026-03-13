package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CartCache interface {
	Get(ctx context.Context, userId int64) ([]byte, error)
	Set(ctx context.Context, userId int64, data []byte) error
	Delete(ctx context.Context, userId int64) error
}

type RedisCartCache struct {
	client redis.Cmdable
}

func NewCartCache(client redis.Cmdable) CartCache {
	return &RedisCartCache{client: client}
}

func cartKey(userId int64) string {
	return fmt.Sprintf("cart:items:%d", userId)
}

func (c *RedisCartCache) Get(ctx context.Context, userId int64) ([]byte, error) {
	return c.client.Get(ctx, cartKey(userId)).Bytes()
}

func (c *RedisCartCache) Set(ctx context.Context, userId int64, data []byte) error {
	return c.client.Set(ctx, cartKey(userId), data, 30*time.Minute).Err()
}

func (c *RedisCartCache) Delete(ctx context.Context, userId int64) error {
	return c.client.Del(ctx, cartKey(userId)).Err()
}
