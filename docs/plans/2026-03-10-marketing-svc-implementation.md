# Marketing Service + BFF 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 marketing-svc 营销微服务（16 个 gRPC RPC + Kafka Producer/Consumer）+ merchant-bff 营销管理接口（10 个端点）+ consumer-bff 营销消费者接口（5 个端点）。涵盖优惠券、秒杀（Redis+Lua→Kafka 削峰）、满减规则三大功能域。

**Architecture:** DDD 分层（domain → dao(MySQL) → cache(Redis+Lua) → repository → service → grpc → events → ioc → wire）。MySQL 持久化优惠券/秒杀/满减数据，Redis 做优惠券库存和秒杀库存的原子扣减（Lua 脚本），Kafka 生产 `seckill_success` 事件由 order-svc 异步创建订单，消费 `order_cancelled` 事件释放优惠券。merchant-bff 提供管理接口，consumer-bff 提供 C 端领券/秒杀接口。

**Tech Stack:** Go, gRPC, MySQL (GORM), Redis (go-redis/v9 + Lua), Kafka (Sarama), Wire DI, etcd, Gin (BFF)

---

## Task 1: Domain + DAO + Init（marketing-svc 数据层）

**Files:**
- Create: `marketing/domain/marketing.go`
- Create: `marketing/repository/dao/marketing.go`
- Create: `marketing/repository/dao/init.go`

**Step 1: 创建 domain 模型**

Create `marketing/domain/marketing.go`:

```go
package domain

import "time"

// ==================== 优惠券 ====================

type Coupon struct {
	ID            int64
	TenantID      int64
	Name          string
	Type          int32 // 1-满减 2-折扣 3-无门槛
	Threshold     int64 // 使用门槛（分），0=无门槛
	DiscountValue int64 // 满减=金额分，折扣=折扣比*100
	TotalCount    int32
	ReceivedCount int32
	UsedCount     int32
	PerLimit      int32 // 每人限领
	StartTime     time.Time
	EndTime       time.Time
	ScopeType     int32  // 1-全店 2-指定分类 3-指定商品
	ScopeIDs      string // JSON
	Status        int32  // 1-未开始 2-进行中 3-已结束 4-已停用
	Ctime         time.Time
}

type UserCoupon struct {
	ID          int64
	UserID      int64
	CouponID    int64
	TenantID    int64
	Status      int32 // 1-未使用 2-已使用 3-已过期
	OrderID     int64
	ReceiveTime time.Time
	UseTime     time.Time
	Coupon      Coupon // 嵌套
}

// ==================== 秒杀 ====================

type SeckillActivity struct {
	ID        int64
	TenantID  int64
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Status    int32 // 1-未开始 2-进行中 3-已结束
	Items     []SeckillItem
}

type SeckillItem struct {
	ID           int64
	ActivityID   int64
	TenantID     int64
	SkuID        int64
	SeckillPrice int64 // 分
	SeckillStock int32
	PerLimit     int32
}

type SeckillOrder struct {
	ID       int64
	UserID   int64
	ItemID   int64
	TenantID int64
	OrderNo  string
	Status   int32 // 1-排队中 2-已创建订单 3-失败
}

// ==================== 满减 ====================

type PromotionRule struct {
	ID            int64
	TenantID      int64
	Name          string
	Type          int32 // 1-满减 2-满折
	Threshold     int64
	DiscountValue int64
	StartTime     time.Time
	EndTime       time.Time
	Status        int32
}

// ==================== 优惠计算结果 ====================

type DiscountResult struct {
	CouponDiscount    int64 // 优惠券优惠金额（分）
	PromotionDiscount int64 // 满减优惠金额（分）
	TotalDiscount     int64 // 总优惠（分）
	PayAmount         int64 // 应付（分）
}
```

**Step 2: 创建 DAO**

Create `marketing/repository/dao/marketing.go`:

```go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ==================== Models ====================

type Coupon struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	TenantID      int64  `gorm:"not null;index:idx_tenant_status"`
	Name          string `gorm:"type:varchar(200);not null"`
	Type          int32  `gorm:"not null"` // 1-满减 2-折扣 3-无门槛
	Threshold     int64  `gorm:"not null;default:0"`
	DiscountValue int64  `gorm:"not null"`
	TotalCount    int32  `gorm:"not null"`
	ReceivedCount int32  `gorm:"not null;default:0"`
	UsedCount     int32  `gorm:"not null;default:0"`
	PerLimit      int32  `gorm:"not null;default:1"`
	StartTime     int64  `gorm:"not null"`
	EndTime       int64  `gorm:"not null"`
	ScopeType     int32  `gorm:"not null;default:1"`
	ScopeIDs      string `gorm:"type:varchar(1000)"`
	Status        int32  `gorm:"not null;default:1;index:idx_tenant_status,priority:2"`
	Ctime         int64  `gorm:"not null"`
	Utime         int64  `gorm:"not null"`
}

type UserCoupon struct {
	ID          int64 `gorm:"primaryKey;autoIncrement"`
	UserID      int64 `gorm:"not null;uniqueIndex:uk_user_coupon;index:idx_user_tenant_status"`
	CouponID    int64 `gorm:"not null;uniqueIndex:uk_user_coupon,priority:2"`
	TenantID    int64 `gorm:"not null;index:idx_user_tenant_status,priority:2"`
	Status      int32 `gorm:"not null;default:1;index:idx_user_tenant_status,priority:3"` // 1-未使用 2-已使用 3-已过期
	OrderID     int64 `gorm:"default:0"`
	ReceiveTime int64 `gorm:"not null"`
	UseTime     int64 `gorm:"default:0"`
	Ctime       int64 `gorm:"not null"`
	Utime       int64 `gorm:"not null"`
}

type SeckillActivity struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	TenantID  int64  `gorm:"not null;index:idx_tenant_status"`
	Name      string `gorm:"type:varchar(200);not null"`
	StartTime int64  `gorm:"not null"`
	EndTime   int64  `gorm:"not null"`
	Status    int32  `gorm:"not null;default:1;index:idx_tenant_status,priority:2"`
	Ctime     int64  `gorm:"not null"`
	Utime     int64  `gorm:"not null"`
}

type SeckillItem struct {
	ID           int64 `gorm:"primaryKey;autoIncrement"`
	ActivityID   int64 `gorm:"not null;index:idx_activity"`
	TenantID     int64 `gorm:"not null"`
	SkuID        int64 `gorm:"not null;uniqueIndex:uk_activity_sku"`
	SeckillPrice int64 `gorm:"not null"`
	SeckillStock int32 `gorm:"not null"`
	PerLimit     int32 `gorm:"not null;default:1"`
}

type SeckillOrder struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	UserID   int64  `gorm:"not null;uniqueIndex:uk_user_item"`
	ItemID   int64  `gorm:"not null;uniqueIndex:uk_user_item,priority:2"`
	TenantID int64  `gorm:"not null"`
	OrderNo  string `gorm:"type:varchar(64)"`
	Status   int32  `gorm:"not null;default:1"` // 1-排队中 2-已创建订单 3-失败
	Ctime    int64  `gorm:"not null"`
	Utime    int64  `gorm:"not null"`
}

type PromotionRule struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	TenantID      int64  `gorm:"not null;index:idx_tenant_status"`
	Name          string `gorm:"type:varchar(200);not null"`
	Type          int32  `gorm:"not null"` // 1-满减 2-满折
	Threshold     int64  `gorm:"not null"`
	DiscountValue int64  `gorm:"not null"`
	StartTime     int64  `gorm:"not null"`
	EndTime       int64  `gorm:"not null"`
	Status        int32  `gorm:"not null;default:1;index:idx_tenant_status,priority:2"`
	Ctime         int64  `gorm:"not null"`
	Utime         int64  `gorm:"not null"`
}

// ==================== CouponDAO ====================

type CouponDAO interface {
	Insert(ctx context.Context, c Coupon) (Coupon, error)
	Update(ctx context.Context, c Coupon) error
	List(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]Coupon, int64, error)
	FindById(ctx context.Context, id int64) (Coupon, error)
	IncrReceivedCount(ctx context.Context, id int64) error
	IncrUsedCount(ctx context.Context, id int64) error
	DecrUsedCount(ctx context.Context, id int64) error
}

type GORMCouponDAO struct {
	db *gorm.DB
}

func NewCouponDAO(db *gorm.DB) CouponDAO {
	return &GORMCouponDAO{db: db}
}

func (d *GORMCouponDAO) Insert(ctx context.Context, c Coupon) (Coupon, error) {
	now := time.Now().UnixMilli()
	c.Ctime = now
	c.Utime = now
	err := d.db.WithContext(ctx).Create(&c).Error
	return c, err
}

func (d *GORMCouponDAO) Update(ctx context.Context, c Coupon) error {
	c.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", c.ID, c.TenantID).Updates(&c).Error
}

func (d *GORMCouponDAO) List(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]Coupon, int64, error) {
	db := d.db.WithContext(ctx).Model(&Coupon{}).Where("tenant_id = ?", tenantId)
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	var coupons []Coupon
	err = db.Order("id DESC").Offset(offset).Limit(limit).Find(&coupons).Error
	return coupons, total, err
}

func (d *GORMCouponDAO) FindById(ctx context.Context, id int64) (Coupon, error) {
	var c Coupon
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	return c, err
}

func (d *GORMCouponDAO) IncrReceivedCount(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Model(&Coupon{}).Where("id = ?", id).
		UpdateColumn("received_count", gorm.Expr("received_count + 1")).Error
}

func (d *GORMCouponDAO) IncrUsedCount(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Model(&Coupon{}).Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

func (d *GORMCouponDAO) DecrUsedCount(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Model(&Coupon{}).Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count - 1")).Error
}

// ==================== UserCouponDAO ====================

type UserCouponDAO interface {
	Insert(ctx context.Context, uc UserCoupon) (UserCoupon, error)
	FindById(ctx context.Context, id int64) (UserCoupon, error)
	ListByUser(ctx context.Context, userId, tenantId int64, status int32) ([]UserCoupon, error)
	UpdateStatus(ctx context.Context, id int64, status int32, orderId int64) error
	CountByUserAndCoupon(ctx context.Context, userId, couponId int64) (int64, error)
}

type GORMUserCouponDAO struct {
	db *gorm.DB
}

func NewUserCouponDAO(db *gorm.DB) UserCouponDAO {
	return &GORMUserCouponDAO{db: db}
}

func (d *GORMUserCouponDAO) Insert(ctx context.Context, uc UserCoupon) (UserCoupon, error) {
	now := time.Now().UnixMilli()
	uc.Ctime = now
	uc.Utime = now
	uc.ReceiveTime = now
	err := d.db.WithContext(ctx).Create(&uc).Error
	return uc, err
}

func (d *GORMUserCouponDAO) FindById(ctx context.Context, id int64) (UserCoupon, error) {
	var uc UserCoupon
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&uc).Error
	return uc, err
}

func (d *GORMUserCouponDAO) ListByUser(ctx context.Context, userId, tenantId int64, status int32) ([]UserCoupon, error) {
	db := d.db.WithContext(ctx).Where("user_id = ?", userId)
	if tenantId > 0 {
		db = db.Where("tenant_id = ?", tenantId)
	}
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var ucs []UserCoupon
	err := db.Order("id DESC").Find(&ucs).Error
	return ucs, err
}

func (d *GORMUserCouponDAO) UpdateStatus(ctx context.Context, id int64, status int32, orderId int64) error {
	updates := map[string]any{
		"status": status,
		"utime":  time.Now().UnixMilli(),
	}
	if orderId > 0 {
		updates["order_id"] = orderId
		updates["use_time"] = time.Now().UnixMilli()
	}
	if status == 1 {
		// 释放时清空 order_id 和 use_time
		updates["order_id"] = 0
		updates["use_time"] = 0
	}
	return d.db.WithContext(ctx).Model(&UserCoupon{}).Where("id = ?", id).Updates(updates).Error
}

func (d *GORMUserCouponDAO) CountByUserAndCoupon(ctx context.Context, userId, couponId int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&UserCoupon{}).
		Where("user_id = ? AND coupon_id = ?", userId, couponId).Count(&count).Error
	return count, err
}

// ==================== SeckillDAO ====================

type SeckillDAO interface {
	InsertActivity(ctx context.Context, a SeckillActivity, items []SeckillItem) (SeckillActivity, error)
	UpdateActivity(ctx context.Context, a SeckillActivity, items []SeckillItem) error
	ListActivities(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]SeckillActivity, int64, error)
	FindActivityById(ctx context.Context, id int64) (SeckillActivity, []SeckillItem, error)
	FindItemById(ctx context.Context, id int64) (SeckillItem, error)
	InsertSeckillOrder(ctx context.Context, o SeckillOrder) (SeckillOrder, error)
	FindSeckillOrderByUserAndItem(ctx context.Context, userId, itemId int64) (SeckillOrder, error)
}

type GORMSeckillDAO struct {
	db *gorm.DB
}

func NewSeckillDAO(db *gorm.DB) SeckillDAO {
	return &GORMSeckillDAO{db: db}
}

func (d *GORMSeckillDAO) InsertActivity(ctx context.Context, a SeckillActivity, items []SeckillItem) (SeckillActivity, error) {
	now := time.Now().UnixMilli()
	a.Ctime = now
	a.Utime = now
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&a).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].ActivityID = a.ID
			items[i].TenantID = a.TenantID
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
	return a, err
}

func (d *GORMSeckillDAO) UpdateActivity(ctx context.Context, a SeckillActivity, items []SeckillItem) error {
	a.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND tenant_id = ?", a.ID, a.TenantID).Updates(&a).Error; err != nil {
			return err
		}
		// 删除旧 items 并重建
		if err := tx.Where("activity_id = ?", a.ID).Delete(&SeckillItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].ActivityID = a.ID
			items[i].TenantID = a.TenantID
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
}

func (d *GORMSeckillDAO) ListActivities(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]SeckillActivity, int64, error) {
	db := d.db.WithContext(ctx).Model(&SeckillActivity{}).Where("tenant_id = ?", tenantId)
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	var activities []SeckillActivity
	err = db.Order("id DESC").Offset(offset).Limit(limit).Find(&activities).Error
	return activities, total, err
}

func (d *GORMSeckillDAO) FindActivityById(ctx context.Context, id int64) (SeckillActivity, []SeckillItem, error) {
	var a SeckillActivity
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&a).Error
	if err != nil {
		return a, nil, err
	}
	var items []SeckillItem
	err = d.db.WithContext(ctx).Where("activity_id = ?", id).Find(&items).Error
	return a, items, err
}

func (d *GORMSeckillDAO) FindItemById(ctx context.Context, id int64) (SeckillItem, error) {
	var item SeckillItem
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return item, err
}

func (d *GORMSeckillDAO) InsertSeckillOrder(ctx context.Context, o SeckillOrder) (SeckillOrder, error) {
	now := time.Now().UnixMilli()
	o.Ctime = now
	o.Utime = now
	err := d.db.WithContext(ctx).Create(&o).Error
	return o, err
}

func (d *GORMSeckillDAO) FindSeckillOrderByUserAndItem(ctx context.Context, userId, itemId int64) (SeckillOrder, error) {
	var o SeckillOrder
	err := d.db.WithContext(ctx).Where("user_id = ? AND item_id = ?", userId, itemId).First(&o).Error
	return o, err
}

// ==================== PromotionDAO ====================

type PromotionDAO interface {
	Insert(ctx context.Context, r PromotionRule) (PromotionRule, error)
	Update(ctx context.Context, r PromotionRule) error
	List(ctx context.Context, tenantId int64, status int32) ([]PromotionRule, error)
	ListActive(ctx context.Context, tenantId int64, now int64) ([]PromotionRule, error)
}

type GORMPromotionDAO struct {
	db *gorm.DB
}

func NewPromotionDAO(db *gorm.DB) PromotionDAO {
	return &GORMPromotionDAO{db: db}
}

func (d *GORMPromotionDAO) Insert(ctx context.Context, r PromotionRule) (PromotionRule, error) {
	now := time.Now().UnixMilli()
	r.Ctime = now
	r.Utime = now
	err := d.db.WithContext(ctx).Create(&r).Error
	return r, err
}

func (d *GORMPromotionDAO) Update(ctx context.Context, r PromotionRule) error {
	r.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", r.ID, r.TenantID).Updates(&r).Error
}

func (d *GORMPromotionDAO) List(ctx context.Context, tenantId int64, status int32) ([]PromotionRule, error) {
	db := d.db.WithContext(ctx).Where("tenant_id = ?", tenantId)
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var rules []PromotionRule
	err := db.Order("id DESC").Find(&rules).Error
	return rules, err
}

func (d *GORMPromotionDAO) ListActive(ctx context.Context, tenantId int64, now int64) ([]PromotionRule, error) {
	var rules []PromotionRule
	err := d.db.WithContext(ctx).
		Where("tenant_id = ? AND status = 2 AND start_time <= ? AND end_time >= ?", tenantId, now, now).
		Find(&rules).Error
	return rules, err
}
```

