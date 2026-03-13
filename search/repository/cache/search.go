package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SearchCache interface {
	IncrHotWord(ctx context.Context, keyword string) error
	GetHotWords(ctx context.Context, limit int) ([]redis.Z, error)
	AddSearchHistory(ctx context.Context, userId int64, keyword string) error
	GetSearchHistory(ctx context.Context, userId int64, limit int) ([]string, error)
	ClearSearchHistory(ctx context.Context, userId int64) error
}

type RedisSearchCache struct {
	client redis.Cmdable
}

func NewSearchCache(client redis.Cmdable) SearchCache {
	return &RedisSearchCache{client: client}
}

const hotWordKey = "search:hot"

func historyKey(userId int64) string {
	return fmt.Sprintf("search:history:%d", userId)
}

func (c *RedisSearchCache) IncrHotWord(ctx context.Context, keyword string) error {
	return c.client.ZIncrBy(ctx, hotWordKey, 1, keyword).Err()
}

func (c *RedisSearchCache) GetHotWords(ctx context.Context, limit int) ([]redis.Z, error) {
	return c.client.ZRevRangeWithScores(ctx, hotWordKey, 0, int64(limit-1)).Result()
}

func (c *RedisSearchCache) AddSearchHistory(ctx context.Context, userId int64, keyword string) error {
	key := historyKey(userId)
	pipe := c.client.Pipeline()
	pipe.LRem(ctx, key, 0, keyword)
	pipe.LPush(ctx, key, keyword)
	pipe.LTrim(ctx, key, 0, 19)
	pipe.Expire(ctx, key, 30*24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisSearchCache) GetSearchHistory(ctx context.Context, userId int64, limit int) ([]string, error) {
	return c.client.LRange(ctx, historyKey(userId), 0, int64(limit-1)).Result()
}

func (c *RedisSearchCache) ClearSearchHistory(ctx context.Context, userId int64) error {
	return c.client.Del(ctx, historyKey(userId)).Err()
}
