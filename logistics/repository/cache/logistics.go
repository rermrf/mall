package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ==================== Cached Models ====================

type CachedTemplate struct {
	ID            int64              `json:"id"`
	TenantID      int64              `json:"tenantId"`
	Name          string             `json:"name"`
	ChargeType    int32              `json:"chargeType"`
	FreeThreshold int64              `json:"freeThreshold"`
	Rules         []CachedFreightRule `json:"rules"`
}

type CachedFreightRule struct {
	ID              int64  `json:"id"`
	TemplateID      int64  `json:"templateId"`
	Regions         string `json:"regions"`
	FirstUnit       int32  `json:"firstUnit"`
	FirstPrice      int64  `json:"firstPrice"`
	AdditionalUnit  int32  `json:"additionalUnit"`
	AdditionalPrice int64  `json:"additionalPrice"`
}

// ==================== Interface ====================

type LogisticsCache interface {
	GetTemplates(ctx context.Context, tenantId int64) ([]CachedTemplate, error)
	SetTemplates(ctx context.Context, tenantId int64, templates []CachedTemplate) error
	DeleteTemplates(ctx context.Context, tenantId int64) error
}

// ==================== Implementation ====================

type RedisLogisticsCache struct {
	client redis.Cmdable
}

func NewLogisticsCache(client redis.Cmdable) LogisticsCache {
	return &RedisLogisticsCache{client: client}
}

func templatesKey(tenantId int64) string {
	return fmt.Sprintf("logistics:templates:%d", tenantId)
}

func (c *RedisLogisticsCache) GetTemplates(ctx context.Context, tenantId int64) ([]CachedTemplate, error) {
	data, err := c.client.Get(ctx, templatesKey(tenantId)).Bytes()
	if err != nil {
		return nil, err
	}
	var templates []CachedTemplate
	err = json.Unmarshal(data, &templates)
	return templates, err
}

func (c *RedisLogisticsCache) SetTemplates(ctx context.Context, tenantId int64, templates []CachedTemplate) error {
	data, err := json.Marshal(templates)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, templatesKey(tenantId), data, 30*time.Minute).Err()
}

func (c *RedisLogisticsCache) DeleteTemplates(ctx context.Context, tenantId int64) error {
	return c.client.Del(ctx, templatesKey(tenantId)).Err()
}