**Step 3: 创建 AutoMigrate**

Create `marketing/repository/dao/init.go`:

```go
package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&Coupon{},
		&UserCoupon{},
		&SeckillActivity{},
		&SeckillItem{},
		&SeckillOrder{},
		&PromotionRule{},
	)
}
```

**Step 4: 验证编译**

```bash
go build ./marketing/...
```

---

## Task 2: Cache + Lua + Repository（marketing-svc 缓存与仓储层）

**Files:**
- Create: `marketing/repository/cache/lua/seckill.lua`
- Create: `marketing/repository/cache/marketing.go`
- Create: `marketing/repository/marketing.go`

**Step 1: 创建秒杀 Lua 脚本**

Create `marketing/repository/cache/lua/seckill.lua`:

```lua
-- seckill.lua: 秒杀原子扣减 + 防重复
-- KEYS[1]: seckill:stock:{itemId}
-- KEYS[2]: seckill:user:{itemId}
-- ARGV[1]: userId
-- ARGV[2]: perLimit
-- 返回: 0=成功, 1=库存不足, 2=已抢购过, 3=超出限购

-- 检查是否已抢购
local userSet = KEYS[2]
local userId = ARGV[1]
local isMember = redis.call('SISMEMBER', userSet, userId)
if isMember == 1 then
    return 2
end

-- 检查库存
local stockKey = KEYS[1]
local stock = tonumber(redis.call('GET', stockKey) or '0')
if stock <= 0 then
    return 1
end

-- 扣减库存
redis.call('DECR', stockKey)
-- 记录用户
redis.call('SADD', userSet, userId)
return 0
```

**Step 2: 创建 Redis Cache**

Create `marketing/repository/cache/marketing.go`:

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
```

**Step 3: 创建 Repository**

Create `marketing/repository/marketing.go`:

```go
package repository

import (
	"context"
	"time"

	"github.com/rermrf/mall/marketing/domain"
	"github.com/rermrf/mall/marketing/repository/cache"
	"github.com/rermrf/mall/marketing/repository/dao"
)

type MarketingRepository interface {
	// 优惠券
	CreateCoupon(ctx context.Context, c domain.Coupon) (domain.Coupon, error)
	UpdateCoupon(ctx context.Context, c domain.Coupon) error
	ListCoupons(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.Coupon, int64, error)
	FindCouponById(ctx context.Context, id int64) (domain.Coupon, error)
	// 领券
	ReceiveCoupon(ctx context.Context, userId, couponId, tenantId int64) error
	ListUserCoupons(ctx context.Context, userId, tenantId int64, status int32) ([]domain.UserCoupon, error)
	UseCoupon(ctx context.Context, userCouponId, orderId int64) error
	ReleaseCoupon(ctx context.Context, userCouponId int64) error
	FindUserCouponById(ctx context.Context, id int64) (domain.UserCoupon, error)
	CountUserCoupon(ctx context.Context, userId, couponId int64) (int64, error)
	// 秒杀
	CreateSeckillActivity(ctx context.Context, a domain.SeckillActivity) (domain.SeckillActivity, error)
	UpdateSeckillActivity(ctx context.Context, a domain.SeckillActivity) error
	ListSeckillActivities(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SeckillActivity, int64, error)
	FindSeckillActivityById(ctx context.Context, id int64) (domain.SeckillActivity, error)
	FindSeckillItemById(ctx context.Context, id int64) (domain.SeckillItem, error)
	Seckill(ctx context.Context, itemId, userId int64, perLimit int32) (int64, error)
	CreateSeckillOrder(ctx context.Context, o domain.SeckillOrder) (domain.SeckillOrder, error)
	// 满减
	CreatePromotionRule(ctx context.Context, r domain.PromotionRule) (domain.PromotionRule, error)
	UpdatePromotionRule(ctx context.Context, r domain.PromotionRule) error
	ListPromotionRules(ctx context.Context, tenantId int64, status int32) ([]domain.PromotionRule, error)
	ListActivePromotionRules(ctx context.Context, tenantId int64) ([]domain.PromotionRule, error)
}

type marketingRepository struct {
	couponDAO    dao.CouponDAO
	ucDAO        dao.UserCouponDAO
	seckillDAO   dao.SeckillDAO
	promotionDAO dao.PromotionDAO
	cache        cache.MarketingCache
}

