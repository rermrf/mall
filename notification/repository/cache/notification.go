package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type NotificationCache interface {
	GetUnreadCount(ctx context.Context, userId int64) (int64, error)
	SetUnreadCount(ctx context.Context, userId int64, count int64) error
	DeleteUnreadCount(ctx context.Context, userId int64) error
}

type RedisNotificationCache struct {
	client redis.Cmdable
}

func NewNotificationCache(client redis.Cmdable) NotificationCache {
	return &RedisNotificationCache{client: client}
}

func unreadKey(userId int64) string {
	return fmt.Sprintf("notification:unread:%d", userId)
}

func (c *RedisNotificationCache) GetUnreadCount(ctx context.Context, userId int64) (int64, error) {
	val, err := c.client.Get(ctx, unreadKey(userId)).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

func (c *RedisNotificationCache) SetUnreadCount(ctx context.Context, userId int64, count int64) error {
	return c.client.Set(ctx, unreadKey(userId), count, 10*time.Minute).Err()
}

func (c *RedisNotificationCache) DeleteUnreadCount(ctx context.Context, userId int64) error {
	return c.client.Del(ctx, unreadKey(userId)).Err()
}
