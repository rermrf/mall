package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rermrf/mall/product/domain"
)

type ProductCache interface {
	GetProduct(ctx context.Context, id int64) (domain.Product, error)
	SetProduct(ctx context.Context, p domain.Product) error
	DeleteProduct(ctx context.Context, id int64) error

	GetCategoryTree(ctx context.Context, tenantId int64) ([]domain.Category, error)
	SetCategoryTree(ctx context.Context, tenantId int64, tree []domain.Category) error
	DeleteCategoryTree(ctx context.Context, tenantId int64) error
}

type RedisProductCache struct {
	cmd redis.Cmdable
}

func NewProductCache(cmd redis.Cmdable) ProductCache {
	return &RedisProductCache{cmd: cmd}
}

func (c *RedisProductCache) productKey(id int64) string {
	return fmt.Sprintf("product:info:%d", id)
}

func (c *RedisProductCache) categoryTreeKey(tenantId int64) string {
	return fmt.Sprintf("product:category:tree:%d", tenantId)
}

func (c *RedisProductCache) GetProduct(ctx context.Context, id int64) (domain.Product, error) {
	val, err := c.cmd.Get(ctx, c.productKey(id)).Result()
	if err != nil {
		return domain.Product{}, err
	}
	var p domain.Product
	err = json.Unmarshal([]byte(val), &p)
	return p, err
}

func (c *RedisProductCache) SetProduct(ctx context.Context, p domain.Product) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return c.cmd.Set(ctx, c.productKey(p.ID), data, 15*time.Minute).Err()
}

func (c *RedisProductCache) DeleteProduct(ctx context.Context, id int64) error {
	return c.cmd.Del(ctx, c.productKey(id)).Err()
}

func (c *RedisProductCache) GetCategoryTree(ctx context.Context, tenantId int64) ([]domain.Category, error) {
	val, err := c.cmd.Get(ctx, c.categoryTreeKey(tenantId)).Result()
	if err != nil {
		return nil, err
	}
	var tree []domain.Category
	err = json.Unmarshal([]byte(val), &tree)
	return tree, err
}

func (c *RedisProductCache) SetCategoryTree(ctx context.Context, tenantId int64, tree []domain.Category) error {
	data, err := json.Marshal(tree)
	if err != nil {
		return err
	}
	return c.cmd.Set(ctx, c.categoryTreeKey(tenantId), data, 30*time.Minute).Err()
}

func (c *RedisProductCache) DeleteCategoryTree(ctx context.Context, tenantId int64) error {
	return c.cmd.Del(ctx, c.categoryTreeKey(tenantId)).Err()
}