func NewMarketingRepository(
	couponDAO dao.CouponDAO,
	ucDAO dao.UserCouponDAO,
	seckillDAO dao.SeckillDAO,
	promotionDAO dao.PromotionDAO,
	c cache.MarketingCache,
) MarketingRepository {
	return &marketingRepository{
		couponDAO:    couponDAO,
		ucDAO:        ucDAO,
		seckillDAO:   seckillDAO,
		promotionDAO: promotionDAO,
		cache:        c,
	}
}

// ==================== 优惠券 ====================

func (r *marketingRepository) CreateCoupon(ctx context.Context, c domain.Coupon) (domain.Coupon, error) {
	dc, err := r.couponDAO.Insert(ctx, r.couponToDAO(c))
	if err != nil {
		return domain.Coupon{}, err
	}
	// 初始化 Redis 库存
	_ = r.cache.SetCouponStock(ctx, dc.ID, dc.TotalCount-dc.ReceivedCount)
	return r.couponToDomain(dc), nil
}

func (r *marketingRepository) UpdateCoupon(ctx context.Context, c domain.Coupon) error {
	return r.couponDAO.Update(ctx, r.couponToDAO(c))
}

func (r *marketingRepository) ListCoupons(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.Coupon, int64, error) {
	offset := int((page - 1) * pageSize)
	coupons, total, err := r.couponDAO.List(ctx, tenantId, status, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	result := make([]domain.Coupon, 0, len(coupons))
	for _, c := range coupons {
		result = append(result, r.couponToDomain(c))
	}
	return result, total, nil
}

func (r *marketingRepository) FindCouponById(ctx context.Context, id int64) (domain.Coupon, error) {
	c, err := r.couponDAO.FindById(ctx, id)
	if err != nil {
		return domain.Coupon{}, err
	}
	return r.couponToDomain(c), nil
}

func (r *marketingRepository) ReceiveCoupon(ctx context.Context, userId, couponId, tenantId int64) error {
	// Redis 原子扣减库存
	remaining, err := r.cache.DecrCouponStock(ctx, couponId)
	if err != nil {
		return err
	}
	if remaining < 0 {
		// 回补库存
		_ = r.cache.IncrCouponStock(ctx, couponId)
		return ErrCouponStockNotEnough
	}
	// MySQL 写入领取记录
	_, err = r.ucDAO.Insert(ctx, dao.UserCoupon{
		UserID:   userId,
		CouponID: couponId,
		TenantID: tenantId,
		Status:   1,
	})
	if err != nil {
		// 回补 Redis 库存
		_ = r.cache.IncrCouponStock(ctx, couponId)
		return err
	}
	// 更新 MySQL 已领取计数
	_ = r.couponDAO.IncrReceivedCount(ctx, couponId)
	return nil
}

func (r *marketingRepository) ListUserCoupons(ctx context.Context, userId, tenantId int64, status int32) ([]domain.UserCoupon, error) {
	ucs, err := r.ucDAO.ListByUser(ctx, userId, tenantId, status)
	if err != nil {
		return nil, err
	}
	result := make([]domain.UserCoupon, 0, len(ucs))
	for _, uc := range ucs {
		duc := r.userCouponToDomain(uc)
		// 查询关联的优惠券信息
		coupon, err := r.couponDAO.FindById(ctx, uc.CouponID)
		if err == nil {
			duc.Coupon = r.couponToDomain(coupon)
		}
		result = append(result, duc)
	}
	return result, nil
}

func (r *marketingRepository) UseCoupon(ctx context.Context, userCouponId, orderId int64) error {
	return r.ucDAO.UpdateStatus(ctx, userCouponId, 2, orderId)
}

func (r *marketingRepository) ReleaseCoupon(ctx context.Context, userCouponId int64) error {
	// 查出 user_coupon 获取 coupon_id
	uc, err := r.ucDAO.FindById(ctx, userCouponId)
	if err != nil {
		return err
	}
	// 更新状态为未使用
	err = r.ucDAO.UpdateStatus(ctx, userCouponId, 1, 0)
	if err != nil {
		return err
	}
	// Redis 回补库存
	_ = r.cache.IncrCouponStock(ctx, uc.CouponID)
	// MySQL 扣减已使用计数
	_ = r.couponDAO.DecrUsedCount(ctx, uc.CouponID)
	return nil
}

func (r *marketingRepository) FindUserCouponById(ctx context.Context, id int64) (domain.UserCoupon, error) {
	uc, err := r.ucDAO.FindById(ctx, id)
	if err != nil {
		return domain.UserCoupon{}, err
	}
	return r.userCouponToDomain(uc), nil
}

func (r *marketingRepository) CountUserCoupon(ctx context.Context, userId, couponId int64) (int64, error) {
	return r.ucDAO.CountByUserAndCoupon(ctx, userId, couponId)
}

// ==================== 秒杀 ====================

func (r *marketingRepository) CreateSeckillActivity(ctx context.Context, a domain.SeckillActivity) (domain.SeckillActivity, error) {
	items := make([]dao.SeckillItem, 0, len(a.Items))
	for _, item := range a.Items {
		items = append(items, r.seckillItemToDAO(item))
	}
	da, err := r.seckillDAO.InsertActivity(ctx, r.seckillActivityToDAO(a), items)
	if err != nil {
		return domain.SeckillActivity{}, err
	}
	// 初始化 Redis 秒杀库存
	for _, item := range items {
		_ = r.cache.SetSeckillStock(ctx, item.ID, item.SeckillStock)
	}
	return r.seckillActivityToDomain(da, items), nil
}

func (r *marketingRepository) UpdateSeckillActivity(ctx context.Context, a domain.SeckillActivity) error {
	items := make([]dao.SeckillItem, 0, len(a.Items))
	for _, item := range a.Items {
		items = append(items, r.seckillItemToDAO(item))
	}
	return r.seckillDAO.UpdateActivity(ctx, r.seckillActivityToDAO(a), items)
}

func (r *marketingRepository) ListSeckillActivities(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SeckillActivity, int64, error) {
	offset := int((page - 1) * pageSize)
	activities, total, err := r.seckillDAO.ListActivities(ctx, tenantId, status, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	result := make([]domain.SeckillActivity, 0, len(activities))
	for _, a := range activities {
		result = append(result, r.seckillActivityToDomain(a, nil))
	}
	return result, total, nil
}

func (r *marketingRepository) FindSeckillActivityById(ctx context.Context, id int64) (domain.SeckillActivity, error) {
	a, items, err := r.seckillDAO.FindActivityById(ctx, id)
	if err != nil {
		return domain.SeckillActivity{}, err
	}
	return r.seckillActivityToDomain(a, items), nil
}

func (r *marketingRepository) FindSeckillItemById(ctx context.Context, id int64) (domain.SeckillItem, error) {
	item, err := r.seckillDAO.FindItemById(ctx, id)
	if err != nil {
		return domain.SeckillItem{}, err
	}
	return r.seckillItemToDomain(item), nil
}

func (r *marketingRepository) Seckill(ctx context.Context, itemId, userId int64, perLimit int32) (int64, error) {
	return r.cache.Seckill(ctx, itemId, userId, perLimit)
}

func (r *marketingRepository) CreateSeckillOrder(ctx context.Context, o domain.SeckillOrder) (domain.SeckillOrder, error) {
	do, err := r.seckillDAO.InsertSeckillOrder(ctx, dao.SeckillOrder{
		UserID:   o.UserID,
		ItemID:   o.ItemID,
		TenantID: o.TenantID,
		OrderNo:  o.OrderNo,
		Status:   o.Status,
	})
	if err != nil {
		return domain.SeckillOrder{}, err
	}
	o.ID = do.ID
	return o, nil
}

// ==================== 满减 ====================

func (r *marketingRepository) CreatePromotionRule(ctx context.Context, rule domain.PromotionRule) (domain.PromotionRule, error) {
	dr, err := r.promotionDAO.Insert(ctx, r.promotionToDAO(rule))
	if err != nil {
		return domain.PromotionRule{}, err
	}
	return r.promotionToDomain(dr), nil
}

func (r *marketingRepository) UpdatePromotionRule(ctx context.Context, rule domain.PromotionRule) error {
	return r.promotionDAO.Update(ctx, r.promotionToDAO(rule))
}

func (r *marketingRepository) ListPromotionRules(ctx context.Context, tenantId int64, status int32) ([]domain.PromotionRule, error) {
	rules, err := r.promotionDAO.List(ctx, tenantId, status)
	if err != nil {
		return nil, err
	}
	result := make([]domain.PromotionRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, r.promotionToDomain(rule))
	}
	return result, nil
}

func (r *marketingRepository) ListActivePromotionRules(ctx context.Context, tenantId int64) ([]domain.PromotionRule, error) {
	rules, err := r.promotionDAO.ListActive(ctx, tenantId, time.Now().UnixMilli())
	if err != nil {
		return nil, err
	}
	result := make([]domain.PromotionRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, r.promotionToDomain(rule))
	}
	return result, nil
}

// ==================== Converters ====================

func (r *marketingRepository) couponToDAO(c domain.Coupon) dao.Coupon {
	return dao.Coupon{
		ID: c.ID, TenantID: c.TenantID, Name: c.Name, Type: c.Type,
		Threshold: c.Threshold, DiscountValue: c.DiscountValue,
		TotalCount: c.TotalCount, ReceivedCount: c.ReceivedCount,
		UsedCount: c.UsedCount, PerLimit: c.PerLimit,
		StartTime: c.StartTime.UnixMilli(), EndTime: c.EndTime.UnixMilli(),
		ScopeType: c.ScopeType, ScopeIDs: c.ScopeIDs, Status: c.Status,
	}
}

func (r *marketingRepository) couponToDomain(c dao.Coupon) domain.Coupon {
	return domain.Coupon{
		ID: c.ID, TenantID: c.TenantID, Name: c.Name, Type: c.Type,
		Threshold: c.Threshold, DiscountValue: c.DiscountValue,
		TotalCount: c.TotalCount, ReceivedCount: c.ReceivedCount,
		UsedCount: c.UsedCount, PerLimit: c.PerLimit,
		StartTime: time.UnixMilli(c.StartTime), EndTime: time.UnixMilli(c.EndTime),
		ScopeType: c.ScopeType, ScopeIDs: c.ScopeIDs, Status: c.Status,
		Ctime: time.UnixMilli(c.Ctime),
	}
}

