# Inventory Service (inventory-svc) 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 SaaS 多租户商城的库存微服务，支持库存设置、查询和三阶段扣减（TCC: Deduct/Confirm/Rollback），核心亮点为 Redis Lua 原子预扣 + go-delay 延迟消息超时回滚。

**Architecture:** DDD 分层（domain → dao/cache → repository → service → grpc），沿用项目统一风格。预扣使用 Redis Lua 脚本保证原子性，预扣记录存 Redis Hash（不入 MySQL），超时回滚通过 go-delay 延迟消息服务投递到 Kafka，inventory-svc 自身消费触发 Rollback。

**Tech Stack:** Go, gRPC, GORM/MySQL, Redis (Lua scripts), Kafka/Sarama, Wire DI, Viper, etcd, go-delay

---

## 参考文件

| 文件 | 用途 |
|------|------|
| `docs/plans/2026-03-07-inventory-svc-design.md` | 设计文档 |
| `api/proto/gen/inventory/v1/inventory.pb.go` | Proto 消息类型 |
| `api/proto/gen/inventory/v1/inventory_grpc.pb.go` | gRPC 服务接口（7 RPC） |
| `product/ioc/*.go` | IoC 模式参考 |
| `product/events/producer.go` | Kafka Producer 模式参考 |
| `user/events/consumer.go` | Kafka Consumer 模式参考（saramax.Handler[T]） |
| `user/ioc/kafka.go` | Consumer 初始化参考 |
| `user/app.go` | App 聚合（Server + Consumers）参考 |
| `user/wire.go` | Wire DI 参考（含 Consumer） |

---

## Proto 消息类型速查

```
inventoryv1.Inventory:
  Id, TenantId, SkuId int64; Total, Available, Locked, Sold, AlertThreshold int32; Ctime, Utime int64

inventoryv1.DeductItem:
  SkuId int64; Quantity int32

inventoryv1.InventoryLog:
  Id, SkuId, OrderId int64; Type, Quantity, BeforeAvailable, AfterAvailable int32; Ctime, TenantId int64

RPC:
  SetStock(SetStockRequest{TenantId, SkuId int64, Total, AlertThreshold int32}) → SetStockResponse{}
  GetStock(GetStockRequest{SkuId int64}) → GetStockResponse{Inventory}
  BatchGetStock(BatchGetStockRequest{SkuIds []int64}) → BatchGetStockResponse{Inventories []*Inventory}
  Deduct(DeductRequest{OrderId, TenantId int64, Items []*DeductItem}) → DeductResponse{Success bool, Message string}
  Confirm(ConfirmRequest{OrderId int64}) → ConfirmResponse{}
  Rollback(RollbackRequest{OrderId int64}) → RollbackResponse{}
  ListLogs(ListLogsRequest{TenantId, SkuId int64, Page, PageSize int32}) → ListLogsResponse{Logs []*InventoryLog, Total int64}
```

---

## Task 1: Domain 层

**Files:**
- Create: `inventory/domain/inventory.go`

**说明：** 定义 Inventory、DeductRecord、DeductItem、InventoryLog 领域实体和类型枚举。

```go
package domain

import "time"

type Inventory struct {
	ID             int64
	TenantID       int64
	SKUID          int64
	Total          int32
	Available      int32
	Locked         int32
	Sold           int32
	AlertThreshold int32
	Ctime          time.Time
	Utime          time.Time
}

type DeductRecord struct {
	OrderID  int64
	TenantID int64
	Items    []DeductItem
}

type DeductItem struct {
	SKUID    int64
	Quantity int32
}

type InventoryLog struct {
	ID              int64
	SKUID           int64
	OrderID         int64
	Type            LogType
	Quantity        int32
	BeforeAvailable int32
	AfterAvailable  int32
	TenantID        int64
	Ctime           time.Time
}

type LogType uint8

const (
	LogTypeDeduct   LogType = 1
	LogTypeConfirm  LogType = 2
	LogTypeRollback LogType = 3
	LogTypeManual   LogType = 4
)
```

---

## Task 2: DAO 层

**Files:**
- Create: `inventory/repository/dao/inventory.go`
- Create: `inventory/repository/dao/init.go`

### 2.1 inventory/repository/dao/inventory.go

GORM 模型 2 个 + DAO 接口 1 个。

