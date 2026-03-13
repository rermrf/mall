# Tenant Service (tenant-svc) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the tenant microservice with 17 gRPC RPCs covering tenant CRUD + approval, plan management, quota control, and shop/domain management.

**Architecture:** DDD layered architecture identical to user-svc: domain → dao → cache → repository → service → grpc → events → ioc → wire → main. Separate database `mall_tenant`, gRPC port 8082, etcd service name `tenant`. Kafka producer-only (no consumers).

**Tech Stack:** Go 1.25, gRPC, GORM/MySQL, Redis, Kafka (Sarama), Wire DI, Viper config, etcd service registration, `github.com/rermrf/emo/logger`

**Design Doc:** `docs/plans/2026-03-07-tenant-svc-design.md`

---

## Task 1: Domain Layer

**Files:**
- Create: `tenant/domain/tenant.go`
- Create: `tenant/domain/plan.go`
- Create: `tenant/domain/shop.go`

**Step 1: Create tenant domain**

```go
// tenant/domain/tenant.go
package domain

import "time"

type Tenant struct {
	ID              int64
	Name            string
	ContactName     string
	ContactPhone    string
	BusinessLicense string
	Status          TenantStatus
	PlanID          int64
	PlanExpireTime  int64 // unix timestamp seconds
	Ctime           time.Time
	Utime           time.Time
}

type TenantStatus uint8

const (
	TenantStatusPending  TenantStatus = 1 // 待审核
	TenantStatusNormal   TenantStatus = 2 // 正常
	TenantStatusFrozen   TenantStatus = 3 // 冻结
	TenantStatusCanceled TenantStatus = 4 // 注销
)
```

**Step 2: Create plan domain**

```go
// tenant/domain/plan.go
package domain

type TenantPlan struct {
	ID           int64
	Name         string
	Price        int64 // 分
	DurationDays int32
	MaxProducts  int32
	MaxStaff     int32
	Features     string // JSON
	Status       PlanStatus
	Ctime        int64
	Utime        int64
}

type PlanStatus uint8

const (
	PlanStatusEnabled  PlanStatus = 1
	PlanStatusDisabled PlanStatus = 2
)
```

**Step 3: Create shop domain**

```go
// tenant/domain/shop.go
package domain

type Shop struct {
	ID           int64
	TenantID     int64
	Name         string
	Logo         string
	Description  string
	Status       ShopStatus
	Rating       string // decimal as string
	Subdomain    string // shop1 → shop1.mall.com
	CustomDomain string // www.myshop.com
	Ctime        int64
	Utime        int64
}

type ShopStatus uint8

const (
	ShopStatusOpen   ShopStatus = 1 // 营业中
	ShopStatusRest   ShopStatus = 2 // 休息中
	ShopStatusClosed ShopStatus = 3 // 关闭
)

type QuotaUsage struct {
	QuotaType string // product_count / staff_count
	Used      int32
	MaxLimit  int32
}
```

**Step 4: Verify build**

Run: `go build ./tenant/...`
Expected: PASS (domain-only, no external deps)

---

## Task 2: DAO Layer

**Files:**
- Create: `tenant/repository/dao/tenant.go`
- Create: `tenant/repository/dao/init.go`

**Step 1: Create GORM models + 4 DAO interfaces + implementations**

