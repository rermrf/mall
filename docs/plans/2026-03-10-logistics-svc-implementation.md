# Logistics Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement logistics-svc microservice (10 gRPC RPCs) + Kafka producer (`order_shipped`) + merchant-bff logistics endpoints (7) + consumer-bff logistics endpoint (1), covering freight templates, freight calculation, and shipment tracking.

**Architecture:** DDD layered architecture (domain → dao → cache → repository → service → grpc → events → ioc → wire) consistent with all other services. Redis caches freight templates for CalculateFreight hot path. Kafka produces `order_shipped` on shipment creation. merchant-bff handles freight template CRUD + ship aggregation (logistics + order status update). consumer-bff exposes shipment tracking query.

**Tech Stack:** Go, gRPC, GORM/MySQL, go-redis, Sarama/Kafka, Wire DI, Gin (BFF), etcd service discovery, Viper config

---

## Reference Files

Before implementing, read these files to understand established patterns:

- **Proto (generated):** `api/proto/gen/logistics/v1/logistics_grpc.pb.go`, `logistics.pb.go`
- **Pattern reference (marketing-svc):** `marketing/domain/marketing.go`, `marketing/repository/dao/marketing.go`, `marketing/repository/cache/marketing.go`, `marketing/repository/marketing.go`, `marketing/service/marketing.go`, `marketing/grpc/marketing.go`, `marketing/events/types.go`, `marketing/events/producer.go`, `marketing/ioc/*.go`, `marketing/wire.go`, `marketing/app.go`, `marketing/main.go`
- **BFF patterns:** `merchant-bff/handler/marketing.go`, `merchant-bff/ioc/grpc.go`, `merchant-bff/ioc/gin.go`, `merchant-bff/wire.go`, `consumer-bff/handler/marketing.go`, `consumer-bff/ioc/grpc.go`, `consumer-bff/ioc/gin.go`, `consumer-bff/wire.go`
- **Existing ship handler:** `merchant-bff/handler/order.go:66-84` (ShipOrder — currently only updates order status, will be replaced by logistics handler's aggregation)

---

## Task 1: Domain Models + DAO + Init

**Files:**
- Create: `logistics/domain/logistics.go`
- Create: `logistics/repository/dao/logistics.go`
- Create: `logistics/repository/dao/init.go`

**Step 1: Create domain models**

Create `logistics/domain/logistics.go`:

```go
package domain

import "time"

// ==================== 运费模板 ====================

type FreightTemplate struct {
	ID            int64
	TenantID      int64
	Name          string
	ChargeType    int32 // 1-按件 2-按重量
	FreeThreshold int64 // 包邮门槛（分），0=不包邮
	Rules         []FreightRule
	Ctime         time.Time
	Utime         time.Time
}

type FreightRule struct {
	ID              int64
	TemplateID      int64
	Regions         string // JSON: 适用地区省编码列表
	FirstUnit       int32  // 首件/首重
	FirstPrice      int64  // 首费（分）
	AdditionalUnit  int32  // 续件/续重
	AdditionalPrice int64  // 续费（分）
}

// ==================== 物流单 ====================

type Shipment struct {
	ID          int64
	TenantID    int64
	OrderID     int64
	CarrierCode string
	CarrierName string
	TrackingNo  string
	Status      int32 // 1-已发货 2-运输中 3-已签收
	Tracks      []ShipmentTrack
	Ctime       time.Time
	Utime       time.Time
}

type ShipmentTrack struct {
	ID          int64
	ShipmentID  int64
	Description string
	Location    string
	TrackTime   int64 // Unix timestamp
}
```

**Step 2: Create DAO models + interfaces + implementations**

Create `logistics/repository/dao/logistics.go`:

```go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ==================== Models ====================

type FreightTemplate struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	TenantID      int64  `gorm:"not null;index:idx_tenant"`
	Name          string `gorm:"type:varchar(100);not null"`
	ChargeType    int32  `gorm:"not null"` // 1-按件 2-按重量
	FreeThreshold int64  `gorm:"not null;default:0"`
	Ctime         int64  `gorm:"not null"`
	Utime         int64  `gorm:"not null"`
}

type FreightRule struct {
	ID              int64  `gorm:"primaryKey;autoIncrement"`
	TemplateID      int64  `gorm:"not null;index:idx_template"`
	Regions         string `gorm:"type:text;not null"` // JSON
	FirstUnit       int32  `gorm:"not null"`
	FirstPrice      int64  `gorm:"not null"`
	AdditionalUnit  int32  `gorm:"not null"`
	AdditionalPrice int64  `gorm:"not null"`
	Ctime           int64  `gorm:"not null"`
	Utime           int64  `gorm:"not null"`
}

type Shipment struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	TenantID    int64  `gorm:"not null;index:idx_tenant_status"`
	OrderID     int64  `gorm:"not null;uniqueIndex:uk_order"`
	CarrierCode string `gorm:"type:varchar(20);not null"`
	CarrierName string `gorm:"type:varchar(50);not null"`
	TrackingNo  string `gorm:"type:varchar(50);not null"`
	Status      int32  `gorm:"not null;default:1;index:idx_tenant_status,priority:2"` // 1-已发货 2-运输中 3-已签收
	Ctime       int64  `gorm:"not null"`
	Utime       int64  `gorm:"not null"`
}

type ShipmentTrack struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	ShipmentID  int64  `gorm:"not null;index:idx_shipment"`
	Description string `gorm:"type:varchar(500);not null"`
	Location    string `gorm:"type:varchar(200)"`
	TrackTime   int64  `gorm:"not null"`
	Ctime       int64  `gorm:"not null"`
}

// ==================== FreightTemplateDAO ====================

type FreightTemplateDAO interface {
	Insert(ctx context.Context, t FreightTemplate, rules []FreightRule) (FreightTemplate, error)
	Update(ctx context.Context, t FreightTemplate, rules []FreightRule) error
	FindById(ctx context.Context, id int64) (FreightTemplate, []FreightRule, error)
	ListByTenant(ctx context.Context, tenantId int64) ([]FreightTemplate, error)
	Delete(ctx context.Context, id, tenantId int64) error
	ListByTenantWithRules(ctx context.Context, tenantId int64) ([]FreightTemplate, []FreightRule, error)
}

type GORMFreightTemplateDAO struct {
	db *gorm.DB
}

func NewFreightTemplateDAO(db *gorm.DB) FreightTemplateDAO {
	return &GORMFreightTemplateDAO{db: db}
}

func (d *GORMFreightTemplateDAO) Insert(ctx context.Context, t FreightTemplate, rules []FreightRule) (FreightTemplate, error) {
	now := time.Now().UnixMilli()
	t.Ctime = now
	t.Utime = now
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&t).Error; err != nil {
			return err
		}
		for i := range rules {
			rules[i].TemplateID = t.ID
			rules[i].Ctime = now
			rules[i].Utime = now
		}
		if len(rules) > 0 {
			return tx.Create(&rules).Error
		}
		return nil
	})
	return t, err
}

func (d *GORMFreightTemplateDAO) Update(ctx context.Context, t FreightTemplate, rules []FreightRule) error {
	now := time.Now().UnixMilli()
	t.Utime = now
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND tenant_id = ?", t.ID, t.TenantID).Updates(&t).Error; err != nil {
			return err
		}
		// 删除旧规则并重建
		if err := tx.Where("template_id = ?", t.ID).Delete(&FreightRule{}).Error; err != nil {
			return err
		}
		for i := range rules {
			rules[i].TemplateID = t.ID
			rules[i].Ctime = now
			rules[i].Utime = now
		}
		if len(rules) > 0 {
			return tx.Create(&rules).Error
		}
		return nil
	})
}

func (d *GORMFreightTemplateDAO) FindById(ctx context.Context, id int64) (FreightTemplate, []FreightRule, error) {
	var t FreightTemplate
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if err != nil {
		return t, nil, err
	}
	var rules []FreightRule
	err = d.db.WithContext(ctx).Where("template_id = ?", id).Find(&rules).Error
	return t, rules, err
}

func (d *GORMFreightTemplateDAO) ListByTenant(ctx context.Context, tenantId int64) ([]FreightTemplate, error) {
	var templates []FreightTemplate
	err := d.db.WithContext(ctx).Where("tenant_id = ?", tenantId).Order("id DESC").Find(&templates).Error
	return templates, err
}

func (d *GORMFreightTemplateDAO) Delete(ctx context.Context, id, tenantId int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND tenant_id = ?", id, tenantId).Delete(&FreightTemplate{}).Error; err != nil {
			return err
		}
		return tx.Where("template_id = ?", id).Delete(&FreightRule{}).Error
	})
}

func (d *GORMFreightTemplateDAO) ListByTenantWithRules(ctx context.Context, tenantId int64) ([]FreightTemplate, []FreightRule, error) {
	var templates []FreightTemplate
	err := d.db.WithContext(ctx).Where("tenant_id = ?", tenantId).Find(&templates).Error
	if err != nil {
		return nil, nil, err
	}
	if len(templates) == 0 {
		return templates, nil, nil
	}
	ids := make([]int64, 0, len(templates))
	for _, t := range templates {
		ids = append(ids, t.ID)
	}
	var rules []FreightRule
	err = d.db.WithContext(ctx).Where("template_id IN ?", ids).Find(&rules).Error
	return templates, rules, err
}

// ==================== ShipmentDAO ====================

type ShipmentDAO interface {
	Insert(ctx context.Context, s Shipment) (Shipment, error)
	FindById(ctx context.Context, id int64) (Shipment, error)
	FindByOrderId(ctx context.Context, orderId int64) (Shipment, error)
	UpdateStatus(ctx context.Context, id int64, status int32) error
}

type GORMShipmentDAO struct {
	db *gorm.DB
}

func NewShipmentDAO(db *gorm.DB) ShipmentDAO {
	return &GORMShipmentDAO{db: db}
}

func (d *GORMShipmentDAO) Insert(ctx context.Context, s Shipment) (Shipment, error) {
	now := time.Now().UnixMilli()
	s.Ctime = now
	s.Utime = now
	err := d.db.WithContext(ctx).Create(&s).Error
	return s, err
}

func (d *GORMShipmentDAO) FindById(ctx context.Context, id int64) (Shipment, error) {
	var s Shipment
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	return s, err
}

func (d *GORMShipmentDAO) FindByOrderId(ctx context.Context, orderId int64) (Shipment, error) {
	var s Shipment
	err := d.db.WithContext(ctx).Where("order_id = ?", orderId).First(&s).Error
	return s, err
}

func (d *GORMShipmentDAO) UpdateStatus(ctx context.Context, id int64, status int32) error {
	return d.db.WithContext(ctx).Model(&Shipment{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "utime": time.Now().UnixMilli()}).Error
}

// ==================== ShipmentTrackDAO ====================

type ShipmentTrackDAO interface {
	Insert(ctx context.Context, t ShipmentTrack) (ShipmentTrack, error)
	ListByShipment(ctx context.Context, shipmentId int64) ([]ShipmentTrack, error)
}

type GORMShipmentTrackDAO struct {
	db *gorm.DB
}

func NewShipmentTrackDAO(db *gorm.DB) ShipmentTrackDAO {
	return &GORMShipmentTrackDAO{db: db}
}

func (d *GORMShipmentTrackDAO) Insert(ctx context.Context, t ShipmentTrack) (ShipmentTrack, error) {
	t.Ctime = time.Now().UnixMilli()
	err := d.db.WithContext(ctx).Create(&t).Error
	return t, err
}

func (d *GORMShipmentTrackDAO) ListByShipment(ctx context.Context, shipmentId int64) ([]ShipmentTrack, error) {
	var tracks []ShipmentTrack
	err := d.db.WithContext(ctx).Where("shipment_id = ?", shipmentId).Order("track_time DESC").Find(&tracks).Error
	return tracks, err
}
```

**Step 3: Create init.go**

Create `logistics/repository/dao/init.go`:

```go
package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&FreightTemplate{},
		&FreightRule{},
		&Shipment{},
		&ShipmentTrack{},
	)
}
```

**Step 4: Verify build**

Run: `go build ./logistics/domain/... && go build ./logistics/repository/dao/...`
Expected: PASS

---

## Task 2: Cache + Repository

**Files:**
- Create: `logistics/repository/cache/logistics.go`
- Create: `logistics/repository/logistics.go`

**Step 1: Create cache layer**

Create `logistics/repository/cache/logistics.go`:

```go
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CachedTemplate struct {
	ID            int64              `json:"id"`
	TenantID      int64              `json:"tenant_id"`
	Name          string             `json:"name"`
	ChargeType    int32              `json:"charge_type"`
	FreeThreshold int64              `json:"free_threshold"`
	Rules         []CachedFreightRule `json:"rules"`
}

type CachedFreightRule struct {
	ID              int64  `json:"id"`
	TemplateID      int64  `json:"template_id"`
	Regions         string `json:"regions"`
	FirstUnit       int32  `json:"first_unit"`
	FirstPrice      int64  `json:"first_price"`
	AdditionalUnit  int32  `json:"additional_unit"`
	AdditionalPrice int64  `json:"additional_price"`
}

type LogisticsCache interface {
	GetTemplates(ctx context.Context, tenantId int64) ([]CachedTemplate, error)
	SetTemplates(ctx context.Context, tenantId int64, templates []CachedTemplate) error
	DeleteTemplates(ctx context.Context, tenantId int64) error
}

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
```

**Step 2: Create repository layer**

Create `logistics/repository/logistics.go`:

```go
package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rermrf/mall/logistics/domain"
	"github.com/rermrf/mall/logistics/repository/cache"
	"github.com/rermrf/mall/logistics/repository/dao"
)

type LogisticsRepository interface {
	// 运费模板
	CreateFreightTemplate(ctx context.Context, t domain.FreightTemplate) (domain.FreightTemplate, error)
	UpdateFreightTemplate(ctx context.Context, t domain.FreightTemplate) error
	GetFreightTemplate(ctx context.Context, id int64) (domain.FreightTemplate, error)
	ListFreightTemplates(ctx context.Context, tenantId int64) ([]domain.FreightTemplate, error)
	DeleteFreightTemplate(ctx context.Context, id, tenantId int64) error
	GetTemplatesForCalculation(ctx context.Context, tenantId int64) ([]cache.CachedTemplate, error)
	// 物流单
	CreateShipment(ctx context.Context, s domain.Shipment) (domain.Shipment, error)
	GetShipment(ctx context.Context, id int64) (domain.Shipment, error)
	GetShipmentByOrder(ctx context.Context, orderId int64) (domain.Shipment, error)
	// 物流轨迹
	AddTrack(ctx context.Context, t domain.ShipmentTrack) (domain.ShipmentTrack, error)
	UpdateShipmentStatus(ctx context.Context, shipmentId int64, status int32) error
}

type logisticsRepository struct {
	templateDAO dao.FreightTemplateDAO
	shipmentDAO dao.ShipmentDAO
	trackDAO    dao.ShipmentTrackDAO
	cache       cache.LogisticsCache
}

func NewLogisticsRepository(
	templateDAO dao.FreightTemplateDAO,
	shipmentDAO dao.ShipmentDAO,
	trackDAO dao.ShipmentTrackDAO,
	c cache.LogisticsCache,
) LogisticsRepository {
	return &logisticsRepository{
		templateDAO: templateDAO,
		shipmentDAO: shipmentDAO,
		trackDAO:    trackDAO,
		cache:       c,
	}
}

// ==================== 运费模板 ====================

func (r *logisticsRepository) CreateFreightTemplate(ctx context.Context, t domain.FreightTemplate) (domain.FreightTemplate, error) {
	rules := r.rulesToDAO(t.Rules)
	dt, err := r.templateDAO.Insert(ctx, r.templateToDAO(t), rules)
	if err != nil {
		return domain.FreightTemplate{}, err
	}
	// 清除缓存
	_ = r.cache.DeleteTemplates(ctx, t.TenantID)
	return r.templateToDomain(dt, rules), nil
}

func (r *logisticsRepository) UpdateFreightTemplate(ctx context.Context, t domain.FreightTemplate) error {
	rules := r.rulesToDAO(t.Rules)
	err := r.templateDAO.Update(ctx, r.templateToDAO(t), rules)
	if err != nil {
		return err
	}
	// 清除缓存
	_ = r.cache.DeleteTemplates(ctx, t.TenantID)
	return nil
}

func (r *logisticsRepository) GetFreightTemplate(ctx context.Context, id int64) (domain.FreightTemplate, error) {
	t, rules, err := r.templateDAO.FindById(ctx, id)
	if err != nil {
		return domain.FreightTemplate{}, err
	}
	return r.templateToDomain(t, rules), nil
}

func (r *logisticsRepository) ListFreightTemplates(ctx context.Context, tenantId int64) ([]domain.FreightTemplate, error) {
	templates, err := r.templateDAO.ListByTenant(ctx, tenantId)
	if err != nil {
		return nil, err
	}
	result := make([]domain.FreightTemplate, 0, len(templates))
	for _, t := range templates {
		result = append(result, r.templateToDomain(t, nil))
	}
	return result, nil
}

func (r *logisticsRepository) DeleteFreightTemplate(ctx context.Context, id, tenantId int64) error {
	err := r.templateDAO.Delete(ctx, id, tenantId)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteTemplates(ctx, tenantId)
	return nil
}

func (r *logisticsRepository) GetTemplatesForCalculation(ctx context.Context, tenantId int64) ([]cache.CachedTemplate, error) {
	// 先查缓存
	templates, err := r.cache.GetTemplates(ctx, tenantId)
	if err == nil && len(templates) > 0 {
		return templates, nil
	}
	// 缓存未命中，查 MySQL
	dbTemplates, dbRules, err := r.templateDAO.ListByTenantWithRules(ctx, tenantId)
	if err != nil {
		return nil, err
	}
	// 组装缓存结构
	ruleMap := make(map[int64][]cache.CachedFreightRule)
	for _, rule := range dbRules {
		ruleMap[rule.TemplateID] = append(ruleMap[rule.TemplateID], cache.CachedFreightRule{
			ID: rule.ID, TemplateID: rule.TemplateID,
			Regions: rule.Regions, FirstUnit: rule.FirstUnit, FirstPrice: rule.FirstPrice,
			AdditionalUnit: rule.AdditionalUnit, AdditionalPrice: rule.AdditionalPrice,
		})
	}
	cached := make([]cache.CachedTemplate, 0, len(dbTemplates))
	for _, t := range dbTemplates {
		cached = append(cached, cache.CachedTemplate{
			ID: t.ID, TenantID: t.TenantID, Name: t.Name,
			ChargeType: t.ChargeType, FreeThreshold: t.FreeThreshold,
			Rules: ruleMap[t.ID],
		})
	}
	// 回填缓存
	_ = r.cache.SetTemplates(ctx, tenantId, cached)
	return cached, nil
}

// ==================== 物流单 ====================

func (r *logisticsRepository) CreateShipment(ctx context.Context, s domain.Shipment) (domain.Shipment, error) {
	ds, err := r.shipmentDAO.Insert(ctx, r.shipmentToDAO(s))
	if err != nil {
		return domain.Shipment{}, err
	}
	return r.shipmentToDomain(ds, nil), nil
}

func (r *logisticsRepository) GetShipment(ctx context.Context, id int64) (domain.Shipment, error) {
	s, err := r.shipmentDAO.FindById(ctx, id)
	if err != nil {
		return domain.Shipment{}, err
	}
	tracks, _ := r.trackDAO.ListByShipment(ctx, id)
	return r.shipmentToDomain(s, tracks), nil
}

func (r *logisticsRepository) GetShipmentByOrder(ctx context.Context, orderId int64) (domain.Shipment, error) {
	s, err := r.shipmentDAO.FindByOrderId(ctx, orderId)
	if err != nil {
		return domain.Shipment{}, err
	}
	tracks, _ := r.trackDAO.ListByShipment(ctx, s.ID)
	return r.shipmentToDomain(s, tracks), nil
}

func (r *logisticsRepository) AddTrack(ctx context.Context, t domain.ShipmentTrack) (domain.ShipmentTrack, error) {
	dt, err := r.trackDAO.Insert(ctx, dao.ShipmentTrack{
		ShipmentID: t.ShipmentID, Description: t.Description,
		Location: t.Location, TrackTime: t.TrackTime,
	})
	if err != nil {
		return domain.ShipmentTrack{}, err
	}
	t.ID = dt.ID
	return t, nil
}

func (r *logisticsRepository) UpdateShipmentStatus(ctx context.Context, shipmentId int64, status int32) error {
	return r.shipmentDAO.UpdateStatus(ctx, shipmentId, status)
}

// ==================== Converters ====================

func (r *logisticsRepository) templateToDAO(t domain.FreightTemplate) dao.FreightTemplate {
	return dao.FreightTemplate{
		ID: t.ID, TenantID: t.TenantID, Name: t.Name,
		ChargeType: t.ChargeType, FreeThreshold: t.FreeThreshold,
	}
}

func (r *logisticsRepository) templateToDomain(t dao.FreightTemplate, rules []dao.FreightRule) domain.FreightTemplate {
	domainRules := make([]domain.FreightRule, 0, len(rules))
	for _, rule := range rules {
		domainRules = append(domainRules, r.ruleToDomain(rule))
	}
	return domain.FreightTemplate{
		ID: t.ID, TenantID: t.TenantID, Name: t.Name,
		ChargeType: t.ChargeType, FreeThreshold: t.FreeThreshold,
		Rules: domainRules,
		Ctime: time.UnixMilli(t.Ctime), Utime: time.UnixMilli(t.Utime),
	}
}

func (r *logisticsRepository) rulesToDAO(rules []domain.FreightRule) []dao.FreightRule {
	result := make([]dao.FreightRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, dao.FreightRule{
			ID: rule.ID, TemplateID: rule.TemplateID, Regions: rule.Regions,
			FirstUnit: rule.FirstUnit, FirstPrice: rule.FirstPrice,
			AdditionalUnit: rule.AdditionalUnit, AdditionalPrice: rule.AdditionalPrice,
		})
	}
	return result
}

func (r *logisticsRepository) ruleToDomain(rule dao.FreightRule) domain.FreightRule {
	return domain.FreightRule{
		ID: rule.ID, TemplateID: rule.TemplateID, Regions: rule.Regions,
		FirstUnit: rule.FirstUnit, FirstPrice: rule.FirstPrice,
		AdditionalUnit: rule.AdditionalUnit, AdditionalPrice: rule.AdditionalPrice,
	}
}

func (r *logisticsRepository) shipmentToDAO(s domain.Shipment) dao.Shipment {
	return dao.Shipment{
		ID: s.ID, TenantID: s.TenantID, OrderID: s.OrderID,
		CarrierCode: s.CarrierCode, CarrierName: s.CarrierName,
		TrackingNo: s.TrackingNo, Status: s.Status,
	}
}

func (r *logisticsRepository) shipmentToDomain(s dao.Shipment, tracks []dao.ShipmentTrack) domain.Shipment {
	domainTracks := make([]domain.ShipmentTrack, 0, len(tracks))
	for _, t := range tracks {
		domainTracks = append(domainTracks, domain.ShipmentTrack{
			ID: t.ID, ShipmentID: t.ShipmentID, Description: t.Description,
			Location: t.Location, TrackTime: t.TrackTime,
		})
	}
	return domain.Shipment{
		ID: s.ID, TenantID: s.TenantID, OrderID: s.OrderID,
		CarrierCode: s.CarrierCode, CarrierName: s.CarrierName,
		TrackingNo: s.TrackingNo, Status: s.Status,
		Tracks: domainTracks,
		Ctime:  time.UnixMilli(s.Ctime), Utime: time.UnixMilli(s.Utime),
	}
}
```

Note: The `encoding/json` import in repository is used by `GetTemplatesForCalculation` indirectly through the cache layer. If the compiler flags it as unused, remove it — the cache package handles JSON marshaling.

**Step 3: Verify build**

Run: `go build ./logistics/repository/...`
Expected: PASS

---

## Task 3: Service + Events (types + producer)

**Files:**
- Create: `logistics/events/types.go`
- Create: `logistics/events/producer.go`
- Create: `logistics/service/logistics.go`

**Step 1: Create event types**

Create `logistics/events/types.go`:

```go
package events

const (
	TopicOrderShipped = "order_shipped"
)

type OrderShippedEvent struct {
	OrderId     int64  `json:"order_id"`
	TenantId    int64  `json:"tenant_id"`
	CarrierCode string `json:"carrier_code"`
	CarrierName string `json:"carrier_name"`
	TrackingNo  string `json:"tracking_no"`
}
```

**Step 2: Create producer**

Create `logistics/events/producer.go`:

```go
package events

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

type Producer interface {
	ProduceOrderShipped(ctx context.Context, evt OrderShippedEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceOrderShipped(ctx context.Context, evt OrderShippedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicOrderShipped,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
```

**Step 3: Create service layer**

Create `logistics/service/logistics.go`:

```go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/logistics/domain"
	"github.com/rermrf/mall/logistics/events"
	"github.com/rermrf/mall/logistics/repository"
	"github.com/rermrf/mall/logistics/repository/cache"
)

var (
	ErrTemplateNotFound = errors.New("运费模板不存在")
	ErrShipmentNotFound = errors.New("物流单不存在")
	ErrNoFreightRule    = errors.New("无匹配运费规则")
)

type LogisticsService interface {
	// 运费模板
	CreateFreightTemplate(ctx context.Context, t domain.FreightTemplate) (domain.FreightTemplate, error)
	UpdateFreightTemplate(ctx context.Context, t domain.FreightTemplate) error
	GetFreightTemplate(ctx context.Context, id int64) (domain.FreightTemplate, error)
	ListFreightTemplates(ctx context.Context, tenantId int64) ([]domain.FreightTemplate, error)
	DeleteFreightTemplate(ctx context.Context, id, tenantId int64) error
	// 运费计算
	CalculateFreight(ctx context.Context, tenantId int64, province string, items []FreightCalcItem) (int64, bool, error)
	// 物流
	CreateShipment(ctx context.Context, s domain.Shipment) (domain.Shipment, error)
	GetShipment(ctx context.Context, id int64) (domain.Shipment, error)
	GetShipmentByOrder(ctx context.Context, orderId int64) (domain.Shipment, error)
	AddTrack(ctx context.Context, shipmentId int64, description, location string, trackTime int64) error
}

type FreightCalcItem struct {
	SkuID    int64
	Quantity int32
	Weight   int32 // 克
}

type logisticsService struct {
	repo     repository.LogisticsRepository
	producer events.Producer
	l        logger.Logger
}

func NewLogisticsService(repo repository.LogisticsRepository, producer events.Producer, l logger.Logger) LogisticsService {
	return &logisticsService{repo: repo, producer: producer, l: l}
}

// ==================== 运费模板 ====================

func (s *logisticsService) CreateFreightTemplate(ctx context.Context, t domain.FreightTemplate) (domain.FreightTemplate, error) {
	return s.repo.CreateFreightTemplate(ctx, t)
}

func (s *logisticsService) UpdateFreightTemplate(ctx context.Context, t domain.FreightTemplate) error {
	return s.repo.UpdateFreightTemplate(ctx, t)
}

func (s *logisticsService) GetFreightTemplate(ctx context.Context, id int64) (domain.FreightTemplate, error) {
	return s.repo.GetFreightTemplate(ctx, id)
}

func (s *logisticsService) ListFreightTemplates(ctx context.Context, tenantId int64) ([]domain.FreightTemplate, error) {
	return s.repo.ListFreightTemplates(ctx, tenantId)
}

func (s *logisticsService) DeleteFreightTemplate(ctx context.Context, id, tenantId int64) error {
	return s.repo.DeleteFreightTemplate(ctx, id, tenantId)
}

// ==================== 运费计算 ====================

func (s *logisticsService) CalculateFreight(ctx context.Context, tenantId int64, province string, items []FreightCalcItem) (int64, bool, error) {
	templates, err := s.repo.GetTemplatesForCalculation(ctx, tenantId)
	if err != nil {
		return 0, false, err
	}
	if len(templates) == 0 {
		return 0, true, nil // 没有运费模板，默认包邮
	}

	// 汇总商品件数和重量
	var totalQty int32
	var totalWeight int32
	var totalAmount int64
	for _, item := range items {
		totalQty += item.Quantity
		totalWeight += item.Weight * item.Quantity
	}

	// 遍历模板找匹配的
	for _, tmpl := range templates {
		// 检查包邮门槛（这里简化处理，实际应该由调用方传入订单金额）
		// CalculateFreight 的包邮门槛判断由 BFF 或 order-svc 编排
		if tmpl.FreeThreshold > 0 && totalAmount >= tmpl.FreeThreshold {
			return 0, true, nil
		}

		// 匹配省份规则
		rule := s.matchRule(tmpl.Rules, province)
		if rule == nil {
			continue
		}

		var freight int64
		switch tmpl.ChargeType {
		case 1: // 按件
			freight = s.calcByPiece(*rule, totalQty)
		case 2: // 按重量
			freight = s.calcByWeight(*rule, totalWeight)
		}
		return freight, false, nil
	}

	return 0, true, nil // 无匹配规则，默认包邮
}

func (s *logisticsService) matchRule(rules []cache.CachedFreightRule, province string) *cache.CachedFreightRule {
	for i, rule := range rules {
		var regions []string
		if err := json.Unmarshal([]byte(rule.Regions), &regions); err != nil {
			continue
		}
		for _, r := range regions {
			if strings.EqualFold(r, province) {
				return &rules[i]
			}
		}
	}
	// 没找到精确匹配，查找默认规则（regions 为空或 ["*"]）
	for i, rule := range rules {
		var regions []string
		if err := json.Unmarshal([]byte(rule.Regions), &regions); err != nil {
			continue
		}
		if len(regions) == 0 {
			return &rules[i]
		}
		for _, r := range regions {
			if r == "*" {
				return &rules[i]
			}
		}
	}
	return nil
}

func (s *logisticsService) calcByPiece(rule cache.CachedFreightRule, qty int32) int64 {
	if qty <= rule.FirstUnit {
		return rule.FirstPrice
	}
	extra := qty - rule.FirstUnit
	additionalUnits := int64(extra) / int64(rule.AdditionalUnit)
	if int64(extra)%int64(rule.AdditionalUnit) > 0 {
		additionalUnits++
	}
	return rule.FirstPrice + additionalUnits*rule.AdditionalPrice
}

func (s *logisticsService) calcByWeight(rule cache.CachedFreightRule, weight int32) int64 {
	if weight <= rule.FirstUnit {
		return rule.FirstPrice
	}
	extra := weight - rule.FirstUnit
	additionalUnits := int64(extra) / int64(rule.AdditionalUnit)
	if int64(extra)%int64(rule.AdditionalUnit) > 0 {
		additionalUnits++
	}
	return rule.FirstPrice + additionalUnits*rule.AdditionalPrice
}

// ==================== 物流 ====================

func (s *logisticsService) CreateShipment(ctx context.Context, ship domain.Shipment) (domain.Shipment, error) {
	ship.Status = 1 // 已发货
	result, err := s.repo.CreateShipment(ctx, ship)
	if err != nil {
		return domain.Shipment{}, err
	}
	// 发送 Kafka 事件
	err = s.producer.ProduceOrderShipped(ctx, events.OrderShippedEvent{
		OrderId:     result.OrderID,
		TenantId:    result.TenantID,
		CarrierCode: result.CarrierCode,
		CarrierName: result.CarrierName,
		TrackingNo:  result.TrackingNo,
	})
	if err != nil {
		s.l.Error("发送发货事件失败", logger.Error(err))
	}
	return result, nil
}

func (s *logisticsService) GetShipment(ctx context.Context, id int64) (domain.Shipment, error) {
	return s.repo.GetShipment(ctx, id)
}

func (s *logisticsService) GetShipmentByOrder(ctx context.Context, orderId int64) (domain.Shipment, error) {
	return s.repo.GetShipmentByOrder(ctx, orderId)
}

func (s *logisticsService) AddTrack(ctx context.Context, shipmentId int64, description, location string, trackTime int64) error {
	_, err := s.repo.AddTrack(ctx, domain.ShipmentTrack{
		ShipmentID:  shipmentId,
		Description: description,
		Location:    location,
		TrackTime:   trackTime,
	})
	return err
}
```

**Step 4: Verify build**

Run: `go build ./logistics/service/... && go build ./logistics/events/...`
Expected: PASS

---

## Task 4: gRPC Handler

**Files:**
- Create: `logistics/grpc/logistics.go`

**Step 1: Create gRPC handler**

Create `logistics/grpc/logistics.go`:

```go
package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	logisticsv1 "github.com/rermrf/mall/api/proto/gen/logistics/v1"
	"github.com/rermrf/mall/logistics/domain"
	"github.com/rermrf/mall/logistics/service"
)

type LogisticsGRPCServer struct {
	logisticsv1.UnimplementedLogisticsServiceServer
	svc service.LogisticsService
}

func NewLogisticsGRPCServer(svc service.LogisticsService) *LogisticsGRPCServer {
	return &LogisticsGRPCServer{svc: svc}
}

func (s *LogisticsGRPCServer) Register(server *grpc.Server) {
	logisticsv1.RegisterLogisticsServiceServer(server, s)
}

// ==================== 运费模板 ====================

func (s *LogisticsGRPCServer) CreateFreightTemplate(ctx context.Context, req *logisticsv1.CreateFreightTemplateRequest) (*logisticsv1.CreateFreightTemplateResponse, error) {
	t := req.GetTemplate()
	rules := make([]domain.FreightRule, 0, len(t.GetRules()))
	for _, r := range t.GetRules() {
		rules = append(rules, domain.FreightRule{
			Regions: r.GetRegions(), FirstUnit: r.GetFirstUnit(), FirstPrice: r.GetFirstPrice(),
			AdditionalUnit: r.GetAdditionalUnit(), AdditionalPrice: r.GetAdditionalPrice(),
		})
	}
	template, err := s.svc.CreateFreightTemplate(ctx, domain.FreightTemplate{
		TenantID: t.GetTenantId(), Name: t.GetName(),
		ChargeType: t.GetChargeType(), FreeThreshold: t.GetFreeThreshold(),
		Rules: rules,
	})
	if err != nil {
		return nil, err
	}
	return &logisticsv1.CreateFreightTemplateResponse{Id: template.ID}, nil
}

func (s *LogisticsGRPCServer) UpdateFreightTemplate(ctx context.Context, req *logisticsv1.UpdateFreightTemplateRequest) (*logisticsv1.UpdateFreightTemplateResponse, error) {
	t := req.GetTemplate()
	rules := make([]domain.FreightRule, 0, len(t.GetRules()))
	for _, r := range t.GetRules() {
		rules = append(rules, domain.FreightRule{
			Regions: r.GetRegions(), FirstUnit: r.GetFirstUnit(), FirstPrice: r.GetFirstPrice(),
			AdditionalUnit: r.GetAdditionalUnit(), AdditionalPrice: r.GetAdditionalPrice(),
		})
	}
	err := s.svc.UpdateFreightTemplate(ctx, domain.FreightTemplate{
		ID: t.GetId(), TenantID: t.GetTenantId(), Name: t.GetName(),
		ChargeType: t.GetChargeType(), FreeThreshold: t.GetFreeThreshold(),
		Rules: rules,
	})
	if err != nil {
		return nil, err
	}
	return &logisticsv1.UpdateFreightTemplateResponse{}, nil
}

func (s *LogisticsGRPCServer) GetFreightTemplate(ctx context.Context, req *logisticsv1.GetFreightTemplateRequest) (*logisticsv1.GetFreightTemplateResponse, error) {
	t, err := s.svc.GetFreightTemplate(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &logisticsv1.GetFreightTemplateResponse{Template: toFreightTemplateDTO(t)}, nil
}

func (s *LogisticsGRPCServer) ListFreightTemplates(ctx context.Context, req *logisticsv1.ListFreightTemplatesRequest) (*logisticsv1.ListFreightTemplatesResponse, error) {
	templates, err := s.svc.ListFreightTemplates(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	pbTemplates := make([]*logisticsv1.FreightTemplate, 0, len(templates))
	for _, t := range templates {
		pbTemplates = append(pbTemplates, toFreightTemplateDTO(t))
	}
	return &logisticsv1.ListFreightTemplatesResponse{Templates: pbTemplates}, nil
}

func (s *LogisticsGRPCServer) DeleteFreightTemplate(ctx context.Context, req *logisticsv1.DeleteFreightTemplateRequest) (*logisticsv1.DeleteFreightTemplateResponse, error) {
	err := s.svc.DeleteFreightTemplate(ctx, req.GetId(), req.GetTenantId())
	if err != nil {
		return nil, err
	}
	return &logisticsv1.DeleteFreightTemplateResponse{}, nil
}

// ==================== 运费计算 ====================

func (s *LogisticsGRPCServer) CalculateFreight(ctx context.Context, req *logisticsv1.CalculateFreightRequest) (*logisticsv1.CalculateFreightResponse, error) {
	items := make([]service.FreightCalcItem, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		items = append(items, service.FreightCalcItem{
			SkuID: item.GetSkuId(), Quantity: item.GetQuantity(), Weight: item.GetWeight(),
		})
	}
	freight, freeShipping, err := s.svc.CalculateFreight(ctx, req.GetTenantId(), req.GetProvince(), items)
	if err != nil {
		return nil, err
	}
	return &logisticsv1.CalculateFreightResponse{Freight: freight, FreeShipping: freeShipping}, nil
}

// ==================== 物流 ====================

func (s *LogisticsGRPCServer) CreateShipment(ctx context.Context, req *logisticsv1.CreateShipmentRequest) (*logisticsv1.CreateShipmentResponse, error) {
	shipment, err := s.svc.CreateShipment(ctx, domain.Shipment{
		TenantID: req.GetTenantId(), OrderID: req.GetOrderId(),
		CarrierCode: req.GetCarrierCode(), CarrierName: req.GetCarrierName(),
		TrackingNo: req.GetTrackingNo(),
	})
	if err != nil {
		return nil, err
	}
	return &logisticsv1.CreateShipmentResponse{Id: shipment.ID}, nil
}

func (s *LogisticsGRPCServer) GetShipment(ctx context.Context, req *logisticsv1.GetShipmentRequest) (*logisticsv1.GetShipmentResponse, error) {
	shipment, err := s.svc.GetShipment(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &logisticsv1.GetShipmentResponse{Shipment: toShipmentDTO(shipment)}, nil
}

func (s *LogisticsGRPCServer) GetShipmentByOrder(ctx context.Context, req *logisticsv1.GetShipmentByOrderRequest) (*logisticsv1.GetShipmentByOrderResponse, error) {
	shipment, err := s.svc.GetShipmentByOrder(ctx, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &logisticsv1.GetShipmentByOrderResponse{Shipment: toShipmentDTO(shipment)}, nil
}

func (s *LogisticsGRPCServer) AddTrack(ctx context.Context, req *logisticsv1.AddTrackRequest) (*logisticsv1.AddTrackResponse, error) {
	err := s.svc.AddTrack(ctx, req.GetShipmentId(), req.GetDescription(), req.GetLocation(), req.GetTrackTime())
	if err != nil {
		return nil, err
	}
	return &logisticsv1.AddTrackResponse{}, nil
}

// ==================== DTO Converters ====================

func toFreightTemplateDTO(t domain.FreightTemplate) *logisticsv1.FreightTemplate {
	rules := make([]*logisticsv1.FreightRule, 0, len(t.Rules))
	for _, r := range t.Rules {
		rules = append(rules, &logisticsv1.FreightRule{
			Id: r.ID, TemplateId: r.TemplateID, Regions: r.Regions,
			FirstUnit: r.FirstUnit, FirstPrice: r.FirstPrice,
			AdditionalUnit: r.AdditionalUnit, AdditionalPrice: r.AdditionalPrice,
		})
	}
	return &logisticsv1.FreightTemplate{
		Id: t.ID, TenantId: t.TenantID, Name: t.Name,
		ChargeType: t.ChargeType, FreeThreshold: t.FreeThreshold,
		Rules: rules,
		Ctime: timestamppb.New(t.Ctime), Utime: timestamppb.New(t.Utime),
	}
}

func toShipmentDTO(s domain.Shipment) *logisticsv1.Shipment {
	tracks := make([]*logisticsv1.ShipmentTrack, 0, len(s.Tracks))
	for _, t := range s.Tracks {
		tracks = append(tracks, &logisticsv1.ShipmentTrack{
			Id: t.ID, ShipmentId: t.ShipmentID, Description: t.Description,
			Location: t.Location, TrackTime: t.TrackTime,
		})
	}
	return &logisticsv1.Shipment{
		Id: s.ID, TenantId: s.TenantID, OrderId: s.OrderID,
		CarrierCode: s.CarrierCode, CarrierName: s.CarrierName,
		TrackingNo: s.TrackingNo, Status: s.Status,
		Tracks: tracks,
		Ctime:  timestamppb.New(s.Ctime), Utime: timestamppb.New(s.Utime),
	}
}
```

**Step 2: Verify build**

Run: `go build ./logistics/grpc/...`
Expected: PASS

---

## Task 5: IoC + Wire + Config + Main

**Files:**
- Create: `logistics/ioc/db.go`
- Create: `logistics/ioc/redis.go`
- Create: `logistics/ioc/logger.go`
- Create: `logistics/ioc/grpc.go`
- Create: `logistics/ioc/kafka.go`
- Create: `logistics/wire.go`
- Create: `logistics/app.go`
- Create: `logistics/main.go`
- Create: `logistics/config/dev.yaml`
- Generate: `logistics/wire_gen.go`

**Step 1: Create IoC files**

Create `logistics/ioc/db.go`:
```go
package ioc

import (
	"fmt"

	"github.com/rermrf/mall/logistics/repository/dao"
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

Create `logistics/ioc/redis.go`:
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

Create `logistics/ioc/logger.go`:
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

Create `logistics/ioc/grpc.go`:
```go
package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	lgrpc "github.com/rermrf/mall/logistics/grpc"
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

func InitGRPCServer(logisticsServer *lgrpc.LogisticsGRPCServer, l logger.Logger) *grpcx.Server {
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
	logisticsServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "logistics",
		L:         l,
	}
}
```

Create `logistics/ioc/kafka.go`:
```go
package ioc