```go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InventoryModel 库存表
type InventoryModel struct {
	ID             int64 `gorm:"primaryKey;autoIncrement"`
	TenantId       int64 `gorm:"uniqueIndex:uk_tenant_sku"`
	SkuId          int64 `gorm:"uniqueIndex:uk_tenant_sku"`
	Total          int32
	Available      int32
	Locked         int32
	Sold           int32
	AlertThreshold int32
	Ctime          int64
	Utime          int64
}

func (InventoryModel) TableName() string {
	return "inventories"
}

// InventoryLogModel 库存变更日志表
type InventoryLogModel struct {
	ID              int64 `gorm:"primaryKey;autoIncrement"`
	SkuId           int64 `gorm:"index:idx_sku"`
	OrderId         int64 `gorm:"index:idx_order"`
	Type            uint8
	Quantity        int32
	BeforeAvailable int32
	AfterAvailable  int32
	TenantId        int64
	Ctime           int64
}

func (InventoryLogModel) TableName() string {
	return "inventory_logs"
}

type InventoryDAO interface {
	Upsert(ctx context.Context, inv InventoryModel) error
	FindBySKUID(ctx context.Context, skuId int64) (InventoryModel, error)
	FindBySKUIDs(ctx context.Context, skuIds []int64) ([]InventoryModel, error)
	UpdateStockByConfirm(ctx context.Context, tx *gorm.DB, skuId int64, qty int32) error
	InsertLog(ctx context.Context, tx *gorm.DB, log InventoryLogModel) error
	ListLogs(ctx context.Context, tenantId, skuId int64, offset, limit int) ([]InventoryLogModel, int64, error)
	GetDB() *gorm.DB
}

type GORMInventoryDAO struct {
	db *gorm.DB
}

func NewInventoryDAO(db *gorm.DB) InventoryDAO {
	return &GORMInventoryDAO{db: db}
}

func (d *GORMInventoryDAO) GetDB() *gorm.DB {
	return d.db
}

func (d *GORMInventoryDAO) Upsert(ctx context.Context, inv InventoryModel) error {
	now := time.Now().UnixMilli()
	inv.Ctime = now
	inv.Utime = now
	// available = total - locked - sold（Upsert 时重新计算 available）
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "sku_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"total":           inv.Total,
			"available":       gorm.Expr("? - locked - sold", inv.Total),
			"alert_threshold": inv.AlertThreshold,
			"utime":           now,
		}),
	}).Create(&inv).Error
}

func (d *GORMInventoryDAO) FindBySKUID(ctx context.Context, skuId int64) (InventoryModel, error) {
	var inv InventoryModel
	err := d.db.WithContext(ctx).Where("sku_id = ?", skuId).First(&inv).Error
	return inv, err
}

func (d *GORMInventoryDAO) FindBySKUIDs(ctx context.Context, skuIds []int64) ([]InventoryModel, error) {
	var invs []InventoryModel
	err := d.db.WithContext(ctx).Where("sku_id IN ?", skuIds).Find(&invs).Error
	return invs, err
}

// UpdateStockByConfirm 确认扣减：locked -= qty, sold += qty
func (d *GORMInventoryDAO) UpdateStockByConfirm(ctx context.Context, tx *gorm.DB, skuId int64, qty int32) error {
	return tx.WithContext(ctx).Model(&InventoryModel{}).
		Where("sku_id = ? AND locked >= ?", skuId, qty).
		Updates(map[string]any{
			"locked": gorm.Expr("locked - ?", qty),
			"sold":   gorm.Expr("sold + ?", qty),
			"utime":  time.Now().UnixMilli(),
		}).Error
}

func (d *GORMInventoryDAO) InsertLog(ctx context.Context, tx *gorm.DB, log InventoryLogModel) error {
	log.Ctime = time.Now().UnixMilli()
	return tx.WithContext(ctx).Create(&log).Error
}

func (d *GORMInventoryDAO) ListLogs(ctx context.Context, tenantId, skuId int64, offset, limit int) ([]InventoryLogModel, int64, error) {
	var logs []InventoryLogModel
	var total int64
	query := d.db.WithContext(ctx).Model(&InventoryLogModel{}).Where("tenant_id = ?", tenantId)
	if skuId > 0 {
		query = query.Where("sku_id = ?", skuId)
	}
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = query.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}
```

### 2.2 inventory/repository/dao/init.go

```go
package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&InventoryModel{},
		&InventoryLogModel{},
	)
}
```

---

## Task 3: Cache 层（Redis Hash + Lua 脚本）

**Files:**
- Create: `inventory/repository/cache/inventory.go`

**说明：** 这是本服务的核心亮点。包含 Redis Hash 读写、Lua 原子预扣脚本和回滚脚本。

```go
package cache

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var (
	//go:embed lua/deduct.lua
	deductLua string
	//go:embed lua/rollback.lua
	rollbackLua string
)

type InventoryCache interface {
	// SetStock 设置库存 Hash
	SetStock(ctx context.Context, skuId int64, total, available, locked, sold, alertThreshold int32) error
	// GetStock 读取库存 Hash
	GetStock(ctx context.Context, skuId int64) (total, available, locked, sold, alertThreshold int32, err error)
	// Deduct Redis Lua 原子预扣（多 SKU）
	Deduct(ctx context.Context, items map[int64]int32) (bool, string, error)
	// Rollback Redis Lua 原子回滚（多 SKU）
	Rollback(ctx context.Context, items map[int64]int32) error
	// SetDeductRecord 存储预扣记录
	SetDeductRecord(ctx context.Context, orderId int64, items map[int64]int32) error
	// GetDeductRecord 获取预扣记录
	GetDeductRecord(ctx context.Context, orderId int64) (map[int64]int32, error)
	// DeleteDeductRecord 删除预扣记录
	DeleteDeductRecord(ctx context.Context, orderId int64) error
	// Exists 检查库存 Hash 是否存在
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
	// 构建 Lua 参数：KEYS = [stock_key1, stock_key2, ...], ARGV = [qty1, qty2, ...]
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
	// Lua 返回：0 = 成功, >0 = 失败的 SKU 索引（1-based）
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
	pipe.Expire(ctx, key, 35*60*1e9) // 35 minutes
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
```

### Lua 脚本

**Create: `inventory/repository/cache/lua/deduct.lua`**