func (r *marketingRepository) userCouponToDomain(uc dao.UserCoupon) domain.UserCoupon {
	return domain.UserCoupon{
		ID: uc.ID, UserID: uc.UserID, CouponID: uc.CouponID,
		TenantID: uc.TenantID, Status: uc.Status, OrderID: uc.OrderID,
		ReceiveTime: time.UnixMilli(uc.ReceiveTime), UseTime: time.UnixMilli(uc.UseTime),
	}
}

func (r *marketingRepository) seckillActivityToDAO(a domain.SeckillActivity) dao.SeckillActivity {
	return dao.SeckillActivity{
		ID: a.ID, TenantID: a.TenantID, Name: a.Name,
		StartTime: a.StartTime.UnixMilli(), EndTime: a.EndTime.UnixMilli(),
		Status: a.Status,
	}
}

func (r *marketingRepository) seckillActivityToDomain(a dao.SeckillActivity, items []dao.SeckillItem) domain.SeckillActivity {
	domainItems := make([]domain.SeckillItem, 0, len(items))
	for _, item := range items {
		domainItems = append(domainItems, r.seckillItemToDomain(item))
	}
	return domain.SeckillActivity{
		ID: a.ID, TenantID: a.TenantID, Name: a.Name,
		StartTime: time.UnixMilli(a.StartTime), EndTime: time.UnixMilli(a.EndTime),
		Status: a.Status, Items: domainItems,
	}
}

func (r *marketingRepository) seckillItemToDAO(item domain.SeckillItem) dao.SeckillItem {
	return dao.SeckillItem{
		ID: item.ID, ActivityID: item.ActivityID, TenantID: item.TenantID,
		SkuID: item.SkuID, SeckillPrice: item.SeckillPrice,
		SeckillStock: item.SeckillStock, PerLimit: item.PerLimit,
	}
}

func (r *marketingRepository) seckillItemToDomain(item dao.SeckillItem) domain.SeckillItem {
	return domain.SeckillItem{
		ID: item.ID, ActivityID: item.ActivityID, TenantID: item.TenantID,
		SkuID: item.SkuID, SeckillPrice: item.SeckillPrice,
		SeckillStock: item.SeckillStock, PerLimit: item.PerLimit,
	}
}

func (r *marketingRepository) promotionToDAO(rule domain.PromotionRule) dao.PromotionRule {
	return dao.PromotionRule{
		ID: rule.ID, TenantID: rule.TenantID, Name: rule.Name, Type: rule.Type,
		Threshold: rule.Threshold, DiscountValue: rule.DiscountValue,
		StartTime: rule.StartTime.UnixMilli(), EndTime: rule.EndTime.UnixMilli(),
		Status: rule.Status,
	}
}

func (r *marketingRepository) promotionToDomain(rule dao.PromotionRule) domain.PromotionRule {
	return domain.PromotionRule{
		ID: rule.ID, TenantID: rule.TenantID, Name: rule.Name, Type: rule.Type,
		Threshold: rule.Threshold, DiscountValue: rule.DiscountValue,
		StartTime: time.UnixMilli(rule.StartTime), EndTime: time.UnixMilli(rule.EndTime),
		Status: rule.Status,
	}
}

// ==================== Errors ====================

var ErrCouponStockNotEnough = fmt.Errorf("优惠券库存不足")
```

> 注意：repository/marketing.go 需要 import "fmt"。

**Step 4: 验证编译**

```bash
go build ./marketing/...
```

---

## Task 3: Service（marketing-svc 业务逻辑层）

**Files:**
- Create: `marketing/service/marketing.go`

**Step 1: 创建 Service**

Create `marketing/service/marketing.go`:

```go
package service

import (
	"context"
	"errors"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/marketing/domain"
	"github.com/rermrf/mall/marketing/events"
	"github.com/rermrf/mall/marketing/repository"
)

var (
	ErrCouponNotFound    = errors.New("优惠券不存在")
	ErrCouponExpired     = errors.New("优惠券已过期")
	ErrCouponLimitExceed = errors.New("领取次数已达上限")
	ErrSeckillNotActive  = errors.New("秒杀活动未开始或已结束")
	ErrSeckillStockOut   = errors.New("秒杀库存不足")
	ErrSeckillDuplicate  = errors.New("已参与过该秒杀")
)

type MarketingService interface {
	// 优惠券
	CreateCoupon(ctx context.Context, c domain.Coupon) (domain.Coupon, error)
	UpdateCoupon(ctx context.Context, c domain.Coupon) error
	ListCoupons(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.Coupon, int64, error)
	ReceiveCoupon(ctx context.Context, userId, couponId int64) error
	ListUserCoupons(ctx context.Context, userId, tenantId int64, status int32) ([]domain.UserCoupon, error)
	UseCoupon(ctx context.Context, userCouponId, orderId int64) error
	ReleaseCoupon(ctx context.Context, userCouponId int64) error
	CalculateDiscount(ctx context.Context, tenantId, userId, couponId, totalAmount int64, categoryIds []int64) (domain.DiscountResult, error)
	// 秒杀
	CreateSeckillActivity(ctx context.Context, a domain.SeckillActivity) (domain.SeckillActivity, error)
	UpdateSeckillActivity(ctx context.Context, a domain.SeckillActivity) error
	ListSeckillActivities(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SeckillActivity, int64, error)
	GetSeckillActivity(ctx context.Context, id int64) (domain.SeckillActivity, error)
	Seckill(ctx context.Context, userId, itemId int64) (bool, string, error)
	// 满减
	CreatePromotionRule(ctx context.Context, r domain.PromotionRule) (domain.PromotionRule, error)
	UpdatePromotionRule(ctx context.Context, r domain.PromotionRule) error
	ListPromotionRules(ctx context.Context, tenantId int64, status int32) ([]domain.PromotionRule, error)
}

type marketingService struct {
	repo     repository.MarketingRepository
	producer events.Producer
	l        logger.Logger
}

func NewMarketingService(repo repository.MarketingRepository, producer events.Producer, l logger.Logger) MarketingService {
	return &marketingService{repo: repo, producer: producer, l: l}
}

// ==================== 优惠券 ====================

func (s *marketingService) CreateCoupon(ctx context.Context, c domain.Coupon) (domain.Coupon, error) {
	return s.repo.CreateCoupon(ctx, c)
}

func (s *marketingService) UpdateCoupon(ctx context.Context, c domain.Coupon) error {
	return s.repo.UpdateCoupon(ctx, c)
}

func (s *marketingService) ListCoupons(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.Coupon, int64, error) {
	return s.repo.ListCoupons(ctx, tenantId, status, page, pageSize)
}

func (s *marketingService) ReceiveCoupon(ctx context.Context, userId, couponId int64) error {
	coupon, err := s.repo.FindCouponById(ctx, couponId)
	if err != nil {
		return ErrCouponNotFound
	}
	// 检查优惠券状态
	if coupon.Status != 2 {
		return ErrCouponExpired
	}
	// 检查领取限制
	if coupon.PerLimit > 0 {
		count, err := s.repo.CountUserCoupon(ctx, userId, couponId)
		if err != nil {
			return err
		}
		if count >= int64(coupon.PerLimit) {
			return ErrCouponLimitExceed
		}
	}
	return s.repo.ReceiveCoupon(ctx, userId, couponId, coupon.TenantID)
}

func (s *marketingService) ListUserCoupons(ctx context.Context, userId, tenantId int64, status int32) ([]domain.UserCoupon, error) {
	return s.repo.ListUserCoupons(ctx, userId, tenantId, status)
}

func (s *marketingService) UseCoupon(ctx context.Context, userCouponId, orderId int64) error {
	return s.repo.UseCoupon(ctx, userCouponId, orderId)
}

func (s *marketingService) ReleaseCoupon(ctx context.Context, userCouponId int64) error {
	return s.repo.ReleaseCoupon(ctx, userCouponId)
}

func (s *marketingService) CalculateDiscount(ctx context.Context, tenantId, userId, couponId, totalAmount int64, categoryIds []int64) (domain.DiscountResult, error) {
	var couponDiscount int64
	// 计算优惠券优惠
	if couponId > 0 {
		uc, err := s.repo.FindUserCouponById(ctx, couponId)
		if err == nil && uc.Status == 1 {
			coupon, err := s.repo.FindCouponById(ctx, uc.CouponID)
			if err == nil {
				couponDiscount = s.calcCouponDiscount(coupon, totalAmount)
			}
		}
	}
	// 计算满减优惠
	var promotionDiscount int64
	rules, err := s.repo.ListActivePromotionRules(ctx, tenantId)
	if err == nil {
		for _, rule := range rules {
			promotionDiscount += s.calcPromotionDiscount(rule, totalAmount)
		}
	}
	totalDiscount := couponDiscount + promotionDiscount
	payAmount := totalAmount - totalDiscount
	if payAmount < 0 {
		payAmount = 0
	}
	return domain.DiscountResult{
		CouponDiscount:    couponDiscount,
		PromotionDiscount: promotionDiscount,
		TotalDiscount:     totalDiscount,
		PayAmount:         payAmount,
	}, nil
}

func (s *marketingService) calcCouponDiscount(c domain.Coupon, amount int64) int64 {
	if c.Threshold > 0 && amount < c.Threshold {
		return 0
	}
	switch c.Type {
	case 1: // 满减
		return c.DiscountValue
	case 2: // 折扣
		return amount - amount*c.DiscountValue/100
	case 3: // 无门槛
		return c.DiscountValue
	}
	return 0
}

func (s *marketingService) calcPromotionDiscount(r domain.PromotionRule, amount int64) int64 {
	if amount < r.Threshold {
		return 0
	}
	switch r.Type {
	case 1: // 满减
		return r.DiscountValue
	case 2: // 满折
		return amount - amount*r.DiscountValue/100
	}
	return 0
}

// ==================== 秒杀 ====================

func (s *marketingService) CreateSeckillActivity(ctx context.Context, a domain.SeckillActivity) (domain.SeckillActivity, error) {
	return s.repo.CreateSeckillActivity(ctx, a)
}

func (s *marketingService) UpdateSeckillActivity(ctx context.Context, a domain.SeckillActivity) error {
	return s.repo.UpdateSeckillActivity(ctx, a)
}

func (s *marketingService) ListSeckillActivities(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SeckillActivity, int64, error) {
	return s.repo.ListSeckillActivities(ctx, tenantId, status, page, pageSize)
}

func (s *marketingService) GetSeckillActivity(ctx context.Context, id int64) (domain.SeckillActivity, error) {
	return s.repo.FindSeckillActivityById(ctx, id)
}