import (
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rermrf/mall/logistics/events"
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
```

**Step 2: Create app.go**

Create `logistics/app.go`:
```go
package main

import "github.com/rermrf/mall/pkg/grpcx"

type App struct {
	Server *grpcx.Server
}
```

Note: No `Consumers` field — logistics is producer-only with no Kafka consumer.

**Step 3: Create wire.go**

Create `logistics/wire.go`:
```go
//go:build wireinject

package main

import (
	"github.com/google/wire"
	lgrpc "github.com/rermrf/mall/logistics/grpc"
	"github.com/rermrf/mall/logistics/ioc"
	"github.com/rermrf/mall/logistics/repository"
	"github.com/rermrf/mall/logistics/repository/cache"
	"github.com/rermrf/mall/logistics/repository/dao"
	"github.com/rermrf/mall/logistics/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
)

var logisticsSet = wire.NewSet(
	dao.NewFreightTemplateDAO,
	dao.NewShipmentDAO,
	dao.NewShipmentTrackDAO,
	cache.NewLogisticsCache,
	repository.NewLogisticsRepository,
	service.NewLogisticsService,
	lgrpc.NewLogisticsGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, logisticsSet, wire.Struct(new(App), "*"))
	return new(App)
}
```

**Step 4: Create main.go**

Create `logistics/main.go`:
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

**Step 5: Create config**

Create `logistics/config/dev.yaml`:
```yaml
db:
  dsn: "root:wen123...@tcp(rermrf.icu:3306)/mall_logistics?charset=utf8mb4&parseTime=true&loc=Local"