```lua
-- deduct.lua: 多 SKU 原子预扣
-- KEYS: [stock_key1, stock_key2, ...]
-- ARGV: [qty1, qty2, ...]
-- 返回: 0=成功, >0=失败的 key 索引(1-based)

local n = #KEYS
-- 第一轮：检查所有 SKU 库存是否充足
for i = 1, n do
    local available = tonumber(redis.call('HGET', KEYS[i], 'available') or 0)
    local qty = tonumber(ARGV[i])
    if available < qty then
        return i
    end
end
-- 第二轮：执行扣减
for i = 1, n do
    local qty = tonumber(ARGV[i])
    redis.call('HINCRBY', KEYS[i], 'available', -qty)
    redis.call('HINCRBY', KEYS[i], 'locked', qty)
end
return 0
```

**Create: `inventory/repository/cache/lua/rollback.lua`**

```lua
-- rollback.lua: 多 SKU 原子回滚
-- KEYS: [stock_key1, stock_key2, ...]
-- ARGV: [qty1, qty2, ...]

local n = #KEYS
for i = 1, n do
    local qty = tonumber(ARGV[i])
    redis.call('HINCRBY', KEYS[i], 'available', qty)
    redis.call('HINCRBY', KEYS[i], 'locked', -qty)
end
return 0
```

---

## Task 4: Repository 层

**Files:**
- Create: `inventory/repository/inventory.go`

**说明：** 协调 MySQL DAO 和 Redis Cache，实现 Cache-Aside 模式。

```go
package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/rermrf/mall/inventory/domain"
	"github.com/rermrf/mall/inventory/repository/cache"
	"github.com/rermrf/mall/inventory/repository/dao"
)

type InventoryRepository interface {
	SetStock(ctx context.Context, inv domain.Inventory) error
	GetStock(ctx context.Context, skuId int64) (domain.Inventory, error)
	BatchGetStock(ctx context.Context, skuIds []int64) ([]domain.Inventory, error)
	Deduct(ctx context.Context, items map[int64]int32) (bool, string, error)
	GetDeductRecord(ctx context.Context, orderId int64) (map[int64]int32, error)
	SetDeductRecord(ctx context.Context, orderId int64, items map[int64]int32) error
	DeleteDeductRecord(ctx context.Context, orderId int64) error
	RollbackStock(ctx context.Context, items map[int64]int32) error
	ConfirmStock(ctx context.Context, items map[int64]int32, orderId, tenantId int64) error
	InsertLog(ctx context.Context, log domain.InventoryLog) error
	ListLogs(ctx context.Context, tenantId, skuId int64, page, pageSize int32) ([]domain.InventoryLog, int64, error)
}

type inventoryRepository struct {
	dao   dao.InventoryDAO
	cache cache.InventoryCache
}

func NewInventoryRepository(d dao.InventoryDAO, c cache.InventoryCache) InventoryRepository {
	return &inventoryRepository{dao: d, cache: c}
}

func (r *inventoryRepository) SetStock(ctx context.Context, inv domain.Inventory) error {
	// MySQL upsert
	err := r.dao.Upsert(ctx, r.domainToEntity(inv))
	if err != nil {
		return err
	}
	// 重新从 MySQL 读取（获取完整数据含 locked/sold）然后同步写 Redis
	entity, err := r.dao.FindBySKUID(ctx, inv.SKUID)
	if err != nil {
		return err
	}
	return r.cache.SetStock(ctx, entity.SkuId, entity.Total, entity.Available, entity.Locked, entity.Sold, entity.AlertThreshold)
}

func (r *inventoryRepository) GetStock(ctx context.Context, skuId int64) (domain.Inventory, error) {
	// 优先读 Redis
	total, available, locked, sold, alertThreshold, err := r.cache.GetStock(ctx, skuId)
	if err == nil {
		return domain.Inventory{
			SKUID:          skuId,
			Total:          total,
			Available:      available,
			Locked:         locked,
			Sold:           sold,
			AlertThreshold: alertThreshold,
		}, nil
	}
	if err != redis.Nil {
		return domain.Inventory{}, err
	}
	// Cache miss, 读 MySQL 回填
	entity, err := r.dao.FindBySKUID(ctx, skuId)
	if err != nil {
		return domain.Inventory{}, err
	}
	inv := r.entityToDomain(entity)
	// 回填 Redis
	_ = r.cache.SetStock(ctx, entity.SkuId, entity.Total, entity.Available, entity.Locked, entity.Sold, entity.AlertThreshold)
	return inv, nil
}

func (r *inventoryRepository) BatchGetStock(ctx context.Context, skuIds []int64) ([]domain.Inventory, error) {
	// 批量查询直接走 MySQL
	entities, err := r.dao.FindBySKUIDs(ctx, skuIds)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Inventory, 0, len(entities))
	for _, e := range entities {
		result = append(result, r.entityToDomain(e))
	}
	return result, nil
}

func (r *inventoryRepository) Deduct(ctx context.Context, items map[int64]int32) (bool, string, error) {
	return r.cache.Deduct(ctx, items)
}

func (r *inventoryRepository) GetDeductRecord(ctx context.Context, orderId int64) (map[int64]int32, error) {
	return r.cache.GetDeductRecord(ctx, orderId)
}

func (r *inventoryRepository) SetDeductRecord(ctx context.Context, orderId int64, items map[int64]int32) error {
	return r.cache.SetDeductRecord(ctx, orderId, items)
}

func (r *inventoryRepository) DeleteDeductRecord(ctx context.Context, orderId int64) error {
	return r.cache.DeleteDeductRecord(ctx, orderId)
}

func (r *inventoryRepository) RollbackStock(ctx context.Context, items map[int64]int32) error {
	return r.cache.Rollback(ctx, items)
}

// ConfirmStock MySQL 事务：更新 locked/sold + 写日志
func (r *inventoryRepository) ConfirmStock(ctx context.Context, items map[int64]int32, orderId, tenantId int64) error {
	db := r.dao.GetDB()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for skuId, qty := range items {
			// 获取 confirm 前的 available（从 MySQL 查）
			var inv dao.InventoryModel
			if err := tx.Where("sku_id = ?", skuId).First(&inv).Error; err != nil {
				return err
			}
			err := r.dao.UpdateStockByConfirm(ctx, tx, skuId, qty)
			if err != nil {
				return err
			}
			// 写确认日志
			err = r.dao.InsertLog(ctx, tx, dao.InventoryLogModel{
				SkuId:           skuId,
				OrderId:         orderId,
				Type:            uint8(domain.LogTypeConfirm),
				Quantity:        qty,
				BeforeAvailable: inv.Available,
				AfterAvailable:  inv.Available, // Confirm 不改 available，改 locked->sold
				TenantId:        tenantId,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *inventoryRepository) InsertLog(ctx context.Context, log domain.InventoryLog) error {
	db := r.dao.GetDB()
	return r.dao.InsertLog(ctx, db, dao.InventoryLogModel{
		SkuId:           log.SKUID,
		OrderId:         log.OrderID,
		Type:            uint8(log.Type),
		Quantity:        log.Quantity,
		BeforeAvailable: log.BeforeAvailable,
		AfterAvailable:  log.AfterAvailable,
		TenantId:        log.TenantID,
	})
}

func (r *inventoryRepository) ListLogs(ctx context.Context, tenantId, skuId int64, page, pageSize int32) ([]domain.InventoryLog, int64, error) {
	offset := int((page - 1) * pageSize)
	entities, total, err := r.dao.ListLogs(ctx, tenantId, skuId, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	logs := make([]domain.InventoryLog, 0, len(entities))
	for _, e := range entities {
		logs = append(logs, domain.InventoryLog{
			ID:              e.ID,
			SKUID:           e.SkuId,
			OrderID:         e.OrderId,
			Type:            domain.LogType(e.Type),
			Quantity:        e.Quantity,
			BeforeAvailable: e.BeforeAvailable,
			AfterAvailable:  e.AfterAvailable,
			TenantID:        e.TenantId,
			Ctime:           time.UnixMilli(e.Ctime),
		})
	}
	return logs, total, nil
}

func (r *inventoryRepository) domainToEntity(inv domain.Inventory) dao.InventoryModel {
	return dao.InventoryModel{
		ID:             inv.ID,
		TenantId:       inv.TenantID,
		SkuId:          inv.SKUID,
		Total:          inv.Total,
		Available:      inv.Available,
		Locked:         inv.Locked,
		Sold:           inv.Sold,
		AlertThreshold: inv.AlertThreshold,
	}
}

func (r *inventoryRepository) entityToDomain(e dao.InventoryModel) domain.Inventory {
	return domain.Inventory{
		ID:             e.ID,
		TenantID:       e.TenantId,
		SKUID:          e.SkuId,
		Total:          e.Total,
		Available:      e.Available,
		Locked:         e.Locked,
		Sold:           e.Sold,
		AlertThreshold: e.AlertThreshold,
		Ctime:          time.UnixMilli(e.Ctime),
		Utime:          time.UnixMilli(e.Utime),
	}
}
```