func (s *marketingService) Seckill(ctx context.Context, userId, itemId int64) (bool, string, error) {
	// 查询秒杀商品
	item, err := s.repo.FindSeckillItemById(ctx, itemId)
	if err != nil {
		return false, "秒杀商品不存在", nil
	}
	// Redis+Lua 原子扣减
	result, err := s.repo.Seckill(ctx, itemId, userId, item.PerLimit)
	if err != nil {
		return false, "系统繁忙", err
	}
	switch result {
	case 1:
		return false, "秒杀库存不足", nil
	case 2:
		return false, "您已参与过该秒杀", nil
	case 3:
		return false, "超出限购数量", nil
	}
	// 写入秒杀订单记录
	_, err = s.repo.CreateSeckillOrder(ctx, domain.SeckillOrder{
		UserID:   userId,
		ItemID:   itemId,
		TenantID: item.TenantID,
		Status:   1, // 排队中
	})
	if err != nil {
		s.l.Error("创建秒杀订单记录失败", logger.Error(err))
	}
	// 发 Kafka 事件
	err = s.producer.ProduceSeckillSuccess(ctx, events.SeckillSuccessEvent{
		UserId:       userId,
		ItemId:       itemId,
		SkuId:        item.SkuID,
		SeckillPrice: item.SeckillPrice,
		TenantId:     item.TenantID,
	})
	if err != nil {
		s.l.Error("发送秒杀成功事件失败", logger.Error(err))
		return false, "系统繁忙", err
	}
	return true, "秒杀成功，订单创建中", nil
}

// ==================== 满减 ====================

func (s *marketingService) CreatePromotionRule(ctx context.Context, r domain.PromotionRule) (domain.PromotionRule, error) {
	return s.repo.CreatePromotionRule(ctx, r)
}

func (s *marketingService) UpdatePromotionRule(ctx context.Context, r domain.PromotionRule) error {
	return s.repo.UpdatePromotionRule(ctx, r)
}

func (s *marketingService) ListPromotionRules(ctx context.Context, tenantId int64, status int32) ([]domain.PromotionRule, error) {
	return s.repo.ListPromotionRules(ctx, tenantId, status)
}
```

**Step 2: 验证编译**

```bash
go build ./marketing/...
```

---

## Task 4: gRPC Handler（marketing-svc 接口层）

**Files:**
- Create: `marketing/grpc/marketing.go`

**Step 1: 创建 gRPC Handler**

Create `marketing/grpc/marketing.go`:

```go
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	marketingv1 "github.com/rermrf/mall/api/proto/gen/marketing/v1"
	"github.com/rermrf/mall/marketing/domain"
	"github.com/rermrf/mall/marketing/service"
)

type MarketingGRPCServer struct {
	marketingv1.UnimplementedMarketingServiceServer
	svc service.MarketingService
}

func NewMarketingGRPCServer(svc service.MarketingService) *MarketingGRPCServer {
	return &MarketingGRPCServer{svc: svc}
}

func (s *MarketingGRPCServer) Register(server *grpc.Server) {
	marketingv1.RegisterMarketingServiceServer(server, s)
}

// ==================== 优惠券 ====================

