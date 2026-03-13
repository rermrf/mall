package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rermrf/mall/tenant/domain"
)

var ErrKeyNotExist = redis.Nil

type TenantCache interface {
	GetTenant(ctx context.Context, id int64) (domain.Tenant, error)
	SetTenant(ctx context.Context, t domain.Tenant) error
	DeleteTenant(ctx context.Context, id int64) error

	GetShop(ctx context.Context, tenantId int64) (domain.Shop, error)
	SetShop(ctx context.Context, s domain.Shop) error
	DeleteShop(ctx context.Context, tenantId int64) error

	GetShopByDomain(ctx context.Context, domain string) (domain.Shop, error)
	SetShopByDomain(ctx context.Context, domainName string, s domain.Shop) error
	DeleteShopByDomain(ctx context.Context, domainName string) error

	GetQuota(ctx context.Context, tenantId int64, quotaType string) (domain.QuotaUsage, error)
	SetQuota(ctx context.Context, tenantId int64, quotaType string, q domain.QuotaUsage) error
	DeleteQuota(ctx context.Context, tenantId int64, quotaType string) error
}

type RedisTenantCache struct {
	cmd redis.Cmdable
}

func NewTenantCache(cmd redis.Cmdable) TenantCache {
	return &RedisTenantCache{cmd: cmd}
}

func (c *RedisTenantCache) tenantKey(id int64) string {
	return fmt.Sprintf("tenant:info:%d", id)
}

func (c *RedisTenantCache) shopKey(tenantId int64) string {
	return fmt.Sprintf("shop:info:%d", tenantId)
}

func (c *RedisTenantCache) shopDomainKey(domain string) string {
	return fmt.Sprintf("shop:domain:%s", domain)
}

func (c *RedisTenantCache) quotaKey(tenantId int64, quotaType string) string {
	return fmt.Sprintf("tenant:quota:%d:%s", tenantId, quotaType)
}

// ==================== Tenant ====================

func (c *RedisTenantCache) GetTenant(ctx context.Context, id int64) (domain.Tenant, error) {
	val, err := c.cmd.Get(ctx, c.tenantKey(id)).Result()
	if err != nil {
		return domain.Tenant{}, err
	}
	var t domain.Tenant
	err = json.Unmarshal([]byte(val), &t)
	return t, err
}

func (c *RedisTenantCache) SetTenant(ctx context.Context, t domain.Tenant) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return c.cmd.Set(ctx, c.tenantKey(t.ID), data, 30*time.Minute).Err()
}

func (c *RedisTenantCache) DeleteTenant(ctx context.Context, id int64) error {
	return c.cmd.Del(ctx, c.tenantKey(id)).Err()
}

// ==================== Shop ====================

func (c *RedisTenantCache) GetShop(ctx context.Context, tenantId int64) (domain.Shop, error) {
	val, err := c.cmd.Get(ctx, c.shopKey(tenantId)).Result()
	if err != nil {
		return domain.Shop{}, err
	}
	var s domain.Shop
	err = json.Unmarshal([]byte(val), &s)
	return s, err
}

func (c *RedisTenantCache) SetShop(ctx context.Context, s domain.Shop) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return c.cmd.Set(ctx, c.shopKey(s.TenantID), data, 15*time.Minute).Err()
}

func (c *RedisTenantCache) DeleteShop(ctx context.Context, tenantId int64) error {
	return c.cmd.Del(ctx, c.shopKey(tenantId)).Err()
}

// ==================== Shop by Domain ====================

func (c *RedisTenantCache) GetShopByDomain(ctx context.Context, domainName string) (domain.Shop, error) {
	val, err := c.cmd.Get(ctx, c.shopDomainKey(domainName)).Result()
	if err != nil {
		return domain.Shop{}, err
	}
	var s domain.Shop
	err = json.Unmarshal([]byte(val), &s)
	return s, err
}

func (c *RedisTenantCache) SetShopByDomain(ctx context.Context, domainName string, s domain.Shop) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return c.cmd.Set(ctx, c.shopDomainKey(domainName), data, 15*time.Minute).Err()
}

func (c *RedisTenantCache) DeleteShopByDomain(ctx context.Context, domainName string) error {
	return c.cmd.Del(ctx, c.shopDomainKey(domainName)).Err()
}

// ==================== Quota ====================

func (c *RedisTenantCache) GetQuota(ctx context.Context, tenantId int64, quotaType string) (domain.QuotaUsage, error) {
	val, err := c.cmd.Get(ctx, c.quotaKey(tenantId, quotaType)).Result()
	if err != nil {
		return domain.QuotaUsage{}, err
	}
	var q domain.QuotaUsage
	err = json.Unmarshal([]byte(val), &q)
	return q, err
}

func (c *RedisTenantCache) SetQuota(ctx context.Context, tenantId int64, quotaType string, q domain.QuotaUsage) error {
	data, err := json.Marshal(q)
	if err != nil {
		return err
	}
	return c.cmd.Set(ctx, c.quotaKey(tenantId, quotaType), data, 10*time.Minute).Err()
}

func (c *RedisTenantCache) DeleteQuota(ctx context.Context, tenantId int64, quotaType string) error {
	return c.cmd.Del(ctx, c.quotaKey(tenantId, quotaType)).Err()
}