---

## Task 5: Events 层

**Files:**
- Create: `inventory/events/types.go`
- Create: `inventory/events/producer.go`
- Create: `inventory/events/consumer.go`

### 5.1 inventory/events/types.go

```go
package events

// DelayMessage go-delay 延迟消息格式
type DelayMessage struct {
	Biz      string `json:"biz"`
	Key      string `json:"key"`
	Payload  string `json:"payload,omitempty"`
	BizTopic string `json:"biz_topic"`
	ExecuteAt int64 `json:"execute_at"` // Unix 秒
}

// DeductExpireEvent inventory_deduct_expire 消费事件
type DeductExpireEvent struct {
	Biz      string `json:"biz"`
	Key      string `json:"key"`
	Payload  string `json:"payload,omitempty"`
	BizTopic string `json:"biz_topic"`
}

// InventoryAlertEvent 库存预警事件
type InventoryAlertEvent struct {
	TenantID  int64 `json:"tenant_id"`
	SKUID     int64 `json:"sku_id"`
	Available int32 `json:"available"`
	Threshold int32 `json:"threshold"`
}

const (
	TopicDelayMessage        = "delay_topic"
	TopicDeductExpire        = "inventory_deduct_expire"
	TopicInventoryAlert      = "inventory_alert"
)
```

### 5.2 inventory/events/producer.go

```go
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

type Producer interface {
	ProduceDelay(ctx context.Context, orderId int64) error
	ProduceAlert(ctx context.Context, evt InventoryAlertEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

// ProduceDelay 发送延迟消息到 go-delay（30 分钟后投递到 inventory_deduct_expire）
func (p *SaramaProducer) ProduceDelay(ctx context.Context, orderId int64) error {
	msg := DelayMessage{
		Biz:       "inventory",
		Key:       fmt.Sprintf("%d", orderId),
		BizTopic:  TopicDeductExpire,
		ExecuteAt: time.Now().Add(30 * time.Minute).Unix(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicDelayMessage,
		Key:   sarama.StringEncoder(fmt.Sprintf("inventory:%d", orderId)),
		Value: sarama.ByteEncoder(data),
	})
	return err
}

func (p *SaramaProducer) ProduceAlert(ctx context.Context, evt InventoryAlertEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicInventoryAlert,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
```