func (s *MarketingGRPCServer) CreateCoupon(ctx context.Context, req *marketingv1.CreateCouponRequest) (*marketingv1.CreateCouponResponse, error) {
	c := req.GetCoupon()
	coupon, err := s.svc.CreateCoupon(ctx, domain.Coupon{
		TenantID: c.GetTenantId(), Name: c.GetName(), Type: c.GetType(),
		Threshold: c.GetThreshold(), DiscountValue: c.GetDiscountValue(),
		TotalCount: c.GetTotalCount(), PerLimit: c.GetPerLimit(),
		StartTime: time.UnixMilli(c.GetStartTime()), EndTime: time.UnixMilli(c.GetEndTime()),
		ScopeType: c.GetScopeType(), ScopeIDs: c.GetScopeIds(), Status: c.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.CreateCouponResponse{Id: coupon.ID}, nil
}

func (s *MarketingGRPCServer) UpdateCoupon(ctx context.Context, req *marketingv1.UpdateCouponRequest) (*marketingv1.UpdateCouponResponse, error) {
	c := req.GetCoupon()
	err := s.svc.UpdateCoupon(ctx, domain.Coupon{
		ID: c.GetId(), TenantID: c.GetTenantId(), Name: c.GetName(), Type: c.GetType(),
		Threshold: c.GetThreshold(), DiscountValue: c.GetDiscountValue(),
		TotalCount: c.GetTotalCount(), PerLimit: c.GetPerLimit(),
		StartTime: time.UnixMilli(c.GetStartTime()), EndTime: time.UnixMilli(c.GetEndTime()),
		ScopeType: c.GetScopeType(), ScopeIDs: c.GetScopeIds(), Status: c.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.UpdateCouponResponse{}, nil
}

func (s *MarketingGRPCServer) ListCoupons(ctx context.Context, req *marketingv1.ListCouponsRequest) (*marketingv1.ListCouponsResponse, error) {
	coupons, total, err := s.svc.ListCoupons(ctx, req.GetTenantId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	pbCoupons := make([]*marketingv1.Coupon, 0, len(coupons))
	for _, c := range coupons {
		pbCoupons = append(pbCoupons, toCouponDTO(c))
	}
	return &marketingv1.ListCouponsResponse{Coupons: pbCoupons, Total: total}, nil
}

func (s *MarketingGRPCServer) ReceiveCoupon(ctx context.Context, req *marketingv1.ReceiveCouponRequest) (*marketingv1.ReceiveCouponResponse, error) {
	err := s.svc.ReceiveCoupon(ctx, req.GetUserId(), req.GetCouponId())
	if err != nil {
		return nil, err
	}
	return &marketingv1.ReceiveCouponResponse{}, nil
}

func (s *MarketingGRPCServer) ListUserCoupons(ctx context.Context, req *marketingv1.ListUserCouponsRequest) (*marketingv1.ListUserCouponsResponse, error) {
	ucs, err := s.svc.ListUserCoupons(ctx, req.GetUserId(), req.GetTenantId(), req.GetStatus())
	if err != nil {
		return nil, err
	}
	pbUCs := make([]*marketingv1.UserCoupon, 0, len(ucs))
	for _, uc := range ucs {
		pbUCs = append(pbUCs, toUserCouponDTO(uc))
	}
	return &marketingv1.ListUserCouponsResponse{Coupons: pbUCs}, nil
}

func (s *MarketingGRPCServer) UseCoupon(ctx context.Context, req *marketingv1.UseCouponRequest) (*marketingv1.UseCouponResponse, error) {
	err := s.svc.UseCoupon(ctx, req.GetUserCouponId(), req.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &marketingv1.UseCouponResponse{}, nil
}

func (s *MarketingGRPCServer) ReleaseCoupon(ctx context.Context, req *marketingv1.ReleaseCouponRequest) (*marketingv1.ReleaseCouponResponse, error) {
	err := s.svc.ReleaseCoupon(ctx, req.GetUserCouponId())
	if err != nil {
		return nil, err
	}
	return &marketingv1.ReleaseCouponResponse{}, nil
}

func (s *MarketingGRPCServer) CalculateDiscount(ctx context.Context, req *marketingv1.CalculateDiscountRequest) (*marketingv1.CalculateDiscountResponse, error) {
	result, err := s.svc.CalculateDiscount(ctx, req.GetTenantId(), req.GetUserId(), req.GetCouponId(), req.GetTotalAmount(), req.GetCategoryIds())
	if err != nil {
		return nil, err
	}
	return &marketingv1.CalculateDiscountResponse{
		Result: &marketingv1.DiscountResult{
			CouponDiscount:    result.CouponDiscount,
			PromotionDiscount: result.PromotionDiscount,
			TotalDiscount:     result.TotalDiscount,
			PayAmount:         result.PayAmount,
		},
	}, nil
}

// ==================== 秒杀 ====================

func (s *MarketingGRPCServer) CreateSeckillActivity(ctx context.Context, req *marketingv1.CreateSeckillActivityRequest) (*marketingv1.CreateSeckillActivityResponse, error) {
	a := req.GetActivity()
	items := make([]domain.SeckillItem, 0, len(a.GetItems()))
	for _, item := range a.GetItems() {
		items = append(items, domain.SeckillItem{
			SkuID: item.GetSkuId(), SeckillPrice: item.GetSeckillPrice(),
			SeckillStock: item.GetSeckillStock(), PerLimit: item.GetPerLimit(),
		})
	}
	activity, err := s.svc.CreateSeckillActivity(ctx, domain.SeckillActivity{
		TenantID: a.GetTenantId(), Name: a.GetName(),
		StartTime: time.UnixMilli(a.GetStartTime()), EndTime: time.UnixMilli(a.GetEndTime()),
		Status: a.GetStatus(), Items: items,
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.CreateSeckillActivityResponse{Id: activity.ID}, nil
}

func (s *MarketingGRPCServer) UpdateSeckillActivity(ctx context.Context, req *marketingv1.UpdateSeckillActivityRequest) (*marketingv1.UpdateSeckillActivityResponse, error) {
	a := req.GetActivity()
	items := make([]domain.SeckillItem, 0, len(a.GetItems()))
	for _, item := range a.GetItems() {
		items = append(items, domain.SeckillItem{
			SkuID: item.GetSkuId(), SeckillPrice: item.GetSeckillPrice(),
			SeckillStock: item.GetSeckillStock(), PerLimit: item.GetPerLimit(),
		})
	}
	err := s.svc.UpdateSeckillActivity(ctx, domain.SeckillActivity{
		ID: a.GetId(), TenantID: a.GetTenantId(), Name: a.GetName(),
		StartTime: time.UnixMilli(a.GetStartTime()), EndTime: time.UnixMilli(a.GetEndTime()),
		Status: a.GetStatus(), Items: items,
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.UpdateSeckillActivityResponse{}, nil
}

func (s *MarketingGRPCServer) ListSeckillActivities(ctx context.Context, req *marketingv1.ListSeckillActivitiesRequest) (*marketingv1.ListSeckillActivitiesResponse, error) {
	activities, total, err := s.svc.ListSeckillActivities(ctx, req.GetTenantId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	pbActivities := make([]*marketingv1.SeckillActivity, 0, len(activities))
	for _, a := range activities {
		pbActivities = append(pbActivities, toSeckillActivityDTO(a))
	}
	return &marketingv1.ListSeckillActivitiesResponse{Activities: pbActivities, Total: total}, nil
}

func (s *MarketingGRPCServer) GetSeckillActivity(ctx context.Context, req *marketingv1.GetSeckillActivityRequest) (*marketingv1.GetSeckillActivityResponse, error) {
	activity, err := s.svc.GetSeckillActivity(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &marketingv1.GetSeckillActivityResponse{Activity: toSeckillActivityDTO(activity)}, nil
}

func (s *MarketingGRPCServer) Seckill(ctx context.Context, req *marketingv1.SeckillRequest) (*marketingv1.SeckillResponse, error) {
	success, msg, err := s.svc.Seckill(ctx, req.GetUserId(), req.GetItemId())
	if err != nil {
		return nil, err
	}
	return &marketingv1.SeckillResponse{Success: success, Message: msg}, nil
}

// ==================== 满减 ====================

func (s *MarketingGRPCServer) CreatePromotionRule(ctx context.Context, req *marketingv1.CreatePromotionRuleRequest) (*marketingv1.CreatePromotionRuleResponse, error) {
	r := req.GetRule()
	rule, err := s.svc.CreatePromotionRule(ctx, domain.PromotionRule{
		TenantID: r.GetTenantId(), Name: r.GetName(), Type: r.GetType(),
		Threshold: r.GetThreshold(), DiscountValue: r.GetDiscountValue(),
		StartTime: time.UnixMilli(r.GetStartTime()), EndTime: time.UnixMilli(r.GetEndTime()),
		Status: r.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.CreatePromotionRuleResponse{Id: rule.ID}, nil
}

func (s *MarketingGRPCServer) UpdatePromotionRule(ctx context.Context, req *marketingv1.UpdatePromotionRuleRequest) (*marketingv1.UpdatePromotionRuleResponse, error) {
	r := req.GetRule()
	err := s.svc.UpdatePromotionRule(ctx, domain.PromotionRule{
		ID: r.GetId(), TenantID: r.GetTenantId(), Name: r.GetName(), Type: r.GetType(),
		Threshold: r.GetThreshold(), DiscountValue: r.GetDiscountValue(),
		StartTime: time.UnixMilli(r.GetStartTime()), EndTime: time.UnixMilli(r.GetEndTime()),
		Status: r.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.UpdatePromotionRuleResponse{}, nil
}

func (s *MarketingGRPCServer) ListPromotionRules(ctx context.Context, req *marketingv1.ListPromotionRulesRequest) (*marketingv1.ListPromotionRulesResponse, error) {
	rules, err := s.svc.ListPromotionRules(ctx, req.GetTenantId(), req.GetStatus())
	if err != nil {
		return nil, err
	}
	pbRules := make([]*marketingv1.PromotionRule, 0, len(rules))
	for _, r := range rules {
		pbRules = append(pbRules, toPromotionRuleDTO(r))
	}
	return &marketingv1.ListPromotionRulesResponse{Rules: pbRules}, nil
}

// ==================== DTO Converters ====================

func toCouponDTO(c domain.Coupon) *marketingv1.Coupon {
	return &marketingv1.Coupon{
		Id: c.ID, TenantId: c.TenantID, Name: c.Name, Type: c.Type,
		Threshold: c.Threshold, DiscountValue: c.DiscountValue,
		TotalCount: c.TotalCount, ReceivedCount: c.ReceivedCount,
		UsedCount: c.UsedCount, PerLimit: c.PerLimit,
		StartTime: c.StartTime.UnixMilli(), EndTime: c.EndTime.UnixMilli(),
		ScopeType: c.ScopeType, ScopeIds: c.ScopeIDs, Status: c.Status,
		Ctime: timestamppb.New(c.Ctime),
	}
}

func toUserCouponDTO(uc domain.UserCoupon) *marketingv1.UserCoupon {
	return &marketingv1.UserCoupon{
		Id: uc.ID, UserId: uc.UserID, CouponId: uc.CouponID,
		TenantId: uc.TenantID, Status: uc.Status, OrderId: uc.OrderID,
		ReceiveTime: uc.ReceiveTime.UnixMilli(), UseTime: uc.UseTime.UnixMilli(),
		Coupon: toCouponDTO(uc.Coupon),
	}
}

func toSeckillActivityDTO(a domain.SeckillActivity) *marketingv1.SeckillActivity {
	items := make([]*marketingv1.SeckillItem, 0, len(a.Items))
	for _, item := range a.Items {
		items = append(items, &marketingv1.SeckillItem{
			Id: item.ID, ActivityId: item.ActivityID, TenantId: item.TenantID,
			SkuId: item.SkuID, SeckillPrice: item.SeckillPrice,
			SeckillStock: item.SeckillStock, PerLimit: item.PerLimit,
		})
	}
	return &marketingv1.SeckillActivity{
		Id: a.ID, TenantId: a.TenantID, Name: a.Name,
		StartTime: a.StartTime.UnixMilli(), EndTime: a.EndTime.UnixMilli(),
		Status: a.Status, Items: items,
	}
}

func toPromotionRuleDTO(r domain.PromotionRule) *marketingv1.PromotionRule {
	return &marketingv1.PromotionRule{
		Id: r.ID, TenantId: r.TenantID, Name: r.Name, Type: r.Type,
		Threshold: r.Threshold, DiscountValue: r.DiscountValue,
		StartTime: r.StartTime.UnixMilli(), EndTime: r.EndTime.UnixMilli(),
		Status: r.Status,
	}
}
```

**Step 2: 验证编译**

```bash
go build ./marketing/...
```

---

## Task 5: Events（Kafka Producer + Consumer）

**Files:**
- Create: `marketing/events/types.go`
- Create: `marketing/events/producer.go`
- Create: `marketing/events/consumer.go`

**Step 1: 创建事件类型**

Create `marketing/events/types.go`:

```go
package events

const (
	TopicSeckillSuccess  = "seckill_success"
	TopicOrderCancelled  = "order_cancelled"
)

type SeckillSuccessEvent struct {
	UserId       int64 `json:"user_id"`
	ItemId       int64 `json:"item_id"`
	SkuId        int64 `json:"sku_id"`
	SeckillPrice int64 `json:"seckill_price"`
	TenantId     int64 `json:"tenant_id"`
}

type OrderCancelledEvent struct {
	OrderNo  string `json:"order_no"`
	TenantID int64  `json:"tenant_id"`
	Reason   string `json:"reason"`
}
```

**Step 2: 创建 Producer**

Create `marketing/events/producer.go`:

```go
package events

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

type Producer interface {
	ProduceSeckillSuccess(ctx context.Context, evt SeckillSuccessEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceSeckillSuccess(ctx context.Context, evt SeckillSuccessEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicSeckillSuccess,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
```

**Step 3: 创建 Consumer**

Create `marketing/events/consumer.go`:

```go
package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/saramax"
)

// OrderCancelledConsumer 消费 order_cancelled 事件，释放优惠券
type OrderCancelledConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt OrderCancelledEvent) error
}

func NewOrderCancelledConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	handler func(ctx context.Context, evt OrderCancelledEvent) error,
) *OrderCancelledConsumer {
	return &OrderCancelledConsumer{client: client, l: l, handler: handler}
}

func (c *OrderCancelledConsumer) Start() error {
	h := saramax.NewHandler[OrderCancelledEvent](c.l, c.Consume)
	go func() {
		for {
			err := c.client.Consume(context.Background(), []string{TopicOrderCancelled}, h)
			if err != nil {
				c.l.Error("消费 order_cancelled 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderCancelledConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderCancelledEvent) error {
	return c.handler(context.Background(), evt)
}
```

**Step 4: 验证编译**

```bash
go build ./marketing/...
```

---

## Task 6: IoC + Wire + Config + Main（marketing-svc 基础设施）

**Files:**
- Create: `marketing/ioc/db.go`
- Create: `marketing/ioc/redis.go`
- Create: `marketing/ioc/logger.go`
- Create: `marketing/ioc/grpc.go`
- Create: `marketing/ioc/kafka.go`
- Create: `marketing/wire.go`
- Create: `marketing/app.go`
- Create: `marketing/main.go`
- Create: `marketing/config/dev.yaml`
- Generate: `marketing/wire_gen.go`

**Step 1: 创建 IoC — DB**

Create `marketing/ioc/db.go`:

```go
package ioc

import (
	"fmt"

	"github.com/rermrf/mall/marketing/repository/dao"
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

**Step 2: 创建 IoC — Redis**

Create `marketing/ioc/redis.go`:

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

**Step 3: 创建 IoC — Logger**

Create `marketing/ioc/logger.go`:

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

**Step 4: 创建 IoC — gRPC + etcd**

Create `marketing/ioc/grpc.go`:

```go
package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	mgrpc "github.com/rermrf/mall/marketing/grpc"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

func InitEtcdClient() *clientv3.Client {
	var cfg struct {
		Addrs []string `yaml:"addrs"`
	}
	err := viper.UnmarshalKey("etcd", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 etcd 配置失败: %w", err))
	}
	client, err := clientv3.New(clientv3.Config{Endpoints: cfg.Addrs})
	if err != nil {
		panic(fmt.Errorf("连接 etcd 失败: %w", err))
	}
	return client
}

func InitGRPCServer(marketingServer *mgrpc.MarketingGRPCServer, l logger.Logger) *grpcx.Server {
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
	marketingServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "marketing",
		L:         l,
	}
}
```

**Step 5: 创建 IoC — Kafka**

Create `marketing/ioc/kafka.go`:

```go
package ioc

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/marketing/events"
	"github.com/rermrf/mall/marketing/service"
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
	cg, err := sarama.NewConsumerGroupFromClient("marketing-svc", client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka ConsumerGroup 失败: %w", err))
	}
	return cg
}

func NewOrderCancelledConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.MarketingService,
) *events.OrderCancelledConsumer {
	return events.NewOrderCancelledConsumer(cg, l, func(ctx context.Context, evt events.OrderCancelledEvent) error {
		// 订单取消时释放优惠券的逻辑
		// 注意：当前 OrderCancelledEvent 没有 coupon_id，需要通过 order_no 查询
		// 这里简化处理，实际需要 order-svc 在事件中携带 coupon_id
		l.Info("收到订单取消事件", logger.String("orderNo", evt.OrderNo))
		return nil
	})
}

func InitConsumers(
	cancelledConsumer *events.OrderCancelledConsumer,
) []saramax.Consumer {
	return []saramax.Consumer{cancelledConsumer}
}
```

**Step 6: 创建 App**

Create `marketing/app.go`:

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

**Step 7: 创建 Wire DI**

Create `marketing/wire.go`:

```go
//go:build wireinject

package main

import (
	"github.com/google/wire"
	mgrpc "github.com/rermrf/mall/marketing/grpc"
	"github.com/rermrf/mall/marketing/ioc"
	"github.com/rermrf/mall/marketing/repository"
	"github.com/rermrf/mall/marketing/repository/cache"
	"github.com/rermrf/mall/marketing/repository/dao"
	"github.com/rermrf/mall/marketing/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
)

var marketingSet = wire.NewSet(
	dao.NewCouponDAO,
	dao.NewUserCouponDAO,
	dao.NewSeckillDAO,
	dao.NewPromotionDAO,
	cache.NewMarketingCache,
	repository.NewMarketingRepository,
	service.NewMarketingService,
	mgrpc.NewMarketingGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitGRPCServer,
	ioc.InitConsumerGroup,
	ioc.NewOrderCancelledConsumer,
	ioc.InitConsumers,
)

func InitApp() *App {
	wire.Build(thirdPartySet, marketingSet, wire.Struct(new(App), "*"))
	return new(App)
}
```

**Step 8: 创建 main.go**

Create `marketing/main.go`:

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

	for _, c := range app.Consumers {
		if err := c.Start(); err != nil {
			panic(fmt.Errorf("启动消费者失败: %w", err))
		}
	}

	go func() {
		if err := app.Server.Serve(); err != nil {
			fmt.Println("gRPC 服务启动失败:", err)
			os.Exit(1)
		}
	}()

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

**Step 9: 创建配置文件**

Create `marketing/config/dev.yaml`:

```yaml
db:
  dsn: "root:wen123...@tcp(rermrf.icu:3306)/mall_marketing?charset=utf8mb4&parseTime=true&loc=Local"

redis:
  addr: "rermrf.icu:6379"
  password: ""
  db: 8

kafka:
  addrs:
    - "rermrf.icu:9094"

grpc:
  port: 8089
  etcdAddrs:
    - "rermrf.icu:2379"

etcd:
  addrs:
    - "rermrf.icu:2379"
```

**Step 10: 生成 Wire 代码并验证**

```bash
cd marketing && wire && cd ..
go build ./marketing/...
go vet ./marketing/...
```

---

## Task 7: Merchant BFF Marketing 接口（10 个端点）

**Files:**
- Create: `merchant-bff/handler/marketing.go`
- Modify: `merchant-bff/ioc/grpc.go` — +InitMarketingClient
- Modify: `merchant-bff/ioc/gin.go` — +marketingHandler + 10 路由
- Modify: `merchant-bff/wire.go` — +InitMarketingClient + NewMarketingHandler
- Regenerate: `merchant-bff/wire_gen.go`

**Step 1: 创建 MarketingHandler**

Create `merchant-bff/handler/marketing.go`:

```go
package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	marketingv1 "github.com/rermrf/mall/api/proto/gen/marketing/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type MarketingHandler struct {
	marketingClient marketingv1.MarketingServiceClient
	l               logger.Logger
}

func NewMarketingHandler(marketingClient marketingv1.MarketingServiceClient, l logger.Logger) *MarketingHandler {
	return &MarketingHandler{
		marketingClient: marketingClient,
		l:               l,
	}
}

// ==================== 优惠券 ====================

type CreateCouponReq struct {
	Name          string `json:"name" binding:"required"`
	Type          int32  `json:"type" binding:"required"`
	Threshold     int64  `json:"threshold"`
	DiscountValue int64  `json:"discount_value" binding:"required"`
	TotalCount    int32  `json:"total_count" binding:"required"`
	PerLimit      int32  `json:"per_limit"`
	StartTime     int64  `json:"start_time" binding:"required"`
	EndTime       int64  `json:"end_time" binding:"required"`
	ScopeType     int32  `json:"scope_type"`
	ScopeIDs      string `json:"scope_ids"`
	Status        int32  `json:"status"`
}

func (h *MarketingHandler) CreateCoupon(ctx *gin.Context, req CreateCouponReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.marketingClient.CreateCoupon(ctx.Request.Context(), &marketingv1.CreateCouponRequest{
		Coupon: &marketingv1.Coupon{
			TenantId: tenantId.(int64), Name: req.Name, Type: req.Type,
			Threshold: req.Threshold, DiscountValue: req.DiscountValue,
			TotalCount: req.TotalCount, PerLimit: req.PerLimit,
			StartTime: req.StartTime, EndTime: req.EndTime,
			ScopeType: req.ScopeType, ScopeIds: req.ScopeIDs, Status: req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建优惠券失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateCouponReq struct {
	Name          string `json:"name"`
	Type          int32  `json:"type"`
	Threshold     int64  `json:"threshold"`
	DiscountValue int64  `json:"discount_value"`
	TotalCount    int32  `json:"total_count"`
	PerLimit      int32  `json:"per_limit"`
	StartTime     int64  `json:"start_time"`
	EndTime       int64  `json:"end_time"`
	ScopeType     int32  `json:"scope_type"`
	ScopeIDs      string `json:"scope_ids"`
	Status        int32  `json:"status"`
}

func (h *MarketingHandler) UpdateCoupon(ctx *gin.Context, req UpdateCouponReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.marketingClient.UpdateCoupon(ctx.Request.Context(), &marketingv1.UpdateCouponRequest{
		Coupon: &marketingv1.Coupon{
			Id: id, TenantId: tenantId.(int64), Name: req.Name, Type: req.Type,
			Threshold: req.Threshold, DiscountValue: req.DiscountValue,
			TotalCount: req.TotalCount, PerLimit: req.PerLimit,
			StartTime: req.StartTime, EndTime: req.EndTime,
			ScopeType: req.ScopeType, ScopeIds: req.ScopeIDs, Status: req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新优惠券失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListCouponsReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *MarketingHandler) ListCoupons(ctx *gin.Context, req ListCouponsReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.marketingClient.ListCoupons(ctx.Request.Context(), &marketingv1.ListCouponsRequest{
		TenantId: tenantId.(int64), Status: req.Status, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询优惠券列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"coupons": resp.GetCoupons(), "total": resp.GetTotal(),
	}}, nil
}

// ==================== 秒杀 ====================

type CreateSeckillReq struct {
	Name      string             `json:"name" binding:"required"`
	StartTime int64              `json:"start_time" binding:"required"`
	EndTime   int64              `json:"end_time" binding:"required"`
	Status    int32              `json:"status"`
	Items     []CreateSeckillItem `json:"items" binding:"required,min=1"`
}

type CreateSeckillItem struct {
	SkuID        int64 `json:"sku_id" binding:"required"`
	SeckillPrice int64 `json:"seckill_price" binding:"required"`
	SeckillStock int32 `json:"seckill_stock" binding:"required"`
	PerLimit     int32 `json:"per_limit"`
}

func (h *MarketingHandler) CreateSeckill(ctx *gin.Context, req CreateSeckillReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	items := make([]*marketingv1.SeckillItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, &marketingv1.SeckillItem{
			SkuId: item.SkuID, SeckillPrice: item.SeckillPrice,
			SeckillStock: item.SeckillStock, PerLimit: item.PerLimit,
		})
	}
	resp, err := h.marketingClient.CreateSeckillActivity(ctx.Request.Context(), &marketingv1.CreateSeckillActivityRequest{
		Activity: &marketingv1.SeckillActivity{
			TenantId: tenantId.(int64), Name: req.Name,
			StartTime: req.StartTime, EndTime: req.EndTime,
			Status: req.Status, Items: items,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建秒杀活动失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateSeckillReq struct {
	Name      string             `json:"name"`
	StartTime int64              `json:"start_time"`
	EndTime   int64              `json:"end_time"`
	Status    int32              `json:"status"`
	Items     []CreateSeckillItem `json:"items"`
}

func (h *MarketingHandler) UpdateSeckill(ctx *gin.Context, req UpdateSeckillReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	items := make([]*marketingv1.SeckillItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, &marketingv1.SeckillItem{
			SkuId: item.SkuID, SeckillPrice: item.SeckillPrice,
			SeckillStock: item.SeckillStock, PerLimit: item.PerLimit,
		})
	}
	_, err := h.marketingClient.UpdateSeckillActivity(ctx.Request.Context(), &marketingv1.UpdateSeckillActivityRequest{
		Activity: &marketingv1.SeckillActivity{
			Id: id, TenantId: tenantId.(int64), Name: req.Name,
			StartTime: req.StartTime, EndTime: req.EndTime,
			Status: req.Status, Items: items,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新秒杀活动失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListSeckillReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *MarketingHandler) ListSeckill(ctx *gin.Context, req ListSeckillReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.marketingClient.ListSeckillActivities(ctx.Request.Context(), &marketingv1.ListSeckillActivitiesRequest{
		TenantId: tenantId.(int64), Status: req.Status, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询秒杀活动列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"activities": resp.GetActivities(), "total": resp.GetTotal(),
	}}, nil
}

func (h *MarketingHandler) GetSeckill(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resp, err := h.marketingClient.GetSeckillActivity(ctx.Request.Context(), &marketingv1.GetSeckillActivityRequest{Id: id})
	if err != nil {
		h.l.Error("查询秒杀活动详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetActivity()})
}

// ==================== 满减 ====================

type CreatePromotionReq struct {
	Name          string `json:"name" binding:"required"`
	Type          int32  `json:"type" binding:"required"`
	Threshold     int64  `json:"threshold" binding:"required"`
	DiscountValue int64  `json:"discount_value" binding:"required"`
	StartTime     int64  `json:"start_time" binding:"required"`
	EndTime       int64  `json:"end_time" binding:"required"`
	Status        int32  `json:"status"`
}

func (h *MarketingHandler) CreatePromotion(ctx *gin.Context, req CreatePromotionReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.marketingClient.CreatePromotionRule(ctx.Request.Context(), &marketingv1.CreatePromotionRuleRequest{
		Rule: &marketingv1.PromotionRule{
			TenantId: tenantId.(int64), Name: req.Name, Type: req.Type,
			Threshold: req.Threshold, DiscountValue: req.DiscountValue,
			StartTime: req.StartTime, EndTime: req.EndTime, Status: req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建满减规则失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdatePromotionReq struct {
	Name          string `json:"name"`
	Type          int32  `json:"type"`
	Threshold     int64  `json:"threshold"`
	DiscountValue int64  `json:"discount_value"`
	StartTime     int64  `json:"start_time"`
	EndTime       int64  `json:"end_time"`
	Status        int32  `json:"status"`
}

func (h *MarketingHandler) UpdatePromotion(ctx *gin.Context, req UpdatePromotionReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.marketingClient.UpdatePromotionRule(ctx.Request.Context(), &marketingv1.UpdatePromotionRuleRequest{
		Rule: &marketingv1.PromotionRule{
			Id: id, TenantId: tenantId.(int64), Name: req.Name, Type: req.Type,
			Threshold: req.Threshold, DiscountValue: req.DiscountValue,
			StartTime: req.StartTime, EndTime: req.EndTime, Status: req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新满减规则失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListPromotionsReq struct {
	Status int32 `form:"status"`
}

func (h *MarketingHandler) ListPromotions(ctx *gin.Context, req ListPromotionsReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.marketingClient.ListPromotionRules(ctx.Request.Context(), &marketingv1.ListPromotionRulesRequest{
		TenantId: tenantId.(int64), Status: req.Status,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询满减规则列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetRules()}, nil
}
```

**Step 2: 修改 merchant-bff/ioc/grpc.go**

在 import 块中添加：
```go
marketingv1 "github.com/rermrf/mall/api/proto/gen/marketing/v1"
```

在文件末尾添加：
```go
func InitMarketingClient(etcdClient *clientv3.Client) marketingv1.MarketingServiceClient {
	conn := initServiceConn(etcdClient, "marketing")
	return marketingv1.NewMarketingServiceClient(conn)
}
```

**Step 3: 修改 merchant-bff/ioc/gin.go**

`InitGinServer` 函数签名添加 `marketingHandler *handler.MarketingHandler` 参数（在 `paymentHandler` 之后）。

在 `auth` 路由组末尾添加 10 个路由：
```go
		// 营销管理 - 优惠券
		auth.POST("/coupons", ginx.WrapBody[handler.CreateCouponReq](l, marketingHandler.CreateCoupon))
		auth.PUT("/coupons/:id", ginx.WrapBody[handler.UpdateCouponReq](l, marketingHandler.UpdateCoupon))
		auth.GET("/coupons", ginx.WrapQuery[handler.ListCouponsReq](l, marketingHandler.ListCoupons))
		// 营销管理 - 秒杀
		auth.POST("/seckill", ginx.WrapBody[handler.CreateSeckillReq](l, marketingHandler.CreateSeckill))
		auth.PUT("/seckill/:id", ginx.WrapBody[handler.UpdateSeckillReq](l, marketingHandler.UpdateSeckill))
		auth.GET("/seckill", ginx.WrapQuery[handler.ListSeckillReq](l, marketingHandler.ListSeckill))
		auth.GET("/seckill/:id", marketingHandler.GetSeckill)
		// 营销管理 - 满减规则
		auth.POST("/promotions", ginx.WrapBody[handler.CreatePromotionReq](l, marketingHandler.CreatePromotion))
		auth.PUT("/promotions/:id", ginx.WrapBody[handler.UpdatePromotionReq](l, marketingHandler.UpdatePromotion))
		auth.GET("/promotions", ginx.WrapQuery[handler.ListPromotionsReq](l, marketingHandler.ListPromotions))
```

**Step 4: 修改 merchant-bff/wire.go**

`thirdPartySet` 添加 `ioc.InitMarketingClient`。
`handlerSet` 添加 `handler.NewMarketingHandler`。

**Step 5: 重新生成 Wire 代码并验证**

```bash
cd merchant-bff && wire && cd ..
go build ./merchant-bff/...
go vet ./merchant-bff/...
```

---

## Task 8: Consumer BFF Marketing 接口（5 个端点）

**Files:**
- Create: `consumer-bff/handler/marketing.go`
- Modify: `consumer-bff/ioc/grpc.go` — +InitMarketingClient
- Modify: `consumer-bff/ioc/gin.go` — +marketingHandler + 5 路由（2 pub + 3 auth）
- Modify: `consumer-bff/wire.go` — +InitMarketingClient + NewMarketingHandler
- Regenerate: `consumer-bff/wire_gen.go`

**Step 1: 创建 MarketingHandler**

Create `consumer-bff/handler/marketing.go`:

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	marketingv1 "github.com/rermrf/mall/api/proto/gen/marketing/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type MarketingHandler struct {
	marketingClient marketingv1.MarketingServiceClient
	l               logger.Logger
}

func NewMarketingHandler(marketingClient marketingv1.MarketingServiceClient, l logger.Logger) *MarketingHandler {
	return &MarketingHandler{
		marketingClient: marketingClient,
		l:               l,
	}
}

// ListAvailableCoupons 可领优惠券列表（公开）
func (h *MarketingHandler) ListAvailableCoupons(ctx *gin.Context) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.marketingClient.ListCoupons(ctx.Request.Context(), &marketingv1.ListCouponsRequest{
		TenantId: tenantId.(int64),
		Status:   2, // 进行中
		Page:     1,
		PageSize: 50,
	})
	if err != nil {
		h.l.Error("查询可领优惠券列表失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetCoupons()})
}

// ReceiveCoupon 领券（需登录）
func (h *MarketingHandler) ReceiveCoupon(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	idStr := ctx.Param("id")
	couponId, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.marketingClient.ReceiveCoupon(ctx.Request.Context(), &marketingv1.ReceiveCouponRequest{
		UserId:   uid.(int64),
		CouponId: couponId,
	})
	if err != nil {
		h.l.Error("领取优惠券失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

// ListMyCoupons 我的优惠券（需登录）
func (h *MarketingHandler) ListMyCoupons(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	tenantId, _ := ctx.Get("tenant_id")
	statusStr := ctx.DefaultQuery("status", "0")
	status, _ := strconv.ParseInt(statusStr, 10, 32)
	resp, err := h.marketingClient.ListUserCoupons(ctx.Request.Context(), &marketingv1.ListUserCouponsRequest{
		UserId:   uid.(int64),
		TenantId: tenantId.(int64),
		Status:   int32(status),
	})
	if err != nil {
		h.l.Error("查询我的优惠券失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetCoupons()})
}

// ListSeckillActivities 秒杀活动列表（公开）
func (h *MarketingHandler) ListSeckillActivities(ctx *gin.Context) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.marketingClient.ListSeckillActivities(ctx.Request.Context(), &marketingv1.ListSeckillActivitiesRequest{
		TenantId: tenantId.(int64),
		Status:   2, // 进行中
		Page:     1,
		PageSize: 50,
	})
	if err != nil {
		h.l.Error("查询秒杀活动列表失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetActivities()})
}

// Seckill 秒杀抢购（需登录）
func (h *MarketingHandler) Seckill(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	itemIdStr := ctx.Param("itemId")
	itemId, _ := strconv.ParseInt(itemIdStr, 10, 64)
	resp, err := h.marketingClient.Seckill(ctx.Request.Context(), &marketingv1.SeckillRequest{
		UserId: uid.(int64),
		ItemId: itemId,
	})
	if err != nil {
		h.l.Error("秒杀请求失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统繁忙"})
		return
	}
	if !resp.GetSuccess() {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: resp.GetMessage()})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"message":  resp.GetMessage(),
		"order_no": resp.GetOrderNo(),
	}})
}
```

**Step 2: 修改 consumer-bff/ioc/grpc.go**

在 import 块中添加：
```go
marketingv1 "github.com/rermrf/mall/api/proto/gen/marketing/v1"
```

在文件末尾添加：
```go
func InitMarketingClient(etcdClient *clientv3.Client) marketingv1.MarketingServiceClient {
	conn := initServiceConn(etcdClient, "marketing")
	return marketingv1.NewMarketingServiceClient(conn)
}
```

**Step 3: 修改 consumer-bff/ioc/gin.go**

`InitGinServer` 函数签名添加 `marketingHandler *handler.MarketingHandler` 参数（在 `searchHandler` 之后）。

在 `pub` 路由组中添加 2 个公开路由：
```go
		// 营销（公开）
		pub.GET("/coupons", marketingHandler.ListAvailableCoupons)
		pub.GET("/seckill", marketingHandler.ListSeckillActivities)
```

在 `auth` 路由组末尾添加 3 个需登录路由：
```go
		// 营销（需登录）
		auth.POST("/coupons/:id/receive", marketingHandler.ReceiveCoupon)
		auth.GET("/coupons/mine", marketingHandler.ListMyCoupons)
		auth.POST("/seckill/:itemId", marketingHandler.Seckill)
```

**Step 4: 修改 consumer-bff/wire.go**

`thirdPartySet` 添加 `ioc.InitMarketingClient`。
`handlerSet` 添加 `handler.NewMarketingHandler`。

**Step 5: 重新生成 Wire 代码并验证**

```bash
cd consumer-bff && wire && cd ..
go build ./consumer-bff/...
go vet ./consumer-bff/...
```

---

## 文件清单总览

| # | 文件路径 | 操作 | Task |
|---|---------|------|------|
| 1 | `marketing/domain/marketing.go` | 新建 | 1 |
| 2 | `marketing/repository/dao/marketing.go` | 新建 | 1 |
| 3 | `marketing/repository/dao/init.go` | 新建 | 1 |
| 4 | `marketing/repository/cache/lua/seckill.lua` | 新建 | 2 |
| 5 | `marketing/repository/cache/marketing.go` | 新建 | 2 |
| 6 | `marketing/repository/marketing.go` | 新建 | 2 |
| 7 | `marketing/service/marketing.go` | 新建 | 3 |
| 8 | `marketing/grpc/marketing.go` | 新建 | 4 |
| 9 | `marketing/events/types.go` | 新建 | 5 |
| 10 | `marketing/events/producer.go` | 新建 | 5 |
| 11 | `marketing/events/consumer.go` | 新建 | 5 |
| 12 | `marketing/ioc/db.go` | 新建 | 6 |
| 13 | `marketing/ioc/redis.go` | 新建 | 6 |
| 14 | `marketing/ioc/logger.go` | 新建 | 6 |
| 15 | `marketing/ioc/grpc.go` | 新建 | 6 |
| 16 | `marketing/ioc/kafka.go` | 新建 | 6 |
| 17 | `marketing/wire.go` | 新建 | 6 |
| 18 | `marketing/app.go` | 新建 | 6 |
| 19 | `marketing/main.go` | 新建 | 6 |
| 20 | `marketing/config/dev.yaml` | 新建 | 6 |
| 21 | `marketing/wire_gen.go` | 生成 | 6 |
| 22 | `merchant-bff/handler/marketing.go` | 新建 | 7 |
| 23 | `merchant-bff/ioc/grpc.go` | 修改 | 7 |
| 24 | `merchant-bff/ioc/gin.go` | 修改 | 7 |
| 25 | `merchant-bff/wire.go` | 修改 | 7 |
| 26 | `merchant-bff/wire_gen.go` | 重新生成 | 7 |
| 27 | `consumer-bff/handler/marketing.go` | 新建 | 8 |
| 28 | `consumer-bff/ioc/grpc.go` | 修改 | 8 |
| 29 | `consumer-bff/ioc/gin.go` | 修改 | 8 |
| 30 | `consumer-bff/wire.go` | 修改 | 8 |
| 31 | `consumer-bff/wire_gen.go` | 重新生成 | 8 |

共 31 个文件（20 新建 + 5 修改 + 1 生成 + 5 重新生成）

## 验证

```bash
go build ./marketing/...
go vet ./marketing/...
go build ./merchant-bff/...
go vet ./merchant-bff/...
go build ./consumer-bff/...
go vet ./consumer-bff/...
```
