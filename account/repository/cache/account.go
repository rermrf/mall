package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rermrf/mall/account/repository/dao"
)

type AccountCache interface {
	GetAccount(ctx context.Context, tenantId int64) (dao.MerchantAccountModel, error)
	SetAccount(ctx context.Context, account dao.MerchantAccountModel) error
	DeleteAccount(ctx context.Context, tenantId int64) error
}

type RedisAccountCache struct {
	rdb redis.Cmdable
}

func NewAccountCache(rdb redis.Cmdable) AccountCache {
	return &RedisAccountCache{rdb: rdb}
}

func (c *RedisAccountCache) key(tenantId int64) string {
	return fmt.Sprintf("account:info:%d", tenantId)
}

func (c *RedisAccountCache) GetAccount(ctx context.Context, tenantId int64) (dao.MerchantAccountModel, error) {
	var account dao.MerchantAccountModel
	data, err := c.rdb.Get(ctx, c.key(tenantId)).Bytes()
	if err != nil {
		return account, err
	}
	err = json.Unmarshal(data, &account)
	return account, err
}

func (c *RedisAccountCache) SetAccount(ctx context.Context, account dao.MerchantAccountModel) error {
	data, err := json.Marshal(account)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.key(account.TenantId), data, 15*time.Minute).Err()
}

func (c *RedisAccountCache) DeleteAccount(ctx context.Context, tenantId int64) error {
	return c.rdb.Del(ctx, c.key(tenantId)).Err()
}