```go
// tenant/repository/dao/tenant.go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ==================== GORM Models ====================

type Tenant struct {
	ID              int64  `gorm:"primaryKey;autoIncrement"`
	Name            string `gorm:"type:varchar(100);not null"`
	ContactName     string `gorm:"type:varchar(50)"`
	ContactPhone    string `gorm:"type:varchar(20)"`
	BusinessLicense string `gorm:"type:varchar(500)"`
	Status          uint8  `gorm:"default:1;not null"` // 1-待审核 2-正常 3-冻结 4-注销
	PlanId          int64
	PlanExpireTime  int64
	Ctime           int64 `gorm:"not null"`
	Utime           int64 `gorm:"not null"`
}

type TenantPlan struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	Name         string `gorm:"type:varchar(50);not null"`
	Price        int64  `gorm:"not null"` // 分
	DurationDays int32  `gorm:"not null"`
	MaxProducts  int32  `gorm:"not null"`
	MaxStaff     int32  `gorm:"not null"`
	Features     string `gorm:"type:text"`
	Status       uint8  `gorm:"default:1;not null"`
	Ctime        int64  `gorm:"not null"`
	Utime        int64  `gorm:"not null"`
}

func (TenantPlan) TableName() string {
	return "tenant_plans"
}

type TenantQuotaUsage struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	TenantId  int64  `gorm:"uniqueIndex:uk_tenant_type;not null"`
	QuotaType string `gorm:"uniqueIndex:uk_tenant_type;type:varchar(30);not null"`
	Used      int32  `gorm:"default:0;not null"`
	MaxLimit  int32  `gorm:"not null"`
	Utime     int64  `gorm:"not null"`
}

func (TenantQuotaUsage) TableName() string {
	return "tenant_quota_usage"
}

type Shop struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	TenantId     int64  `gorm:"uniqueIndex:uk_tenant;not null"`
	Name         string `gorm:"type:varchar(100);not null"`
	Logo         string `gorm:"type:varchar(500)"`
	Description  string `gorm:"type:text"`
	Status       uint8  `gorm:"default:1;not null"`
	Rating       string `gorm:"type:varchar(10);default:'0.0'"`
	Subdomain    string `gorm:"uniqueIndex:uk_subdomain;type:varchar(64)"`
	CustomDomain string `gorm:"uniqueIndex:uk_custom_domain;type:varchar(128)"`
	Ctime        int64  `gorm:"not null"`
	Utime        int64  `gorm:"not null"`
}

// ==================== TenantDAO ====================

type TenantDAO interface {
	Insert(ctx context.Context, t Tenant) (Tenant, error)
	FindById(ctx context.Context, id int64) (Tenant, error)
	Update(ctx context.Context, t Tenant) error
	UpdateStatus(ctx context.Context, id int64, status uint8) error
	List(ctx context.Context, offset, limit int, status uint8) ([]Tenant, int64, error)
}

type GORMTenantDAO struct{ db *gorm.DB }

func NewTenantDAO(db *gorm.DB) TenantDAO {
	return &GORMTenantDAO{db: db}
}

func (d *GORMTenantDAO) Insert(ctx context.Context, t Tenant) (Tenant, error) {
	now := time.Now().UnixMilli()
	t.Ctime = now
	t.Utime = now
	err := d.db.WithContext(ctx).Create(&t).Error
	return t, err
}

func (d *GORMTenantDAO) FindById(ctx context.Context, id int64) (Tenant, error) {
	var t Tenant
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	return t, err
}

func (d *GORMTenantDAO) Update(ctx context.Context, t Tenant) error {
	t.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Updates(&t).Error
}

func (d *GORMTenantDAO) UpdateStatus(ctx context.Context, id int64, status uint8) error {
	return d.db.WithContext(ctx).Model(&Tenant{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": status, "utime": time.Now().UnixMilli()}).Error
}

func (d *GORMTenantDAO) List(ctx context.Context, offset, limit int, status uint8) ([]Tenant, int64, error) {
	db := d.db.WithContext(ctx).Model(&Tenant{})
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tenants []Tenant
	err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&tenants).Error
	return tenants, total, err
}

// ==================== PlanDAO ====================

type PlanDAO interface {
	Insert(ctx context.Context, p TenantPlan) (TenantPlan, error)
	FindById(ctx context.Context, id int64) (TenantPlan, error)
	Update(ctx context.Context, p TenantPlan) error
	ListAll(ctx context.Context) ([]TenantPlan, error)
}

type GORMPlanDAO struct{ db *gorm.DB }

func NewPlanDAO(db *gorm.DB) PlanDAO {
	return &GORMPlanDAO{db: db}
}

func (d *GORMPlanDAO) Insert(ctx context.Context, p TenantPlan) (TenantPlan, error) {
	now := time.Now().UnixMilli()
	p.Ctime = now
	p.Utime = now
	err := d.db.WithContext(ctx).Create(&p).Error
	return p, err
}

func (d *GORMPlanDAO) FindById(ctx context.Context, id int64) (TenantPlan, error) {
	var p TenantPlan
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	return p, err
}

func (d *GORMPlanDAO) Update(ctx context.Context, p TenantPlan) error {
	p.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Updates(&p).Error
}

func (d *GORMPlanDAO) ListAll(ctx context.Context) ([]TenantPlan, error) {
	var plans []TenantPlan
	err := d.db.WithContext(ctx).Order("id ASC").Find(&plans).Error
	return plans, err
}

// ==================== QuotaDAO ====================

type QuotaDAO interface {
	FindByTenantAndType(ctx context.Context, tenantId int64, quotaType string) (TenantQuotaUsage, error)
	Upsert(ctx context.Context, q TenantQuotaUsage) error
	IncrUsed(ctx context.Context, tenantId int64, quotaType string, delta int32) error
	DecrUsed(ctx context.Context, tenantId int64, quotaType string, delta int32) error
}

type GORMQuotaDAO struct{ db *gorm.DB }

func NewQuotaDAO(db *gorm.DB) QuotaDAO {
	return &GORMQuotaDAO{db: db}
}

func (d *GORMQuotaDAO) FindByTenantAndType(ctx context.Context, tenantId int64, quotaType string) (TenantQuotaUsage, error) {
	var q TenantQuotaUsage
	err := d.db.WithContext(ctx).
		Where("tenant_id = ? AND quota_type = ?", tenantId, quotaType).
		First(&q).Error
	return q, err
}

func (d *GORMQuotaDAO) Upsert(ctx context.Context, q TenantQuotaUsage) error {
	q.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Save(&q).Error
}

func (d *GORMQuotaDAO) IncrUsed(ctx context.Context, tenantId int64, quotaType string, delta int32) error {
	return d.db.WithContext(ctx).
		Model(&TenantQuotaUsage{}).
		Where("tenant_id = ? AND quota_type = ?", tenantId, quotaType).
		Updates(map[string]any{
			"used":  gorm.Expr("used + ?", delta),
			"utime": time.Now().UnixMilli(),
		}).Error
}

func (d *GORMQuotaDAO) DecrUsed(ctx context.Context, tenantId int64, quotaType string, delta int32) error {
	return d.db.WithContext(ctx).
		Model(&TenantQuotaUsage{}).
		Where("tenant_id = ? AND quota_type = ? AND used >= ?", tenantId, quotaType, delta).
		Updates(map[string]any{
			"used":  gorm.Expr("used - ?", delta),
			"utime": time.Now().UnixMilli(),
		}).Error
}

// ==================== ShopDAO ====================

type ShopDAO interface {
	Insert(ctx context.Context, s Shop) (Shop, error)
	FindByTenantId(ctx context.Context, tenantId int64) (Shop, error)
	Update(ctx context.Context, s Shop) error
	FindByDomain(ctx context.Context, domain string) (Shop, error)
}

type GORMShopDAO struct{ db *gorm.DB }

func NewShopDAO(db *gorm.DB) ShopDAO {
	return &GORMShopDAO{db: db}
}

func (d *GORMShopDAO) Insert(ctx context.Context, s Shop) (Shop, error) {
	now := time.Now().UnixMilli()
	s.Ctime = now
	s.Utime = now
	err := d.db.WithContext(ctx).Create(&s).Error
	return s, err
}

func (d *GORMShopDAO) FindByTenantId(ctx context.Context, tenantId int64) (Shop, error) {
	var s Shop
	err := d.db.WithContext(ctx).Where("tenant_id = ?", tenantId).First(&s).Error
	return s, err
}

func (d *GORMShopDAO) Update(ctx context.Context, s Shop) error {
	s.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", s.ID, s.TenantId).Updates(&s).Error
}

func (d *GORMShopDAO) FindByDomain(ctx context.Context, domain string) (Shop, error) {
	var s Shop
	err := d.db.WithContext(ctx).
		Where("subdomain = ? OR custom_domain = ?", domain, domain).
		First(&s).Error
	return s, err
}
```