redis:
  addr: "rermrf.icu:6379"
  password: ""
  db: 9

kafka:
  addrs:
    - "rermrf.icu:9094"

grpc:
  port: 8090
  etcdAddrs:
    - "rermrf.icu:2379"

etcd:
  addrs:
    - "rermrf.icu:2379"
```

**Step 6: Generate wire_gen.go and verify**

Run: `cd logistics && wire && cd ..`
Then: `go build ./logistics/... && go vet ./logistics/...`
Expected: PASS

---

## Task 6: Merchant BFF — Logistics Handler + Routes

**Files:**
- Create: `merchant-bff/handler/logistics.go`
- Modify: `merchant-bff/ioc/grpc.go` — add `InitLogisticsClient`
- Modify: `merchant-bff/ioc/gin.go` — add `logisticsHandler` param + 7 routes
- Modify: `merchant-bff/wire.go` — add providers
- Regenerate: `merchant-bff/wire_gen.go`

**Step 1: Create logistics handler**

Create `merchant-bff/handler/logistics.go`:

```go
package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	logisticsv1 "github.com/rermrf/mall/api/proto/gen/logistics/v1"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type LogisticsHandler struct {
	logisticsClient logisticsv1.LogisticsServiceClient
	orderClient     orderv1.OrderServiceClient
	l               logger.Logger
}

