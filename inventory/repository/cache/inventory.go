package cache

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	//go:embed lua/deduct.lua
	deductLua string
	//go:embed lua/rollback.lua
	rollbackLua string
)

type InventoryCache interface {
	SetStock(ctx context.Context, skuId int64, total, available, locked, sold, alertThreshold int32) error
	GetStock(ctx context.Context, skuId int64) (total, available, locked, sold, alertThreshold int32, err error)
	Deduct(ctx context.Context, items map[int64]int32) (bool, string, error)
	Rollback(ctx context.Context, items map[int64]int32) error
	SetDeductRecord(ctx context.Context, orderId int64, items map[int64]int32) error
	GetDeductRecord(ctx context.Context, orderId int64) (map[int64]int32, error)
	DeleteDeductRecord(ctx context.Context, orderId int64) error
	Exists(ctx context.Context, skuId int64) (bool, error)
}

type RedisInventoryCache struct {
	client redis.Cmdable
}

func NewInventoryCache(client redis.Cmdable) InventoryCache {
	return &RedisInventoryCache{client: client}
}

func stockKey(skuId int64) string {
	return fmt.Sprintf("inventory:stock:%d", skuId)
}

func deductKey(orderId int64) string {
	return fmt.Sprintf("inventory:deduct:%d", orderId)
}

func (c *RedisInventoryCache) SetStock(ctx context.Context, skuId int64, total, available, locked, sold, alertThreshold int32) error {
	key := stockKey(skuId)
	return c.client.HSet(ctx, key, map[string]any{
		"total":           total,
		"available":       available,
		"locked":          locked,
		"sold":            sold,
		"alert_threshold": alertThreshold,
	}).Err()
}

func (c *RedisInventoryCache) GetStock(ctx context.Context, skuId int64) (total, available, locked, sold, alertThreshold int32, err error) {
	key := stockKey(skuId)
	result, err := c.client.HGetAll(ctx, key).Result()
	if err != nil {
		return
	}
	if len(result) == 0 {
		err = redis.Nil
		return
	}
	t, _ := strconv.ParseInt(result["total"], 10, 32)
	a, _ := strconv.ParseInt(result["available"], 10, 32)
	l, _ := strconv.ParseInt(result["locked"], 10, 32)
	s, _ := strconv.ParseInt(result["sold"], 10, 32)
	at, _ := strconv.ParseInt(result["alert_threshold"], 10, 32)
	return int32(t), int32(a), int32(l), int32(s), int32(at), nil
}

func (c *RedisInventoryCache) Deduct(ctx context.Context, items map[int64]int32) (bool, string, error) {
	keys := make([]string, 0, len(items))
	args := make([]any, 0, len(items))
	skuIds := make([]int64, 0, len(items))
	for skuId, qty := range items {
		keys = append(keys, stockKey(skuId))
		args = append(args, qty)
		skuIds = append(skuIds, skuId)
	}
	result, err := c.client.Eval(ctx, deductLua, keys, args...).Result()
	if err != nil {
		return false, "", err
	}
	idx, ok := result.(int64)
	if !ok {
		return false, "", fmt.Errorf("unexpected lua result type: %T", result)
	}
	if idx > 0 {
		failedSkuId := skuIds[idx-1]
		return false, fmt.Sprintf("SKU %d 库存不足", failedSkuId), nil
	}
	return true, "", nil
}

func (c *RedisInventoryCache) Rollback(ctx context.Context, items map[int64]int32) error {
	keys := make([]string, 0, len(items))
	args := make([]any, 0, len(items))
	for skuId, qty := range items {
		keys = append(keys, stockKey(skuId))
		args = append(args, qty)
	}
	_, err := c.client.Eval(ctx, rollbackLua, keys, args...).Result()
	return err
}

func (c *RedisInventoryCache) SetDeductRecord(ctx context.Context, orderId int64, items map[int64]int32) error {
	key := deductKey(orderId)
	fields := make(map[string]any, len(items))
	for skuId, qty := range items {
		fields[strconv.FormatInt(skuId, 10)] = qty
	}
	pipe := c.client.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, 35*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisInventoryCache) GetDeductRecord(ctx context.Context, orderId int64) (map[int64]int32, error) {
	key := deductKey(orderId)
	result, err := c.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	items := make(map[int64]int32, len(result))
	for k, v := range result {
		skuId, _ := strconv.ParseInt(k, 10, 64)
		qty, _ := strconv.ParseInt(v, 10, 32)
		items[skuId] = int32(qty)
	}
	return items, nil
}

func (c *RedisInventoryCache) DeleteDeductRecord(ctx context.Context, orderId int64) error {
	return c.client.Del(ctx, deductKey(orderId)).Err()
}

func (c *RedisInventoryCache) Exists(ctx context.Context, skuId int64) (bool, error) {
	n, err := c.client.Exists(ctx, stockKey(skuId)).Result()
	return n > 0, err
}