**Step 2: Create init.go**

```go
// tenant/repository/dao/init.go
package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&Tenant{},
		&TenantPlan{},
		&TenantQuotaUsage{},
		&Shop{},
	)
}
```

**Step 3: Verify build**

Run: `go build ./tenant/...`

---

## Task 3: Cache Layer

**Files:**
- Create: `tenant/repository/cache/tenant.go`

**Step 1: Create Redis cache**

```go
// tenant/repository/cache/tenant.go
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

	GetShopByDomain(ctx context.Context, d string) (domain.Shop, error)
	SetShopByDomain(ctx context.Context, d string, s domain.Shop) error

	GetQuota(ctx context.Context, tenantId int64, quotaType string) (domain.QuotaUsage, error)
	SetQuota(ctx context.Context, tenantId int64, q domain.QuotaUsage) error
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
func (c *RedisTenantCache) shopDomainKey(d string) string {
	return fmt.Sprintf("shop:domain:%s", d)
}
func (c *RedisTenantCache) quotaKey(tenantId int64, quotaType string) string {
	return fmt.Sprintf("tenant:quota:%d:%s", tenantId, quotaType)
}

func (c *RedisTenantCache) GetTenant(ctx context.Context, id int64) (domain.Tenant, error) {
	val, err := c.cmd.Get(ctx, c.tenantKey(id)).Result()
	if err != nil { return domain.Tenant{}, err }
	var t domain.Tenant
	err = json.Unmarshal([]byte(val), &t)
	return t, err
}
func (c *RedisTenantCache) SetTenant(ctx context.Context, t domain.Tenant) error {
	data, err := json.Marshal(t)
	if err != nil { return err }
	return c.cmd.Set(ctx, c.tenantKey(t.ID), data, 30*time.Minute).Err()
}
func (c *RedisTenantCache) DeleteTenant(ctx context.Context, id int64) error {
	return c.cmd.Del(ctx, c.tenantKey(id)).Err()
}

func (c *RedisTenantCache) GetShop(ctx context.Context, tenantId int64) (domain.Shop, error) {
	val, err := c.cmd.Get(ctx, c.shopKey(tenantId)).Result()
	if err != nil { return domain.Shop{}, err }
	var s domain.Shop
	err = json.Unmarshal([]byte(val), &s)
	return s, err
}
func (c *RedisTenantCache) SetShop(ctx context.Context, s domain.Shop) error {
	data, err := json.Marshal(s)
	if err != nil { return err }
	return c.cmd.Set(ctx, c.shopKey(s.TenantID), data, 15*time.Minute).Err()
}
func (c *RedisTenantCache) DeleteShop(ctx context.Context, tenantId int64) error {
	return c.cmd.Del(ctx, c.shopKey(tenantId)).Err()
}

func (c *RedisTenantCache) GetShopByDomain(ctx context.Context, d string) (domain.Shop, error) {
	val, err := c.cmd.Get(ctx, c.shopDomainKey(d)).Result()
	if err != nil { return domain.Shop{}, err }
	var s domain.Shop
	err = json.Unmarshal([]byte(val), &s)
	return s, err
}
func (c *RedisTenantCache) SetShopByDomain(ctx context.Context, d string, s domain.Shop) error {
	data, err := json.Marshal(s)
	if err != nil { return err }
	return c.cmd.Set(ctx, c.shopDomainKey(d), data, 15*time.Minute).Err()
}

func (c *RedisTenantCache) GetQuota(ctx context.Context, tenantId int64, quotaType string) (domain.QuotaUsage, error) {
	val, err := c.cmd.Get(ctx, c.quotaKey(tenantId, quotaType)).Result()
	if err != nil { return domain.QuotaUsage{}, err }
	var q domain.QuotaUsage
	err = json.Unmarshal([]byte(val), &q)
	return q, err
}
func (c *RedisTenantCache) SetQuota(ctx context.Context, tenantId int64, q domain.QuotaUsage) error {
	data, err := json.Marshal(q)
	if err != nil { return err }
	return c.cmd.Set(ctx, c.quotaKey(tenantId, q.QuotaType), data, 10*time.Minute).Err()
}
func (c *RedisTenantCache) DeleteQuota(ctx context.Context, tenantId int64, quotaType string) error {
	return c.cmd.Del(ctx, c.quotaKey(tenantId, quotaType)).Err()
}
```