func NewLogisticsHandler(
	logisticsClient logisticsv1.LogisticsServiceClient,
	orderClient orderv1.OrderServiceClient,
	l logger.Logger,
) *LogisticsHandler {
	return &LogisticsHandler{
		logisticsClient: logisticsClient,
		orderClient:     orderClient,
		l:               l,
	}
}

// ==================== 运费模板 ====================

type CreateFreightTemplateReq struct {
	Name          string             `json:"name" binding:"required"`
	ChargeType    int32              `json:"charge_type" binding:"required"`
	FreeThreshold int64              `json:"free_threshold"`
	Rules         []FreightRuleReq   `json:"rules" binding:"required,min=1"`
}

type FreightRuleReq struct {
	Regions         string `json:"regions" binding:"required"` // JSON
	FirstUnit       int32  `json:"first_unit" binding:"required"`
	FirstPrice      int64  `json:"first_price" binding:"required"`
	AdditionalUnit  int32  `json:"additional_unit" binding:"required"`
	AdditionalPrice int64  `json:"additional_price" binding:"required"`
}

func (h *LogisticsHandler) CreateFreightTemplate(ctx *gin.Context, req CreateFreightTemplateReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	rules := make([]*logisticsv1.FreightRule, 0, len(req.Rules))
	for _, r := range req.Rules {
		rules = append(rules, &logisticsv1.FreightRule{
			Regions: r.Regions, FirstUnit: r.FirstUnit, FirstPrice: r.FirstPrice,
			AdditionalUnit: r.AdditionalUnit, AdditionalPrice: r.AdditionalPrice,
		})
	}
	resp, err := h.logisticsClient.CreateFreightTemplate(ctx.Request.Context(), &logisticsv1.CreateFreightTemplateRequest{
		Template: &logisticsv1.FreightTemplate{
			TenantId: tenantId.(int64), Name: req.Name,
			ChargeType: req.ChargeType, FreeThreshold: req.FreeThreshold,
			Rules: rules,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建运费模板失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateFreightTemplateReq struct {
	Name          string           `json:"name"`
	ChargeType    int32            `json:"charge_type"`
	FreeThreshold int64            `json:"free_threshold"`
	Rules         []FreightRuleReq `json:"rules"`
}

func (h *LogisticsHandler) UpdateFreightTemplate(ctx *gin.Context, req UpdateFreightTemplateReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	rules := make([]*logisticsv1.FreightRule, 0, len(req.Rules))
	for _, r := range req.Rules {
		rules = append(rules, &logisticsv1.FreightRule{
			Regions: r.Regions, FirstUnit: r.FirstUnit, FirstPrice: r.FirstPrice,
			AdditionalUnit: r.AdditionalUnit, AdditionalPrice: r.AdditionalPrice,
		})
	}
	_, err := h.logisticsClient.UpdateFreightTemplate(ctx.Request.Context(), &logisticsv1.UpdateFreightTemplateRequest{
		Template: &logisticsv1.FreightTemplate{
			Id: id, TenantId: tenantId.(int64), Name: req.Name,
			ChargeType: req.ChargeType, FreeThreshold: req.FreeThreshold,
			Rules: rules,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新运费模板失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

func (h *LogisticsHandler) GetFreightTemplate(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resp, err := h.logisticsClient.GetFreightTemplate(ctx.Request.Context(), &logisticsv1.GetFreightTemplateRequest{Id: id})
	if err != nil {
		h.l.Error("查询运费模板详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetTemplate()})
}

func (h *LogisticsHandler) ListFreightTemplates(ctx *gin.Context) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.logisticsClient.ListFreightTemplates(ctx.Request.Context(), &logisticsv1.ListFreightTemplatesRequest{
		TenantId: tenantId.(int64),
	})
	if err != nil {
		h.l.Error("查询运费模板列表失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetTemplates()})
}

func (h *LogisticsHandler) DeleteFreightTemplate(ctx *gin.Context) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.logisticsClient.DeleteFreightTemplate(ctx.Request.Context(), &logisticsv1.DeleteFreightTemplateRequest{
		Id: id, TenantId: tenantId.(int64),
	})
	if err != nil {
		h.l.Error("删除运费模板失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

// ==================== 发货（聚合端点） ====================

type ShipOrderReq struct {
	CarrierCode string `json:"carrier_code" binding:"required"`
	CarrierName string `json:"carrier_name" binding:"required"`
	TrackingNo  string `json:"tracking_no" binding:"required"`
}

func (h *LogisticsHandler) ShipOrder(ctx *gin.Context, req ShipOrderReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	uid, _ := ctx.Get("uid")
	orderNo := ctx.Param("orderNo")

	// 1. 通过 order-svc 获取订单信息（取 order_id）
	orderResp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{OrderNo: orderNo})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询订单失败: %w", err)
	}

	// 2. 创建物流单
	_, err = h.logisticsClient.CreateShipment(ctx.Request.Context(), &logisticsv1.CreateShipmentRequest{
		TenantId:    tenantId.(int64),
		OrderId:     orderResp.GetOrder().GetId(),
		CarrierCode: req.CarrierCode,
		CarrierName: req.CarrierName,
		TrackingNo:  req.TrackingNo,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建物流单失败: %w", err)
	}

	// 3. 更新订单状态为已发货
	_, err = h.orderClient.UpdateOrderStatus(ctx.Request.Context(), &orderv1.UpdateOrderStatusRequest{
		OrderNo:      orderNo,
		Status:       3, // shipped
		OperatorId:   uid.(int64),
		OperatorType: 2, // 商家
		Remark:       "商家发货",
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新订单状态失败: %w", err)
	}

	return ginx.Result{Code: 0, Msg: "success"}, nil
}

// ==================== 查物流 ====================

func (h *LogisticsHandler) GetOrderLogistics(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")

	// 通过 order-svc 获取 order_id
	orderResp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{OrderNo: orderNo})
	if err != nil {
		h.l.Error("查询订单失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}

	resp, err := h.logisticsClient.GetShipmentByOrder(ctx.Request.Context(), &logisticsv1.GetShipmentByOrderRequest{
		OrderId: orderResp.GetOrder().GetId(),
	})
	if err != nil {
		h.l.Error("查询物流信息失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetShipment()})
}
```

**Step 2: Add InitLogisticsClient to merchant-bff/ioc/grpc.go**

Add import `logisticsv1 "github.com/rermrf/mall/api/proto/gen/logistics/v1"` and function:

```go
func InitLogisticsClient(etcdClient *clientv3.Client) logisticsv1.LogisticsServiceClient {
	conn := initServiceConn(etcdClient, "logistics")
	return logisticsv1.NewLogisticsServiceClient(conn)
}
```

**Step 3: Modify merchant-bff/ioc/gin.go**

Add `logisticsHandler *handler.LogisticsHandler` parameter to `InitGinServer`. Replace the existing `auth.POST("/orders/:orderNo/ship", orderHandler.ShipOrder)` with the logistics handler version, and add freight template + logistics routes:

```go
// In InitGinServer signature, add: logisticsHandler *handler.LogisticsHandler

// Replace the existing ship route:
// OLD: auth.POST("/orders/:orderNo/ship", orderHandler.ShipOrder)
// NEW: auth.POST("/orders/:orderNo/ship", ginx.WrapBody[handler.ShipOrderReq](l, logisticsHandler.ShipOrder))

// Add after order routes:
// 物流查询
auth.GET("/orders/:orderNo/logistics", logisticsHandler.GetOrderLogistics)

// 运费模板管理
auth.POST("/freight-templates", ginx.WrapBody[handler.CreateFreightTemplateReq](l, logisticsHandler.CreateFreightTemplate))
auth.PUT("/freight-templates/:id", ginx.WrapBody[handler.UpdateFreightTemplateReq](l, logisticsHandler.UpdateFreightTemplate))
auth.GET("/freight-templates/:id", logisticsHandler.GetFreightTemplate)
auth.GET("/freight-templates", logisticsHandler.ListFreightTemplates)
auth.DELETE("/freight-templates/:id", logisticsHandler.DeleteFreightTemplate)
```

**Step 4: Modify merchant-bff/wire.go**

Add to `thirdPartySet`: `ioc.InitLogisticsClient`
Add to `handlerSet`: `handler.NewLogisticsHandler`

**Step 5: Regenerate wire_gen.go and verify**

Run: `cd merchant-bff && wire && cd ..`
Then: `go build ./merchant-bff/... && go vet ./merchant-bff/...`
Expected: PASS

---

## Task 7: Consumer BFF — Logistics Handler + Routes

**Files:**
- Create: `consumer-bff/handler/logistics.go`
- Modify: `consumer-bff/ioc/grpc.go` — add `InitLogisticsClient`
- Modify: `consumer-bff/ioc/gin.go` — add `logisticsHandler` param + 1 route
- Modify: `consumer-bff/wire.go` — add providers
- Regenerate: `consumer-bff/wire_gen.go`

**Step 1: Create logistics handler**

Create `consumer-bff/handler/logistics.go`:

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	logisticsv1 "github.com/rermrf/mall/api/proto/gen/logistics/v1"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type LogisticsHandler struct {
	logisticsClient logisticsv1.LogisticsServiceClient
	orderClient     orderv1.OrderServiceClient
	l               logger.Logger
}

func NewLogisticsHandler(
	logisticsClient logisticsv1.LogisticsServiceClient,
	orderClient orderv1.OrderServiceClient,
	l logger.Logger,
) *LogisticsHandler {
	return &LogisticsHandler{
		logisticsClient: logisticsClient,
		orderClient:     orderClient,
		l:               l,
	}
}

// GetOrderLogistics 查询订单物流（需登录）
func (h *LogisticsHandler) GetOrderLogistics(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")

	// 通过 order-svc 获取 order_id
	orderResp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{OrderNo: orderNo})
	if err != nil {
		h.l.Error("查询订单失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}

	resp, err := h.logisticsClient.GetShipmentByOrder(ctx.Request.Context(), &logisticsv1.GetShipmentByOrderRequest{
		OrderId: orderResp.GetOrder().GetId(),
	})
	if err != nil {
		h.l.Error("查询物流信息失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetShipment()})
}
```

**Step 2: Add InitLogisticsClient to consumer-bff/ioc/grpc.go**

Add import `logisticsv1 "github.com/rermrf/mall/api/proto/gen/logistics/v1"` and function:

```go
func InitLogisticsClient(etcdClient *clientv3.Client) logisticsv1.LogisticsServiceClient {
	conn := initServiceConn(etcdClient, "logistics")
	return logisticsv1.NewLogisticsServiceClient(conn)
}
```

**Step 3: Modify consumer-bff/ioc/gin.go**

Add `logisticsHandler *handler.LogisticsHandler` parameter to `InitGinServer`. Add route in auth group:

```go
// In InitGinServer signature, add: logisticsHandler *handler.LogisticsHandler

// Add in auth group:
// 物流查询
auth.GET("/orders/:orderNo/logistics", logisticsHandler.GetOrderLogistics)
```

**Step 4: Modify consumer-bff/wire.go**

Add to `thirdPartySet`: `ioc.InitLogisticsClient`
Add to `handlerSet`: `handler.NewLogisticsHandler`

**Step 5: Regenerate wire_gen.go and verify**

Run: `cd consumer-bff && wire && cd ..`
Then: `go build ./consumer-bff/... && go vet ./consumer-bff/...`
Expected: PASS

---

## Task 8: Final Verification

**Step 1: Build all three packages**

Run: `go build ./logistics/... && go vet ./logistics/... && go build ./merchant-bff/... && go vet ./merchant-bff/... && go build ./consumer-bff/... && go vet ./consumer-bff/...`
Expected: ALL PASS

**Summary of changes:**
- 19 new files in `logistics/` (domain, dao, init, cache, repository, service, grpc, events/types, events/producer, 5 ioc, wire, app, main, config)
- 1 generated: `logistics/wire_gen.go`
- 1 new file: `merchant-bff/handler/logistics.go`
- 3 modified in merchant-bff: `ioc/grpc.go`, `ioc/gin.go`, `wire.go`
- 1 regenerated: `merchant-bff/wire_gen.go`
- 1 new file: `consumer-bff/handler/logistics.go`
- 3 modified in consumer-bff: `ioc/grpc.go`, `ioc/gin.go`, `wire.go`
- 1 regenerated: `consumer-bff/wire_gen.go`
- **Total: 29 files** (21 new + 6 modified + 2 generated/regenerated)