### 5.3 inventory/events/consumer.go

消费 `inventory_deduct_expire` topic，检查 Redis Hash 是否存在，存在则触发 Rollback。

注意：consumer 需要调用 service 的 Rollback，但为避免循环依赖，consumer 依赖一个 `RollbackFunc` 函数类型，由 IoC 层注入。

```go
package events

import (
	"context"
	"strconv"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/saramax"
)

type DeductExpireConsumer struct {
	client     sarama.ConsumerGroup
	l          logger.Logger
	rollbackFn func(ctx context.Context, orderId int64) error
}

func NewDeductExpireConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	rollbackFn func(ctx context.Context, orderId int64) error,
) *DeductExpireConsumer {
	return &DeductExpireConsumer{
		client:     client,
		l:          l,
		rollbackFn: rollbackFn,
	}
}

func (c *DeductExpireConsumer) Start() error {
	cg := c.client
	handler := saramax.NewHandler[DeductExpireEvent](c.l, c.Consume)
	go func() {
		for {
			err := cg.Consume(context.Background(), []string{TopicDeductExpire}, handler)
			if err != nil {
				c.l.Error("消费 inventory_deduct_expire 事件出错",
					logger.Error(err),
				)
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *DeductExpireConsumer) Consume(msg *sarama.ConsumerMessage, evt DeductExpireEvent) error {
	orderId, err := strconv.ParseInt(evt.Key, 10, 64)
	if err != nil {
		c.l.Error("解析 orderId 失败",
			logger.Error(err),
			logger.String("key", evt.Key),
		)
		return err
	}
	c.l.Info("收到库存预扣超时事件",
		logger.Int64("orderId", orderId),
	)
	return c.rollbackFn(context.Background(), orderId)
}
```

---

## Task 6: Service 层

**Files:**
- Create: `inventory/service/inventory.go`

**说明：** 三阶段核心业务逻辑。

```go
package service

import (
	"context"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/inventory/domain"
	"github.com/rermrf/mall/inventory/events"
	"github.com/rermrf/mall/inventory/repository"
)

type InventoryService interface {
	SetStock(ctx context.Context, tenantId, skuId int64, total, alertThreshold int32) error
	GetStock(ctx context.Context, skuId int64) (domain.Inventory, error)
	BatchGetStock(ctx context.Context, skuIds []int64) ([]domain.Inventory, error)
	Deduct(ctx context.Context, orderId, tenantId int64, items []domain.DeductItem) (bool, string, error)
	Confirm(ctx context.Context, orderId int64) error
	Rollback(ctx context.Context, orderId int64) error
	ListLogs(ctx context.Context, tenantId, skuId int64, page, pageSize int32) ([]domain.InventoryLog, int64, error)
}

type inventoryService struct {
	repo     repository.InventoryRepository
	producer events.Producer
	l        logger.Logger
}

func NewInventoryService(
	repo repository.InventoryRepository,
	producer events.Producer,
	l logger.Logger,
) InventoryService {
	return &inventoryService{repo: repo, producer: producer, l: l}
}

func (s *inventoryService) SetStock(ctx context.Context, tenantId, skuId int64, total, alertThreshold int32) error {
	inv := domain.Inventory{
		TenantID:       tenantId,
		SKUID:          skuId,
		Total:          total,
		Available:      total, // 初始 available = total
		AlertThreshold: alertThreshold,
	}
	return s.repo.SetStock(ctx, inv)
}

func (s *inventoryService) GetStock(ctx context.Context, skuId int64) (domain.Inventory, error) {
	return s.repo.GetStock(ctx, skuId)
}

func (s *inventoryService) BatchGetStock(ctx context.Context, skuIds []int64) ([]domain.Inventory, error) {
	return s.repo.BatchGetStock(ctx, skuIds)
}

// Deduct 预扣库存：Redis Lua 原子预扣 → 存 Redis Hash → 发 go-delay 延迟消息
func (s *inventoryService) Deduct(ctx context.Context, orderId, tenantId int64, items []domain.DeductItem) (bool, string, error) {
	// 构建 items map
	itemMap := make(map[int64]int32, len(items))
	for _, item := range items {
		itemMap[item.SKUID] = item.Quantity
	}

	// 1. Redis Lua 原子预扣
	success, msg, err := s.repo.Deduct(ctx, itemMap)
	if err != nil {
		return false, "", err
	}
	if !success {
		return false, msg, nil
	}

	// 2. 存 Redis Hash 预扣记录
	err = s.repo.SetDeductRecord(ctx, orderId, itemMap)
	if err != nil {
		// 预扣成功但记录失败，回滚
		_ = s.repo.RollbackStock(ctx, itemMap)
		return false, "系统错误", err
	}

	// 3. 发 go-delay 延迟消息（30min 后触发回滚检查）
	err = s.producer.ProduceDelay(ctx, orderId)
	if err != nil {
		s.l.Error("发送延迟消息失败",
			logger.Int64("orderId", orderId),
			logger.Error(err),
		)
		// 延迟消息发送失败不影响预扣结果，只记录日志
		// 最坏情况：需要手动回滚
	}

	return true, "", nil
}

// Confirm 确认扣减（幂等）：读 Redis Hash → MySQL 事务更新 → 删 Redis Hash
func (s *inventoryService) Confirm(ctx context.Context, orderId int64) error {
	// 读 Redis Hash 预扣记录
	items, err := s.repo.GetDeductRecord(ctx, orderId)
	if err != nil {
		return err
	}
	// 幂等：不存在则已处理
	if items == nil {
		return nil
	}

	// 获取 tenantId（从任一 SKU 库存记录获取）
	var tenantId int64
	for skuId := range items {
		inv, e := s.repo.GetStock(ctx, skuId)
		if e == nil {
			tenantId = inv.TenantID
			break
		}
	}

	// MySQL 事务：locked -= qty, sold += qty + 写日志
	err = s.repo.ConfirmStock(ctx, items, orderId, tenantId)
	if err != nil {
		return err
	}

	// 删除 Redis Hash 预扣记录
	_ = s.repo.DeleteDeductRecord(ctx, orderId)

	// 同步 Redis 中的库存（Confirm 后 locked 变化了）
	for skuId := range items {
		inv, e := s.repo.GetStock(ctx, skuId)
		if e == nil && inv.Available < inv.AlertThreshold {
			_ = s.producer.ProduceAlert(ctx, events.InventoryAlertEvent{
				TenantID:  inv.TenantID,
				SKUID:     skuId,
				Available: inv.Available,
				Threshold: inv.AlertThreshold,
			})
		}
	}

	return nil
}

// Rollback 回滚库存（幂等）：读 Redis Hash → Redis Lua 回滚 → MySQL 写日志 → 删 Redis Hash
func (s *inventoryService) Rollback(ctx context.Context, orderId int64) error {
	// 读 Redis Hash 预扣记录
	items, err := s.repo.GetDeductRecord(ctx, orderId)
	if err != nil {
		return err
	}
	// 幂等：不存在则已处理
	if items == nil {
		return nil
	}

	// Redis Lua 回滚
	err = s.repo.RollbackStock(ctx, items)
	if err != nil {
		return err
	}

	// 获取 tenantId
	var tenantId int64
	for skuId := range items {
		inv, e := s.repo.GetStock(ctx, skuId)
		if e == nil {
			tenantId = inv.TenantID
			break
		}
	}

	// MySQL 写回滚日志
	for skuId, qty := range items {
		_ = s.repo.InsertLog(ctx, domain.InventoryLog{
			SKUID:    skuId,
			OrderID:  orderId,
			Type:     domain.LogTypeRollback,
			Quantity: qty,
			TenantID: tenantId,
		})
	}

	// 删除 Redis Hash 预扣记录
	_ = s.repo.DeleteDeductRecord(ctx, orderId)
	return nil
}

func (s *inventoryService) ListLogs(ctx context.Context, tenantId, skuId int64, page, pageSize int32) ([]domain.InventoryLog, int64, error) {
	return s.repo.ListLogs(ctx, tenantId, skuId, page, pageSize)
}
```