**Step 2: Verify build**

Run: `go build ./tenant/...`

---

## Task 4: Repository Layer

**Files:**
- Create: `tenant/repository/tenant.go`

**Step 1: Create CachedTenantRepository**

The repository wraps 4 DAOs + cache with Cache-Aside pattern for tenant/shop reads and quota checks. Contains entity↔domain conversion. See full code in design doc. Key patterns:

- `GetTenant`: cache → dao → async set cache
- `UpdateTenant`: dao update → delete cache
- `ListTenants`: direct dao (no cache)
- `GetShop` / `GetShopByDomain`: cache → dao → async set cache
- `CheckQuota`: cache → dao → async set cache
- `IncrQuota` / `DecrQuota`: dao update → delete quota cache

Interface methods map 1:1 to the service layer needs — all 17 RPC operations plus the internal helpers.

**Step 2: Verify build**

Run: `go build ./tenant/...`

---

## Task 5: Events Layer

**Files:**
- Create: `tenant/events/types.go`
- Create: `tenant/events/producer.go`

**Step 1: Create event types**

```go
// tenant/events/types.go
package events

type TenantApprovedEvent struct {
	TenantId int64  `json:"tenant_id"`
	Name     string `json:"name"`
	PlanId   int64  `json:"plan_id"`
}

type TenantPlanChangedEvent struct {
	TenantId  int64 `json:"tenant_id"`
	OldPlanId int64 `json:"old_plan_id"`
	NewPlanId int64 `json:"new_plan_id"`
}
```

