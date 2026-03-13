package cache

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var (
	//go:embed lua/seckill.lua
	seckillLua string
)

type MarketingCache interface {
	// 优惠券库存
	SetCouponStock(ctx context.Context, couponId int64, stock int32) error
	DecrCouponStock(ctx context.Context, couponId int64) (int64, error)
	IncrCouponStock(ctx context.Context, couponId int64) error
	// 秒杀库存
	SetSeckillStock(ctx context.Context, itemId int64, stock int32) error
	Seckill(ctx context.Context, itemId, userId int64, perLimit int32) (int64, error)
}

type RedisMarketingCache struct {
	client redis.Cmdable
}

func NewMarketingCache(client redis.Cmdable) MarketingCache {
	return &RedisMarketingCache{client: client}
}

func couponStockKey(couponId int64) string {
	return fmt.Sprintf("coupon:stock:%d", couponId)
}

func seckillStockKey(itemId int64) string {
	return fmt.Sprintf("seckill:stock:%d", itemId)
}

func seckillUserKey(itemId int64) string {
	return fmt.Sprintf("seckill:user:%d", itemId)
}

func (c *RedisMarketingCache) SetCouponStock(ctx context.Context, couponId int64, stock int32) error {
	return c.client.Set(ctx, couponStockKey(couponId), stock, 0).Err()
}

func (c *RedisMarketingCache) DecrCouponStock(ctx context.Context, couponId int64) (int64, error) {
	return c.client.Decr(ctx, couponStockKey(couponId)).Result()
}

func (c *RedisMarketingCache) IncrCouponStock(ctx context.Context, couponId int64) error {
	return c.client.Incr(ctx, couponStockKey(couponId)).Err()
}

func (c *RedisMarketingCache) SetSeckillStock(ctx context.Context, itemId int64, stock int32) error {
	return c.client.Set(ctx, seckillStockKey(itemId), stock, 0).Err()
}

func (c *RedisMarketingCache) Seckill(ctx context.Context, itemId, userId int64, perLimit int32) (int64, error) {
	result, err := c.client.Eval(ctx, seckillLua,
		[]string{seckillStockKey(itemId), seckillUserKey(itemId)},
		strconv.FormatInt(userId, 10),
		strconv.FormatInt(int64(perLimit), 10),
	).Int64()
	if err != nil {
		return -1, err
	}
	return result, nil
}