---

## Task 7: gRPC Handler

**Files:**
- Create: `inventory/grpc/inventory.go`

**说明：** 实现 proto 中定义的 7 个 RPC。

```go
package grpc

import (
	"context"

	"google.golang.org/grpc"

	inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"
	"github.com/rermrf/mall/inventory/domain"
	"github.com/rermrf/mall/inventory/service"
)

type InventoryGRPCServer struct {
	inventoryv1.UnimplementedInventoryServiceServer
	svc service.InventoryService
}

func NewInventoryGRPCServer(svc service.InventoryService) *InventoryGRPCServer {
	return &InventoryGRPCServer{svc: svc}
}

func (s *InventoryGRPCServer) Register(server *grpc.Server) {
	inventoryv1.RegisterInventoryServiceServer(server, s)
}

func (s *InventoryGRPCServer) SetStock(ctx context.Context, req *inventoryv1.SetStockRequest) (*inventoryv1.SetStockResponse, error) {
	err := s.svc.SetStock(ctx, req.GetTenantId(), req.GetSkuId(), req.GetTotal(), req.GetAlertThreshold())
	if err != nil {
		return nil, err
	}
	return &inventoryv1.SetStockResponse{}, nil
}

func (s *InventoryGRPCServer) GetStock(ctx context.Context, req *inventoryv1.GetStockRequest) (*inventoryv1.GetStockResponse, error) {
	inv, err := s.svc.GetStock(ctx, req.GetSkuId())
	if err != nil {
		return nil, err
	}
	return &inventoryv1.GetStockResponse{
		Inventory: s.toDTO(inv),
	}, nil
}

func (s *InventoryGRPCServer) BatchGetStock(ctx context.Context, req *inventoryv1.BatchGetStockRequest) (*inventoryv1.BatchGetStockResponse, error) {
	invs, err := s.svc.BatchGetStock(ctx, req.GetSkuIds())
	if err != nil {
		return nil, err
	}
	dtos := make([]*inventoryv1.Inventory, 0, len(invs))
	for _, inv := range invs {
		dtos = append(dtos, s.toDTO(inv))
	}
	return &inventoryv1.BatchGetStockResponse{
		Inventories: dtos,
	}, nil
}

func (s *InventoryGRPCServer) Deduct(ctx context.Context, req *inventoryv1.DeductRequest) (*inventoryv1.DeductResponse, error) {
	items := make([]domain.DeductItem, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		items = append(items, domain.DeductItem{
			SKUID:    item.GetSkuId(),
			Quantity: item.GetQuantity(),
		})
	}
	success, msg, err := s.svc.Deduct(ctx, req.GetOrderId(), req.GetTenantId(), items)
	if err != nil {
		return nil, err
	}
	return &inventoryv1.DeductResponse{
		Success: success,
		Message: msg,
	}, nil
}

func (s *InventoryGRPCServer) Confirm(ctx context.Context, req *inventoryv1.ConfirmRequest) (*inventoryv1.ConfirmResponse, error) {
	err := s.svc.Confirm(ctx, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &inventoryv1.ConfirmResponse{}, nil
}

func (s *InventoryGRPCServer) Rollback(ctx context.Context, req *inventoryv1.RollbackRequest) (*inventoryv1.RollbackResponse, error) {
	err := s.svc.Rollback(ctx, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &inventoryv1.RollbackResponse{}, nil
}

func (s *InventoryGRPCServer) ListLogs(ctx context.Context, req *inventoryv1.ListLogsRequest) (*inventoryv1.ListLogsResponse, error) {
	logs, total, err := s.svc.ListLogs(ctx, req.GetTenantId(), req.GetSkuId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*inventoryv1.InventoryLog, 0, len(logs))
	for _, log := range logs {
		dtos = append(dtos, s.toLogDTO(log))
	}
	return &inventoryv1.ListLogsResponse{
		Logs:  dtos,
		Total: total,
	}, nil
}

func (s *InventoryGRPCServer) toDTO(inv domain.Inventory) *inventoryv1.Inventory {
	return &inventoryv1.Inventory{
		Id:             inv.ID,
		TenantId:       inv.TenantID,
		SkuId:          inv.SKUID,
		Total:          inv.Total,
		Available:      inv.Available,
		Locked:         inv.Locked,
		Sold:           inv.Sold,
		AlertThreshold: inv.AlertThreshold,
		Ctime:          inv.Ctime.UnixMilli(),
		Utime:          inv.Utime.UnixMilli(),
	}
}

func (s *InventoryGRPCServer) toLogDTO(log domain.InventoryLog) *inventoryv1.InventoryLog {
	return &inventoryv1.InventoryLog{
		Id:              log.ID,
		SkuId:           log.SKUID,
		OrderId:         log.OrderID,
		Type:            int32(log.Type),
		Quantity:        log.Quantity,
		BeforeAvailable: log.BeforeAvailable,
		AfterAvailable:  log.AfterAvailable,
		TenantId:        log.TenantID,
		Ctime:           log.Ctime.UnixMilli(),
	}
}
```