**Step 2: Create Kafka producer**

```go
// tenant/events/producer.go
package events

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

const (
	TopicTenantApproved   = "tenant_approved"
	TopicTenantPlanChanged = "tenant_plan_changed"
)

type Producer interface {
	ProduceTenantApproved(ctx context.Context, evt TenantApprovedEvent) error
	ProduceTenantPlanChanged(ctx context.Context, evt TenantPlanChangedEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceTenantApproved(ctx context.Context, evt TenantApprovedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil { return err }
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicTenantApproved,
		Value: sarama.ByteEncoder(data),
	})
	return err
}

func (p *SaramaProducer) ProduceTenantPlanChanged(ctx context.Context, evt TenantPlanChangedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil { return err }
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicTenantPlanChanged,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
```

**Step 3: Verify build**

Run: `go build ./tenant/...`

---

## Task 6: Service Layer

**Files:**
- Create: `tenant/service/tenant.go`

**Step 1: Create TenantService**

Interface covers all 17 operations. Key business logic:
- `CreateTenant`: set status = pending, create tenant + create empty shop
- `ApproveTenant(approved=true)`: set status = normal, init quota from plan, async produce `tenant_approved`
- `ApproveTenant(approved=false)`: set status = canceled
- `FreezeTenant(freeze=true)`: set status = frozen; `freeze=false`: set status = normal
- `CheckQuota`: get quota → compare used vs max_limit → return allowed
- `IncrQuota` / `DecrQuota`: update used, invalidate cache

Errors: `ErrTenantNotFound`, `ErrTenantFrozen`, `ErrQuotaExceeded`, `ErrPlanNotFound`, `ErrShopNotFound`

**Step 2: Verify build**

Run: `go build ./tenant/...`

---

## Task 7: gRPC Handler

**Files:**
- Create: `tenant/grpc/tenant.go`

**Step 1: Create TenantGRPCServer**

```go
type TenantGRPCServer struct {
	tenantv1.UnimplementedTenantServiceServer
	svc service.TenantService
}
```