---

## Task 8: IoC + Wire + Config + Main

**Files:**
- Create: `inventory/ioc/db.go`
- Create: `inventory/ioc/redis.go`
- Create: `inventory/ioc/kafka.go`
- Create: `inventory/ioc/logger.go`
- Create: `inventory/ioc/grpc.go`
- Create: `inventory/config/dev.yaml`
- Create: `inventory/app.go`
- Create: `inventory/wire.go`
- Create: `inventory/main.go`

### 8.1 inventory/ioc/db.go

```go
package ioc

import (
	"fmt"

	"github.com/rermrf/mall/inventory/repository/dao"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	type Config struct {
		DSN string `yaml:"dsn"`
	}
	var cfg Config
	err := viper.UnmarshalKey("db", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取数据库配置失败: %w", err))
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("连接数据库失败: %w", err))
	}
	err = dao.InitTables(db)
	if err != nil {
		panic(fmt.Errorf("数据库表初始化失败: %w", err))
	}
	return db
}
```

### 8.2 inventory/ioc/redis.go

```go
package ioc

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
	type Config struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	}
	var cfg Config
	err := viper.UnmarshalKey("redis", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Redis 配置失败: %w", err))
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return client
}
```

### 8.3 inventory/ioc/kafka.go

```go
package ioc

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/inventory/events"
	"github.com/rermrf/mall/inventory/service"
	"github.com/rermrf/mall/pkg/saramax"
	"github.com/spf13/viper"
)

func InitKafka() sarama.Client {
	type Config struct {
		Addrs []string `yaml:"addrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("kafka", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Kafka 配置失败: %w", err))
	}
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	client, err := sarama.NewClient(cfg.Addrs, saramaCfg)
	if err != nil {
		panic(fmt.Errorf("连接 Kafka 失败: %w", err))
	}
	return client
}

func InitSyncProducer(client sarama.Client) sarama.SyncProducer {
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka SyncProducer 失败: %w", err))
	}
	return producer
}

func InitProducer(p sarama.SyncProducer) events.Producer {
	return events.NewSaramaProducer(p)
}

func InitConsumerGroup(client sarama.Client) sarama.ConsumerGroup {
	cg, err := sarama.NewConsumerGroupFromClient("inventory-svc", client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka ConsumerGroup 失败: %w", err))
	}
	return cg
}

func NewDeductExpireConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.InventoryService,
) *events.DeductExpireConsumer {
	return events.NewDeductExpireConsumer(cg, l, func(ctx context.Context, orderId int64) error {
		return svc.Rollback(ctx, orderId)
	})
}

func InitConsumers(c *events.DeductExpireConsumer) []saramax.Consumer {
	return []saramax.Consumer{c}
}
```

### 8.4 inventory/ioc/logger.go

```go
package ioc

import (
	"github.com/rermrf/emo/logger"
	"go.uber.org/zap"
)

func InitLogger() logger.Logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger.NewZapLogger(l)
}
```

### 8.5 inventory/ioc/grpc.go

```go
package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
	igrpc "github.com/rermrf/mall/inventory/grpc"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func InitGRPCServer(inventoryServer *igrpc.InventoryGRPCServer, l logger.Logger) *grpcx.Server {
	type Config struct {
		Port      int      `yaml:"port"`
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 gRPC 配置失败: %w", err))
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(tenantx.GRPCUnaryServerInterceptor()))
	inventoryServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "inventory",
		L:         l,
	}
}
```

### 8.6 inventory/config/dev.yaml

```yaml
db:
  dsn: "root:wen123...@tcp(rermrf.icu:3306)/mall_inventory?charset=utf8mb4&parseTime=true&loc=Local"

redis:
  addr: "rermrf.icu:6379"
  password: ""
  db: 3

kafka:
  addrs:
    - "rermrf.icu:9094"

grpc:
  port: 8084
  etcdAddrs:
    - "rermrf.icu:2379"
```

### 8.7 inventory/app.go

```go
package main

import (
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/saramax"
)

type App struct {
	Server    *grpcx.Server
	Consumers []saramax.Consumer
}
```

### 8.8 inventory/wire.go

```go
//go:build wireinject

package main

import (
	"github.com/google/wire"
	igrpc "github.com/rermrf/mall/inventory/grpc"
	"github.com/rermrf/mall/inventory/ioc"
	"github.com/rermrf/mall/inventory/repository"
	"github.com/rermrf/mall/inventory/repository/cache"
	"github.com/rermrf/mall/inventory/repository/dao"
	"github.com/rermrf/mall/inventory/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
)

var inventorySet = wire.NewSet(
	dao.NewInventoryDAO,
	cache.NewInventoryCache,
	repository.NewInventoryRepository,
	service.NewInventoryService,
	igrpc.NewInventoryGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitConsumerGroup,
	ioc.NewDeductExpireConsumer,
	ioc.InitConsumers,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, inventorySet, wire.Struct(new(App), "*"))
	return new(App)
}
```

### 8.9 inventory/main.go

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()
	app := InitApp()

	// 启动所有消费者
	for _, c := range app.Consumers {
		if err := c.Start(); err != nil {
			panic(fmt.Errorf("启动消费者失败: %w", err))
		}
	}

	// 启动 gRPC 服务
	go func() {
		if err := app.Server.Serve(); err != nil {
			fmt.Println("gRPC 服务启动失败:", err)
			os.Exit(1)
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("正在关闭服务...")
	app.Server.Close()
}

func initViper() {
	configFile := pflag.String("config", "config/dev.yaml", "配置文件路径")
	pflag.Parse()
	viper.SetConfigFile(*configFile)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}
}
```

---

## 验证步骤

1. `go build ./inventory/...` — 编译通过
2. `go vet ./inventory/...` — 无警告
3. `cd inventory && wire` — Wire DI 生成成功
4. 再次 `go build ./inventory/...` — 含 wire_gen.go 编译通过

---

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `inventory/domain/inventory.go` | 新建 | Inventory + DeductRecord + DeductItem + InventoryLog + 枚举 |
| 2 | `inventory/repository/dao/inventory.go` | 新建 | 2 GORM 模型 + InventoryDAO |
| 3 | `inventory/repository/dao/init.go` | 新建 | AutoMigrate 2 张表 |
| 4 | `inventory/repository/cache/inventory.go` | 新建 | Redis Hash + Lua 脚本调用 |
| 5 | `inventory/repository/cache/lua/deduct.lua` | 新建 | 多 SKU 原子预扣 Lua |
| 6 | `inventory/repository/cache/lua/rollback.lua` | 新建 | 多 SKU 原子回滚 Lua |
| 7 | `inventory/repository/inventory.go` | 新建 | MySQL + Redis 协调 |
| 8 | `inventory/events/types.go` | 新建 | 延迟消息 + 预警事件 DTO |
| 9 | `inventory/events/producer.go` | 新建 | Kafka Producer（delay_topic + inventory_alert） |
| 10 | `inventory/events/consumer.go` | 新建 | 消费 inventory_deduct_expire → Rollback |
| 11 | `inventory/service/inventory.go` | 新建 | 三阶段核心业务逻辑 |
| 12 | `inventory/grpc/inventory.go` | 新建 | 7 RPC handler |
| 13 | `inventory/ioc/db.go` | 新建 | MySQL 初始化 |
| 14 | `inventory/ioc/redis.go` | 新建 | Redis 初始化 |
| 15 | `inventory/ioc/kafka.go` | 新建 | Kafka + Producer + ConsumerGroup + Consumer |
| 16 | `inventory/ioc/logger.go` | 新建 | Logger 初始化 |
| 17 | `inventory/ioc/grpc.go` | 新建 | gRPC server 初始化 |
| 18 | `inventory/config/dev.yaml` | 新建 | 配置（port 8084, db mall_inventory, redis db 3） |
| 19 | `inventory/app.go` | 新建 | App 聚合（Server + Consumers） |
| 20 | `inventory/wire.go` | 新建 | Wire DI |
| 21 | `inventory/main.go` | 新建 | 服务入口 |

共 21 个文件（设计文档说 19 个，实际增加了 2 个 Lua 脚本文件）。