Implements all 17 RPCs. Each RPC:
1. Extract fields from proto request via `req.GetXxx()`
2. Call corresponding service method
3. Convert domain result → proto DTO via helper functions (`toTenantDTO`, `toPlanDTO`, `toShopDTO`, `toQuotaDTO`)
4. Return proto response or gRPC status error

`Register(server *grpc.Server)` calls `tenantv1.RegisterTenantServiceServer(server, s)`.

DTO converters use `timestamppb.New()` for time fields (same as user service).

Error mapping:
- `ErrTenantNotFound` / `ErrShopNotFound` / `ErrPlanNotFound` → `codes.NotFound`
- `ErrTenantFrozen` → `codes.PermissionDenied`
- `ErrQuotaExceeded` → `codes.ResourceExhausted`
- default → `codes.Internal`

**Step 2: Verify build**

Run: `go build ./tenant/...`

---

## Task 8: IoC + Wire + Config + Main

**Files:**
- Create: `tenant/ioc/db.go`
- Create: `tenant/ioc/redis.go`
- Create: `tenant/ioc/kafka.go`
- Create: `tenant/ioc/logger.go`
- Create: `tenant/ioc/grpc.go`
- Create: `tenant/app.go`
- Create: `tenant/wire.go`
- Create: `tenant/config/dev.yaml`
- Create: `tenant/main.go`

**Step 1: Create IoC files**

All IoC files follow user service patterns exactly. Key differences:
- `ioc/db.go`: calls `dao.InitTables(db)` from tenant dao package
- `ioc/grpc.go`: registers `TenantGRPCServer`, service name `"tenant"`
- `ioc/kafka.go`: no consumer group, no consumers — only `InitKafka` → `InitSyncProducer` → `InitProducer`

**Step 2: Create app.go**

```go
// tenant/app.go
package main

import "github.com/rermrf/mall/pkg/grpcx"

type App struct {
	Server *grpcx.Server
}
```

No Consumers field — tenant service is producer-only.

**Step 3: Create wire.go**

```go
//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/rermrf/mall/tenant/grpc"
	"github.com/rermrf/mall/tenant/ioc"
	"github.com/rermrf/mall/tenant/repository"
	"github.com/rermrf/mall/tenant/repository/cache"
	"github.com/rermrf/mall/tenant/repository/dao"
	"github.com/rermrf/mall/tenant/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
)

var tenantSet = wire.NewSet(
	dao.NewTenantDAO,
	dao.NewPlanDAO,
	dao.NewQuotaDAO,
	dao.NewShopDAO,
	cache.NewTenantCache,
	repository.NewTenantRepository,
	service.NewTenantService,
	grpc.NewTenantGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, tenantSet, wire.Struct(new(App), "*"))
	return new(App)
}
```

**Step 4: Create config/dev.yaml**

```yaml
db:
  dsn: "root:wen123...@tcp(rermrf.icu:3306)/mall_tenant?charset=utf8mb4&parseTime=true&loc=Local"

redis:
  addr: "rermrf.icu:6379"
  password: ""
  db: 1

kafka:
  addrs:
    - "rermrf.icu:9094"

grpc:
  port: 8082
  etcdAddrs:
    - "rermrf.icu:2379"
```

Note: Redis DB=1 (user service uses DB=0), gRPC port=8082, database=`mall_tenant`.

**Step 5: Create main.go**

```go
// tenant/main.go
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

**Step 6: Run Wire**

Run: `cd tenant && GOTOOLCHAIN=go1.25.6 go run github.com/google/wire/cmd/wire@latest`
Expected: `wire: github.com/rermrf/mall/tenant: wrote .../wire_gen.go`

**Step 7: Final build + vet**

Run: `go build ./tenant/... && go vet ./tenant/...`
Expected: PASS (clean, no output)

---

## Verification Checklist

1. `go mod tidy` — no errors
2. `go build ./tenant/...` — compiles
3. `go vet ./tenant/...` — clean
4. Wire generation — `wire_gen.go` created with correct dependency graph
5. Start infra → `go run ./tenant/ --config=tenant/config/dev.yaml` — service starts, registers with etcd
