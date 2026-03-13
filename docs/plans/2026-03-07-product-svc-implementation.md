# Product Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement product-svc microservice with 17 gRPC RPCs covering SPU/SKU management, three-level categories, brands, and quota integration with tenant-svc.

**Architecture:** DDD layered architecture (domain → service → repository → dao/cache) following tenant-svc patterns exactly. Product-svc is a gRPC client of tenant-svc for quota checking. Kafka events for product changes. Cache-Aside for product info and category tree.

**Tech Stack:** Go 1.25.6, gRPC, GORM/MySQL, Redis, Kafka/Sarama, Wire DI, Viper config, etcd service discovery

---

## Task 1: Domain 层 — 3 个实体文件

**Files:**
- Create: `product/domain/product.go`
- Create: `product/domain/category.go`
- Create: `product/domain/brand.go`

### product/domain/product.go

SPU + SKU + Spec 实体和状态枚举。

```go
package domain

import "time"

// ==================== Product (SPU) ====================

type Product struct {
	ID          int64
	TenantID    int64
	CategoryID  int64
	BrandID     int64
	Name        string
	Subtitle    string
	MainImage   string
	Images      string // JSON array
	Description string
	Status      ProductStatus
	Sales       int64
	SKUs        []SKU
	Specs       []ProductSpec
	Ctime       time.Time
	Utime       time.Time
}

type ProductStatus uint8

const (
	ProductStatusDraft       ProductStatus = 1
	ProductStatusPublished   ProductStatus = 2
	ProductStatusUnpublished ProductStatus = 3
)

// ==================== SKU ====================

type SKU struct {
	ID            int64
	TenantID      int64
	ProductID     int64
	SpecValues    string // JSON: {"颜色":"红","尺码":"XL"}
	Price         int64  // 分
	OriginalPrice int64
	CostPrice     int64
	SKUCode       string
	BarCode       string
	Status        SKUStatus
	Ctime         time.Time
	Utime         time.Time
}

type SKUStatus uint8

const (
	SKUStatusActive   SKUStatus = 1
	SKUStatusInactive SKUStatus = 2
)

// ==================== ProductSpec ====================

type ProductSpec struct {
	ID        int64
	ProductID int64
	TenantID  int64
	Name      string // e.g. "颜色"
	Values    string // JSON array: ["红色","蓝色"]
}
```

### product/domain/category.go

```go
package domain

type Category struct {
	ID       int64
	TenantID int64
	ParentID int64
	Name     string
	Level    int32
	Sort     int32
	Icon     string
	Status   CategoryStatus
	Children []Category
}

type CategoryStatus uint8

const (
	CategoryStatusActive CategoryStatus = 1
	CategoryStatusHidden CategoryStatus = 2
)
```

### product/domain/brand.go

```go
package domain

type Brand struct {
	ID       int64
	TenantID int64
	Name     string
	Logo     string
	Status   BrandStatus
}

type BrandStatus uint8

const (
	BrandStatusActive   BrandStatus = 1
	BrandStatusInactive BrandStatus = 2
)
```

**验证：** `go build ./product/domain/...`

---

## Task 2: DAO 层 — GORM 模型 + 接口

**Files:**
- Create: `product/repository/dao/product.go`
- Create: `product/repository/dao/category.go`
- Create: `product/repository/dao/brand.go`
- Create: `product/repository/dao/init.go`

### product/repository/dao/product.go

Product (SPU) + SKU + Spec 三个 GORM 模型和对应 DAO。

```go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ==================== Models ====================

type Product struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	TenantID    int64  `gorm:"not null;index:idx_tenant_status"`
	CategoryID  int64  `gorm:"not null;index:idx_tenant_category,priority:2"`
	BrandID     int64  `gorm:"not null"`
	Name        string `gorm:"type:varchar(200);not null"`
	Subtitle    string `gorm:"type:varchar(500)"`
	MainImage   string `gorm:"type:varchar(500)"`
	Images      string `gorm:"type:text"`
	Description string `gorm:"type:text"`
	Status      uint8  `gorm:"default:1;not null;index:idx_tenant_status,priority:2"`
	Sales       int64  `gorm:"default:0;not null"`
	Ctime       int64  `gorm:"not null"`
	Utime       int64  `gorm:"not null"`
}

type ProductSKU struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	TenantID      int64  `gorm:"not null"`
	ProductID     int64  `gorm:"not null;index:idx_product"`
	SpecValues    string `gorm:"type:varchar(500)"`
	Price         int64  `gorm:"not null"`
	OriginalPrice int64  `gorm:"not null"`
	CostPrice     int64  `gorm:"not null"`
	SKUCode       string `gorm:"type:varchar(100);uniqueIndex:uk_sku_code,priority:2"`
	BarCode       string `gorm:"type:varchar(100)"`
	Status        uint8  `gorm:"default:1;not null"`
	Ctime         int64  `gorm:"not null"`
	Utime         int64  `gorm:"not null"`
}

func (ProductSKU) TableName() string {
	return "product_skus"
}

type ProductSpec struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	ProductID int64  `gorm:"not null;index:idx_product"`
	TenantID  int64  `gorm:"not null"`
	Name      string `gorm:"type:varchar(50);not null"`
	Values    string `gorm:"type:varchar(500);not null"`
}

func (ProductSpec) TableName() string {
	return "product_specs"
}

// ==================== ProductDAO ====================

type ProductDAO interface {
	Insert(ctx context.Context, p Product) (Product, error)
	FindById(ctx context.Context, id int64) (Product, error)
	Update(ctx context.Context, p Product) error
	UpdateStatus(ctx context.Context, id, tenantId int64, status uint8) error
	List(ctx context.Context, tenantId, categoryId int64, status uint8, offset, limit int) ([]Product, int64, error)
	BatchFindByIds(ctx context.Context, ids []int64) ([]Product, error)
	Delete(ctx context.Context, id, tenantId int64) error
	IncrSales(ctx context.Context, id, tenantId int64, delta int32) error
	CountByCategory(ctx context.Context, categoryId, tenantId int64) (int64, error)
	CountByBrand(ctx context.Context, brandId, tenantId int64) (int64, error)
}

type GORMProductDAO struct {
	db *gorm.DB
}

func NewProductDAO(db *gorm.DB) ProductDAO {
	return &GORMProductDAO{db: db}
}

func (d *GORMProductDAO) Insert(ctx context.Context, p Product) (Product, error) {
	now := time.Now().UnixMilli()
	p.Ctime = now
	p.Utime = now
	err := d.db.WithContext(ctx).Create(&p).Error
	return p, err
}

func (d *GORMProductDAO) FindById(ctx context.Context, id int64) (Product, error) {
	var p Product
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	return p, err
}

func (d *GORMProductDAO) Update(ctx context.Context, p Product) error {
	p.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", p.ID, p.TenantID).Updates(&p).Error
}

func (d *GORMProductDAO) UpdateStatus(ctx context.Context, id, tenantId int64, status uint8) error {
	return d.db.WithContext(ctx).
		Model(&Product{}).
		Where("id = ? AND tenant_id = ?", id, tenantId).
		Updates(map[string]any{
			"status": status,
			"utime":  time.Now().UnixMilli(),
		}).Error
}

func (d *GORMProductDAO) List(ctx context.Context, tenantId, categoryId int64, status uint8, offset, limit int) ([]Product, int64, error) {
	db := d.db.WithContext(ctx).Model(&Product{}).Where("tenant_id = ?", tenantId)
	if categoryId > 0 {
		db = db.Where("category_id = ?", categoryId)
	}
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	var products []Product
	err = db.Offset(offset).Limit(limit).Order("id DESC").Find(&products).Error
	return products, total, err
}

func (d *GORMProductDAO) BatchFindByIds(ctx context.Context, ids []int64) ([]Product, error) {
	var products []Product
	err := d.db.WithContext(ctx).Where("id IN ?", ids).Find(&products).Error
	return products, err
}

func (d *GORMProductDAO) Delete(ctx context.Context, id, tenantId int64) error {
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantId).Delete(&Product{}).Error
}

func (d *GORMProductDAO) IncrSales(ctx context.Context, id, tenantId int64, delta int32) error {
	return d.db.WithContext(ctx).
		Model(&Product{}).
		Where("id = ? AND tenant_id = ?", id, tenantId).
		Updates(map[string]any{
			"sales": gorm.Expr("sales + ?", delta),
			"utime": time.Now().UnixMilli(),
		}).Error
}

func (d *GORMProductDAO) CountByCategory(ctx context.Context, categoryId, tenantId int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&Product{}).
		Where("category_id = ? AND tenant_id = ?", categoryId, tenantId).Count(&count).Error
	return count, err
}

func (d *GORMProductDAO) CountByBrand(ctx context.Context, brandId, tenantId int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&Product{}).
		Where("brand_id = ? AND tenant_id = ?", brandId, tenantId).Count(&count).Error
	return count, err
}

// ==================== SKUDAO ====================

type SKUDAO interface {
	BatchInsert(ctx context.Context, skus []ProductSKU) error
	FindByProductId(ctx context.Context, productId int64) ([]ProductSKU, error)
	DeleteByProductId(ctx context.Context, productId int64) error
}

type GORMSKUDAO struct {
	db *gorm.DB
}

func NewSKUDAO(db *gorm.DB) SKUDAO {
	return &GORMSKUDAO{db: db}
}

func (d *GORMSKUDAO) BatchInsert(ctx context.Context, skus []ProductSKU) error {
	if len(skus) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	for i := range skus {
		skus[i].Ctime = now
		skus[i].Utime = now
	}
	return d.db.WithContext(ctx).Create(&skus).Error
}

func (d *GORMSKUDAO) FindByProductId(ctx context.Context, productId int64) ([]ProductSKU, error) {
	var skus []ProductSKU
	err := d.db.WithContext(ctx).Where("product_id = ?", productId).Find(&skus).Error
	return skus, err
}

func (d *GORMSKUDAO) DeleteByProductId(ctx context.Context, productId int64) error {
	return d.db.WithContext(ctx).Where("product_id = ?", productId).Delete(&ProductSKU{}).Error
}

// ==================== SpecDAO ====================

type SpecDAO interface {
	BatchInsert(ctx context.Context, specs []ProductSpec) error
	FindByProductId(ctx context.Context, productId int64) ([]ProductSpec, error)
	DeleteByProductId(ctx context.Context, productId int64) error
}

type GORMSpecDAO struct {
	db *gorm.DB
}

func NewSpecDAO(db *gorm.DB) SpecDAO {
	return &GORMSpecDAO{db: db}
}

func (d *GORMSpecDAO) BatchInsert(ctx context.Context, specs []ProductSpec) error {
	if len(specs) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Create(&specs).Error
}

func (d *GORMSpecDAO) FindByProductId(ctx context.Context, productId int64) ([]ProductSpec, error) {
	var specs []ProductSpec
	err := d.db.WithContext(ctx).Where("product_id = ?", productId).Find(&specs).Error
	return specs, err
}

func (d *GORMSpecDAO) DeleteByProductId(ctx context.Context, productId int64) error {
	return d.db.WithContext(ctx).Where("product_id = ?", productId).Delete(&ProductSpec{}).Error
}
```

### product/repository/dao/category.go

```go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type CategoryModel struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	TenantID int64  `gorm:"not null;index:idx_tenant_parent"`
	ParentID int64  `gorm:"not null;index:idx_tenant_parent,priority:2"`
	Name     string `gorm:"type:varchar(100);not null"`
	Level    int32  `gorm:"not null"`
	Sort     int32  `gorm:"default:0"`
	Icon     string `gorm:"type:varchar(500)"`
	Status   uint8  `gorm:"default:1;not null"`
	Ctime    int64  `gorm:"not null"`
	Utime    int64  `gorm:"not null"`
}

func (CategoryModel) TableName() string {
	return "categories"
}

type CategoryDAO interface {
	Insert(ctx context.Context, c CategoryModel) (CategoryModel, error)
	Update(ctx context.Context, c CategoryModel) error
	FindAllByTenant(ctx context.Context, tenantId int64) ([]CategoryModel, error)
	FindById(ctx context.Context, id int64) (CategoryModel, error)
	Delete(ctx context.Context, id, tenantId int64) error
	CountChildren(ctx context.Context, parentId, tenantId int64) (int64, error)
}

type GORMCategoryDAO struct {
	db *gorm.DB
}

func NewCategoryDAO(db *gorm.DB) CategoryDAO {
	return &GORMCategoryDAO{db: db}
}

func (d *GORMCategoryDAO) Insert(ctx context.Context, c CategoryModel) (CategoryModel, error) {
	now := time.Now().UnixMilli()
	c.Ctime = now
	c.Utime = now
	err := d.db.WithContext(ctx).Create(&c).Error
	return c, err
}

func (d *GORMCategoryDAO) Update(ctx context.Context, c CategoryModel) error {
	c.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", c.ID, c.TenantID).Updates(&c).Error
}

func (d *GORMCategoryDAO) FindAllByTenant(ctx context.Context, tenantId int64) ([]CategoryModel, error) {
	var categories []CategoryModel
	err := d.db.WithContext(ctx).
		Where("tenant_id = ?", tenantId).
		Order("level ASC, sort ASC, id ASC").
		Find(&categories).Error
	return categories, err
}

func (d *GORMCategoryDAO) FindById(ctx context.Context, id int64) (CategoryModel, error) {
	var c CategoryModel
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	return c, err
}

func (d *GORMCategoryDAO) Delete(ctx context.Context, id, tenantId int64) error {
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantId).Delete(&CategoryModel{}).Error
}

func (d *GORMCategoryDAO) CountChildren(ctx context.Context, parentId, tenantId int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&CategoryModel{}).
		Where("parent_id = ? AND tenant_id = ?", parentId, tenantId).Count(&count).Error
	return count, err
}
```

### product/repository/dao/brand.go

```go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type BrandModel struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	TenantID int64  `gorm:"not null;index:idx_tenant;uniqueIndex:uk_tenant_name"`
	Name     string `gorm:"type:varchar(100);not null;uniqueIndex:uk_tenant_name,priority:2"`
	Logo     string `gorm:"type:varchar(500)"`
	Status   uint8  `gorm:"default:1;not null"`
	Ctime    int64  `gorm:"not null"`
	Utime    int64  `gorm:"not null"`
}

func (BrandModel) TableName() string {
	return "brands"
}

type BrandDAO interface {
	Insert(ctx context.Context, b BrandModel) (BrandModel, error)
	Update(ctx context.Context, b BrandModel) error
	List(ctx context.Context, tenantId int64, offset, limit int) ([]BrandModel, int64, error)
	FindById(ctx context.Context, id int64) (BrandModel, error)
	Delete(ctx context.Context, id, tenantId int64) error
}

type GORMBrandDAO struct {
	db *gorm.DB
}

func NewBrandDAO(db *gorm.DB) BrandDAO {
	return &GORMBrandDAO{db: db}
}

func (d *GORMBrandDAO) Insert(ctx context.Context, b BrandModel) (BrandModel, error) {
	now := time.Now().UnixMilli()
	b.Ctime = now
	b.Utime = now
	err := d.db.WithContext(ctx).Create(&b).Error
	return b, err
}

func (d *GORMBrandDAO) Update(ctx context.Context, b BrandModel) error {
	b.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", b.ID, b.TenantID).Updates(&b).Error
}

func (d *GORMBrandDAO) List(ctx context.Context, tenantId int64, offset, limit int) ([]BrandModel, int64, error) {
	db := d.db.WithContext(ctx).Model(&BrandModel{}).Where("tenant_id = ?", tenantId)
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	var brands []BrandModel
	err = db.Offset(offset).Limit(limit).Order("id DESC").Find(&brands).Error
	return brands, total, err
}

func (d *GORMBrandDAO) FindById(ctx context.Context, id int64) (BrandModel, error) {
	var b BrandModel
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&b).Error
	return b, err
}

func (d *GORMBrandDAO) Delete(ctx context.Context, id, tenantId int64) error {
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantId).Delete(&BrandModel{}).Error
}
```

### product/repository/dao/init.go

```go
package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&Product{},
		&ProductSKU{},
		&ProductSpec{},
		&CategoryModel{},
		&BrandModel{},
	)
}
```

**验证：** `go build ./product/repository/dao/...`

---

## Task 3: Cache 层 + Repository 层

**Files:**
- Create: `product/repository/cache/product.go`
- Create: `product/repository/product.go`
- Create: `product/repository/category.go`
- Create: `product/repository/brand.go`

### product/repository/cache/product.go

缓存 product info 和 category tree。

```go
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
```

### product/repository/product.go

Product 相关 Repository（含 SKU 和 Spec 的事务操作）。

```go
package repository

import (
	"context"
	"time"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/product/domain"
	"github.com/rermrf/mall/product/repository/cache"
	"github.com/rermrf/mall/product/repository/dao"
	"gorm.io/gorm"
)

type ProductRepository interface {
	CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error)
	GetProduct(ctx context.Context, id int64) (domain.Product, error)
	UpdateProduct(ctx context.Context, p domain.Product) error
	UpdateProductStatus(ctx context.Context, id, tenantId int64, status domain.ProductStatus) error
	ListProducts(ctx context.Context, tenantId, categoryId int64, status uint8, offset, limit int) ([]domain.Product, int64, error)
	BatchGetProducts(ctx context.Context, ids []int64) ([]domain.Product, error)
	DeleteProduct(ctx context.Context, id, tenantId int64) error
	IncrSales(ctx context.Context, id, tenantId int64, delta int32) error
	CountByCategory(ctx context.Context, categoryId, tenantId int64) (int64, error)
	CountByBrand(ctx context.Context, brandId, tenantId int64) (int64, error)
}

type CachedProductRepository struct {
	productDAO dao.ProductDAO
	skuDAO     dao.SKUDAO
	specDAO    dao.SpecDAO
	cache      cache.ProductCache
	db         *gorm.DB
	l          logger.Logger
}

func NewProductRepository(
	productDAO dao.ProductDAO,
	skuDAO dao.SKUDAO,
	specDAO dao.SpecDAO,
	cache cache.ProductCache,
	db *gorm.DB,
	l logger.Logger,
) ProductRepository {
	return &CachedProductRepository{
		productDAO: productDAO,
		skuDAO:     skuDAO,
		specDAO:    specDAO,
		cache:      cache,
		db:         db,
		l:          l,
	}
}

func (r *CachedProductRepository) CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	// 事务：创建 SPU + SKUs + Specs
	var result domain.Product
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		productEntity := r.productToEntity(p)
		created, err := dao.NewProductDAO(tx).Insert(ctx, productEntity)
		if err != nil {
			return err
		}

		// 插入 SKUs
		if len(p.SKUs) > 0 {
			skuEntities := make([]dao.ProductSKU, 0, len(p.SKUs))
			for _, sku := range p.SKUs {
				e := r.skuToEntity(sku)
				e.ProductID = created.ID
				e.TenantID = p.TenantID
				skuEntities = append(skuEntities, e)
			}
			if err := dao.NewSKUDAO(tx).BatchInsert(ctx, skuEntities); err != nil {
				return err
			}
		}

		// 插入 Specs
		if len(p.Specs) > 0 {
			specEntities := make([]dao.ProductSpec, 0, len(p.Specs))
			for _, spec := range p.Specs {
				e := r.specToEntity(spec)
				e.ProductID = created.ID
				e.TenantID = p.TenantID
				specEntities = append(specEntities, e)
			}
			if err := dao.NewSpecDAO(tx).BatchInsert(ctx, specEntities); err != nil {
				return err
			}
		}

		result = r.productToDomain(created)
		return nil
	})
	return result, err
}

func (r *CachedProductRepository) GetProduct(ctx context.Context, id int64) (domain.Product, error) {
	// Cache-Aside
	p, err := r.cache.GetProduct(ctx, id)
	if err == nil {
		return p, nil
	}

	entity, err := r.productDAO.FindById(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}
	p = r.productToDomain(entity)

	// 加载 SKUs
	skuEntities, err := r.skuDAO.FindByProductId(ctx, id)
	if err == nil {
		p.SKUs = make([]domain.SKU, 0, len(skuEntities))
		for _, e := range skuEntities {
			p.SKUs = append(p.SKUs, r.skuToDomain(e))
		}
	}

	// 加载 Specs
	specEntities, err := r.specDAO.FindByProductId(ctx, id)
	if err == nil {
		p.Specs = make([]domain.ProductSpec, 0, len(specEntities))
		for _, e := range specEntities {
			p.Specs = append(p.Specs, r.specToDomain(e))
		}
	}

	// async set cache
	go func() {
		if er := r.cache.SetProduct(context.Background(), p); er != nil {
			r.l.Error("设置商品缓存失败", logger.Error(er), logger.Int64("pid", id))
		}
	}()
	return p, nil
}

func (r *CachedProductRepository) UpdateProduct(ctx context.Context, p domain.Product) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新 SPU
		if err := dao.NewProductDAO(tx).Update(ctx, r.productToEntity(p)); err != nil {
			return err
		}
		// 删后插 SKUs
		skuDAO := dao.NewSKUDAO(tx)
		if err := skuDAO.DeleteByProductId(ctx, p.ID); err != nil {
			return err
		}
		if len(p.SKUs) > 0 {
			skuEntities := make([]dao.ProductSKU, 0, len(p.SKUs))
			for _, sku := range p.SKUs {
				e := r.skuToEntity(sku)
				e.ProductID = p.ID
				e.TenantID = p.TenantID
				skuEntities = append(skuEntities, e)
			}
			if err := skuDAO.BatchInsert(ctx, skuEntities); err != nil {
				return err
			}
		}
		// 删后插 Specs
		specDAO := dao.NewSpecDAO(tx)
		if err := specDAO.DeleteByProductId(ctx, p.ID); err != nil {
			return err
		}
		if len(p.Specs) > 0 {
			specEntities := make([]dao.ProductSpec, 0, len(p.Specs))
			for _, spec := range p.Specs {
				e := r.specToEntity(spec)
				e.ProductID = p.ID
				e.TenantID = p.TenantID
				specEntities = append(specEntities, e)
			}
			if err := specDAO.BatchInsert(ctx, specEntities); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	_ = r.cache.DeleteProduct(ctx, p.ID)
	return nil
}

func (r *CachedProductRepository) UpdateProductStatus(ctx context.Context, id, tenantId int64, status domain.ProductStatus) error {
	err := r.productDAO.UpdateStatus(ctx, id, tenantId, uint8(status))
	if err != nil {
		return err
	}
	_ = r.cache.DeleteProduct(ctx, id)
	return nil
}

func (r *CachedProductRepository) ListProducts(ctx context.Context, tenantId, categoryId int64, status uint8, offset, limit int) ([]domain.Product, int64, error) {
	entities, total, err := r.productDAO.List(ctx, tenantId, categoryId, status, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	products := make([]domain.Product, 0, len(entities))
	for _, e := range entities {
		products = append(products, r.productToDomain(e))
	}
	return products, total, nil
}

func (r *CachedProductRepository) BatchGetProducts(ctx context.Context, ids []int64) ([]domain.Product, error) {
	entities, err := r.productDAO.BatchFindByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	products := make([]domain.Product, 0, len(entities))
	for _, e := range entities {
		p := r.productToDomain(e)
		// 加载 SKUs
		skuEntities, er := r.skuDAO.FindByProductId(ctx, e.ID)
		if er == nil {
			p.SKUs = make([]domain.SKU, 0, len(skuEntities))
			for _, se := range skuEntities {
				p.SKUs = append(p.SKUs, r.skuToDomain(se))
			}
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *CachedProductRepository) DeleteProduct(ctx context.Context, id, tenantId int64) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dao.NewSKUDAO(tx).DeleteByProductId(ctx, id); err != nil {
			return err
		}
		if err := dao.NewSpecDAO(tx).DeleteByProductId(ctx, id); err != nil {
			return err
		}
		return dao.NewProductDAO(tx).Delete(ctx, id, tenantId)
	})
	if err != nil {
		return err
	}
	_ = r.cache.DeleteProduct(ctx, id)
	return nil
}

func (r *CachedProductRepository) IncrSales(ctx context.Context, id, tenantId int64, delta int32) error {
	err := r.productDAO.IncrSales(ctx, id, tenantId, delta)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteProduct(ctx, id)
	return nil
}

func (r *CachedProductRepository) CountByCategory(ctx context.Context, categoryId, tenantId int64) (int64, error) {
	return r.productDAO.CountByCategory(ctx, categoryId, tenantId)
}

func (r *CachedProductRepository) CountByBrand(ctx context.Context, brandId, tenantId int64) (int64, error) {
	return r.productDAO.CountByBrand(ctx, brandId, tenantId)
}

// ==================== Converters ====================

func (r *CachedProductRepository) productToEntity(p domain.Product) dao.Product {
	return dao.Product{
		ID: p.ID, TenantID: p.TenantID, CategoryID: p.CategoryID, BrandID: p.BrandID,
		Name: p.Name, Subtitle: p.Subtitle, MainImage: p.MainImage, Images: p.Images,
		Description: p.Description, Status: uint8(p.Status), Sales: p.Sales,
	}
}

func (r *CachedProductRepository) productToDomain(e dao.Product) domain.Product {
	return domain.Product{
		ID: e.ID, TenantID: e.TenantID, CategoryID: e.CategoryID, BrandID: e.BrandID,
		Name: e.Name, Subtitle: e.Subtitle, MainImage: e.MainImage, Images: e.Images,
		Description: e.Description, Status: domain.ProductStatus(e.Status), Sales: e.Sales,
		Ctime: time.UnixMilli(e.Ctime), Utime: time.UnixMilli(e.Utime),
	}
}

func (r *CachedProductRepository) skuToEntity(s domain.SKU) dao.ProductSKU {
	return dao.ProductSKU{
		ID: s.ID, TenantID: s.TenantID, ProductID: s.ProductID,
		SpecValues: s.SpecValues, Price: s.Price, OriginalPrice: s.OriginalPrice,
		CostPrice: s.CostPrice, SKUCode: s.SKUCode, BarCode: s.BarCode, Status: uint8(s.Status),
	}
}

func (r *CachedProductRepository) skuToDomain(e dao.ProductSKU) domain.SKU {
	return domain.SKU{
		ID: e.ID, TenantID: e.TenantID, ProductID: e.ProductID,
		SpecValues: e.SpecValues, Price: e.Price, OriginalPrice: e.OriginalPrice,
		CostPrice: e.CostPrice, SKUCode: e.SKUCode, BarCode: e.BarCode, Status: domain.SKUStatus(e.Status),
		Ctime: time.UnixMilli(e.Ctime), Utime: time.UnixMilli(e.Utime),
	}
}

func (r *CachedProductRepository) specToEntity(s domain.ProductSpec) dao.ProductSpec {
	return dao.ProductSpec{
		ID: s.ID, ProductID: s.ProductID, TenantID: s.TenantID, Name: s.Name, Values: s.Values,
	}
}

func (r *CachedProductRepository) specToDomain(e dao.ProductSpec) domain.ProductSpec {
	return domain.ProductSpec{
		ID: e.ID, ProductID: e.ProductID, TenantID: e.TenantID, Name: e.Name, Values: e.Values,
	}
}
```

### product/repository/category.go

```go
package repository

import (
	"context"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/product/domain"
	"github.com/rermrf/mall/product/repository/cache"
	"github.com/rermrf/mall/product/repository/dao"
)

type CategoryRepository interface {
	CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error)
	UpdateCategory(ctx context.Context, c domain.Category) error
	ListCategories(ctx context.Context, tenantId int64) ([]domain.Category, error)
	DeleteCategory(ctx context.Context, id, tenantId int64) error
	CountChildren(ctx context.Context, parentId, tenantId int64) (int64, error)
}

type CachedCategoryRepository struct {
	categoryDAO dao.CategoryDAO
	cache       cache.ProductCache
	l           logger.Logger
}

func NewCategoryRepository(categoryDAO dao.CategoryDAO, cache cache.ProductCache, l logger.Logger) CategoryRepository {
	return &CachedCategoryRepository{categoryDAO: categoryDAO, cache: cache, l: l}
}

func (r *CachedCategoryRepository) CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error) {
	entity := r.toEntity(c)
	res, err := r.categoryDAO.Insert(ctx, entity)
	if err != nil {
		return domain.Category{}, err
	}
	_ = r.cache.DeleteCategoryTree(ctx, c.TenantID)
	return r.toDomain(res), nil
}

func (r *CachedCategoryRepository) UpdateCategory(ctx context.Context, c domain.Category) error {
	err := r.categoryDAO.Update(ctx, r.toEntity(c))
	if err != nil {
		return err
	}
	_ = r.cache.DeleteCategoryTree(ctx, c.TenantID)
	return nil
}

func (r *CachedCategoryRepository) ListCategories(ctx context.Context, tenantId int64) ([]domain.Category, error) {
	// Cache-Aside: try cache first
	tree, err := r.cache.GetCategoryTree(ctx, tenantId)
	if err == nil {
		return tree, nil
	}
	entities, err := r.categoryDAO.FindAllByTenant(ctx, tenantId)
	if err != nil {
		return nil, err
	}
	// 邻接表 → 内存建树
	all := make([]domain.Category, 0, len(entities))
	for _, e := range entities {
		all = append(all, r.toDomain(e))
	}
	tree = buildTree(all, 0)
	// async set cache
	go func() {
		if er := r.cache.SetCategoryTree(context.Background(), tenantId, tree); er != nil {
			r.l.Error("设置分类树缓存失败", logger.Error(er), logger.Int64("tid", tenantId))
		}
	}()
	return tree, nil
}

func (r *CachedCategoryRepository) DeleteCategory(ctx context.Context, id, tenantId int64) error {
	err := r.categoryDAO.Delete(ctx, id, tenantId)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteCategoryTree(ctx, tenantId)
	return nil
}

func (r *CachedCategoryRepository) CountChildren(ctx context.Context, parentId, tenantId int64) (int64, error) {
	return r.categoryDAO.CountChildren(ctx, parentId, tenantId)
}

// buildTree 从扁平列表构建树形结构
func buildTree(categories []domain.Category, parentId int64) []domain.Category {
	var tree []domain.Category
	for _, c := range categories {
		if c.ParentID == parentId {
			c.Children = buildTree(categories, c.ID)
			tree = append(tree, c)
		}
	}
	return tree
}

func (r *CachedCategoryRepository) toEntity(c domain.Category) dao.CategoryModel {
	return dao.CategoryModel{
		ID: c.ID, TenantID: c.TenantID, ParentID: c.ParentID,
		Name: c.Name, Level: c.Level, Sort: c.Sort, Icon: c.Icon, Status: uint8(c.Status),
	}
}

func (r *CachedCategoryRepository) toDomain(e dao.CategoryModel) domain.Category {
	return domain.Category{
		ID: e.ID, TenantID: e.TenantID, ParentID: e.ParentID,
		Name: e.Name, Level: e.Level, Sort: e.Sort, Icon: e.Icon, Status: domain.CategoryStatus(e.Status),
	}
}
```

### product/repository/brand.go

```go
package repository

import (
	"context"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/product/domain"
	"github.com/rermrf/mall/product/repository/dao"
)

type BrandRepository interface {
	CreateBrand(ctx context.Context, b domain.Brand) (domain.Brand, error)
	UpdateBrand(ctx context.Context, b domain.Brand) error
	ListBrands(ctx context.Context, tenantId int64, offset, limit int) ([]domain.Brand, int64, error)
	DeleteBrand(ctx context.Context, id, tenantId int64) error
}

type CachedBrandRepository struct {
	brandDAO dao.BrandDAO
	l        logger.Logger
}

func NewBrandRepository(brandDAO dao.BrandDAO, l logger.Logger) BrandRepository {
	return &CachedBrandRepository{brandDAO: brandDAO, l: l}
}

func (r *CachedBrandRepository) CreateBrand(ctx context.Context, b domain.Brand) (domain.Brand, error) {
	entity := r.toEntity(b)
	res, err := r.brandDAO.Insert(ctx, entity)
	if err != nil {
		return domain.Brand{}, err
	}
	return r.toDomain(res), nil
}

func (r *CachedBrandRepository) UpdateBrand(ctx context.Context, b domain.Brand) error {
	return r.brandDAO.Update(ctx, r.toEntity(b))
}

func (r *CachedBrandRepository) ListBrands(ctx context.Context, tenantId int64, offset, limit int) ([]domain.Brand, int64, error) {
	entities, total, err := r.brandDAO.List(ctx, tenantId, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	brands := make([]domain.Brand, 0, len(entities))
	for _, e := range entities {
		brands = append(brands, r.toDomain(e))
	}
	return brands, total, nil
}

func (r *CachedBrandRepository) DeleteBrand(ctx context.Context, id, tenantId int64) error {
	return r.brandDAO.Delete(ctx, id, tenantId)
}

func (r *CachedBrandRepository) toEntity(b domain.Brand) dao.BrandModel {
	return dao.BrandModel{
		ID: b.ID, TenantID: b.TenantID, Name: b.Name, Logo: b.Logo, Status: uint8(b.Status),
	}
}

func (r *CachedBrandRepository) toDomain(e dao.BrandModel) domain.Brand {
	return domain.Brand{
		ID: e.ID, TenantID: e.TenantID, Name: e.Name, Logo: e.Logo, Status: domain.BrandStatus(e.Status),
	}
}
```

**验证：** `go build ./product/repository/...`

---

## Task 4: Events 层 + Service 层

**Files:**
- Create: `product/events/types.go`
- Create: `product/events/producer.go`
- Create: `product/service/product.go`

### product/events/types.go

```go
package events

type ProductStatusChangedEvent struct {
	ProductId int64 `json:"product_id"`
	TenantId  int64 `json:"tenant_id"`
	OldStatus int32 `json:"old_status"`
	NewStatus int32 `json:"new_status"`
}

type ProductUpdatedEvent struct {
	ProductId int64 `json:"product_id"`
	TenantId  int64 `json:"tenant_id"`
}
```

### product/events/producer.go

```go
package events

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

const (
	TopicProductStatusChanged = "product_status_changed"
	TopicProductUpdated       = "product_updated"
)

type Producer interface {
	ProduceProductStatusChanged(ctx context.Context, evt ProductStatusChangedEvent) error
	ProduceProductUpdated(ctx context.Context, evt ProductUpdatedEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceProductStatusChanged(ctx context.Context, evt ProductStatusChangedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicProductStatusChanged,
		Value: sarama.ByteEncoder(data),
	})
	return err
}

func (p *SaramaProducer) ProduceProductUpdated(ctx context.Context, evt ProductUpdatedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicProductUpdated,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
```

### product/service/product.go

业务逻辑，单文件单接口。包含配额检查（调 tenant-svc）。

```go
package service

import (
	"context"
	"errors"

	"github.com/rermrf/emo/logger"
	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/product/domain"
	"github.com/rermrf/mall/product/events"
	"github.com/rermrf/mall/product/repository"
)

var (
	ErrProductNotFound    = errors.New("商品不存在")
	ErrCategoryNotFound   = errors.New("分类不存在")
	ErrBrandNotFound      = errors.New("品牌不存在")
	ErrQuotaExceeded      = errors.New("商品配额已超限")
	ErrCategoryHasChild   = errors.New("该分类下有子分类，不能删除")
	ErrCategoryHasProduct = errors.New("该分类下有商品，不能删除")
	ErrBrandHasProduct    = errors.New("该品牌下有商品，不能删除")
	ErrCategoryLevelLimit = errors.New("分类层级不能超过3级")
)

type ProductService interface {
	// Product
	CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error)
	GetProduct(ctx context.Context, id int64) (domain.Product, error)
	UpdateProduct(ctx context.Context, p domain.Product) error
	UpdateProductStatus(ctx context.Context, id, tenantId int64, status int32) error
	ListProducts(ctx context.Context, tenantId, categoryId int64, status int32, page, pageSize int32) ([]domain.Product, int64, error)
	BatchGetProducts(ctx context.Context, ids []int64) ([]domain.Product, error)
	DeleteProduct(ctx context.Context, id, tenantId int64) error

	// Category
	CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error)
	UpdateCategory(ctx context.Context, c domain.Category) error
	ListCategories(ctx context.Context, tenantId int64) ([]domain.Category, error)
	DeleteCategory(ctx context.Context, id, tenantId int64) error

	// Brand
	CreateBrand(ctx context.Context, b domain.Brand) (domain.Brand, error)
	UpdateBrand(ctx context.Context, b domain.Brand) error
	ListBrands(ctx context.Context, tenantId int64, page, pageSize int32) ([]domain.Brand, int64, error)
	DeleteBrand(ctx context.Context, id, tenantId int64) error

	// Sales
	IncrSales(ctx context.Context, productId, tenantId int64, count int32) error
}

type productService struct {
	productRepo  repository.ProductRepository
	categoryRepo repository.CategoryRepository
	brandRepo    repository.BrandRepository
	tenantClient tenantv1.TenantServiceClient
	producer     events.Producer
	l            logger.Logger
}

func NewProductService(
	productRepo repository.ProductRepository,
	categoryRepo repository.CategoryRepository,
	brandRepo repository.BrandRepository,
	tenantClient tenantv1.TenantServiceClient,
	producer events.Producer,
	l logger.Logger,
) ProductService {
	return &productService{
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		brandRepo:    brandRepo,
		tenantClient: tenantClient,
		producer:     producer,
		l:            l,
	}
}

// ==================== Product ====================

func (s *productService) CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	// 检查配额
	resp, err := s.tenantClient.CheckQuota(ctx, &tenantv1.CheckQuotaRequest{
		TenantId:  p.TenantID,
		QuotaType: "product_count",
	})
	if err != nil {
		return domain.Product{}, err
	}
	if !resp.GetAllowed() {
		return domain.Product{}, ErrQuotaExceeded
	}

	p.Status = domain.ProductStatusDraft
	product, err := s.productRepo.CreateProduct(ctx, p)
	if err != nil {
		return domain.Product{}, err
	}

	// 递增配额
	_, _ = s.tenantClient.IncrQuota(ctx, &tenantv1.IncrQuotaRequest{
		TenantId:  p.TenantID,
		QuotaType: "product_count",
		Delta:     1,
	})

	// 异步发事件
	go func() {
		if er := s.producer.ProduceProductUpdated(context.Background(), events.ProductUpdatedEvent{
			ProductId: product.ID,
			TenantId:  product.TenantID,
		}); er != nil {
			s.l.Error("发送商品更新事件失败", logger.Error(er), logger.Int64("pid", product.ID))
		}
	}()

	return product, nil
}

func (s *productService) GetProduct(ctx context.Context, id int64) (domain.Product, error) {
	return s.productRepo.GetProduct(ctx, id)
}

func (s *productService) UpdateProduct(ctx context.Context, p domain.Product) error {
	err := s.productRepo.UpdateProduct(ctx, p)
	if err != nil {
		return err
	}
	go func() {
		if er := s.producer.ProduceProductUpdated(context.Background(), events.ProductUpdatedEvent{
			ProductId: p.ID,
			TenantId:  p.TenantID,
		}); er != nil {
			s.l.Error("发送商品更新事件失败", logger.Error(er), logger.Int64("pid", p.ID))
		}
	}()
	return nil
}

func (s *productService) UpdateProductStatus(ctx context.Context, id, tenantId int64, status int32) error {
	// 获取旧状态
	old, err := s.productRepo.GetProduct(ctx, id)
	if err != nil {
		return err
	}
	err = s.productRepo.UpdateProductStatus(ctx, id, tenantId, domain.ProductStatus(status))
	if err != nil {
		return err
	}
	go func() {
		if er := s.producer.ProduceProductStatusChanged(context.Background(), events.ProductStatusChangedEvent{
			ProductId: id,
			TenantId:  tenantId,
			OldStatus: int32(old.Status),
			NewStatus: status,
		}); er != nil {
			s.l.Error("发送商品状态变更事件失败", logger.Error(er), logger.Int64("pid", id))
		}
	}()
	return nil
}

func (s *productService) ListProducts(ctx context.Context, tenantId, categoryId int64, status int32, page, pageSize int32) ([]domain.Product, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := int((page - 1) * pageSize)
	limit := int(pageSize)
	return s.productRepo.ListProducts(ctx, tenantId, categoryId, uint8(status), offset, limit)
}

func (s *productService) BatchGetProducts(ctx context.Context, ids []int64) ([]domain.Product, error) {
	return s.productRepo.BatchGetProducts(ctx, ids)
}

func (s *productService) DeleteProduct(ctx context.Context, id, tenantId int64) error {
	err := s.productRepo.DeleteProduct(ctx, id, tenantId)
	if err != nil {
		return err
	}
	// 递减配额
	_, _ = s.tenantClient.DecrQuota(ctx, &tenantv1.DecrQuotaRequest{
		TenantId:  tenantId,
		QuotaType: "product_count",
		Delta:     1,
	})
	return nil
}

// ==================== Category ====================

func (s *productService) CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error) {
	if c.Level > 3 {
		return domain.Category{}, ErrCategoryLevelLimit
	}
	return s.categoryRepo.CreateCategory(ctx, c)
}

func (s *productService) UpdateCategory(ctx context.Context, c domain.Category) error {
	return s.categoryRepo.UpdateCategory(ctx, c)
}

func (s *productService) ListCategories(ctx context.Context, tenantId int64) ([]domain.Category, error) {
	return s.categoryRepo.ListCategories(ctx, tenantId)
}

func (s *productService) DeleteCategory(ctx context.Context, id, tenantId int64) error {
	// 校验无子分类
	childCount, err := s.categoryRepo.CountChildren(ctx, id, tenantId)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return ErrCategoryHasChild
	}
	// 校验无商品引用
	productCount, err := s.productRepo.CountByCategory(ctx, id, tenantId)
	if err != nil {
		return err
	}
	if productCount > 0 {
		return ErrCategoryHasProduct
	}
	return s.categoryRepo.DeleteCategory(ctx, id, tenantId)
}

// ==================== Brand ====================

func (s *productService) CreateBrand(ctx context.Context, b domain.Brand) (domain.Brand, error) {
	return s.brandRepo.CreateBrand(ctx, b)
}

func (s *productService) UpdateBrand(ctx context.Context, b domain.Brand) error {
	return s.brandRepo.UpdateBrand(ctx, b)
}

func (s *productService) ListBrands(ctx context.Context, tenantId int64, page, pageSize int32) ([]domain.Brand, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := int((page - 1) * pageSize)
	limit := int(pageSize)
	return s.brandRepo.ListBrands(ctx, tenantId, offset, limit)
}

func (s *productService) DeleteBrand(ctx context.Context, id, tenantId int64) error {
	productCount, err := s.productRepo.CountByBrand(ctx, id, tenantId)
	if err != nil {
		return err
	}
	if productCount > 0 {
		return ErrBrandHasProduct
	}
	return s.brandRepo.DeleteBrand(ctx, id, tenantId)
}

// ==================== Sales ====================

func (s *productService) IncrSales(ctx context.Context, productId, tenantId int64, count int32) error {
	return s.productRepo.IncrSales(ctx, productId, tenantId, count)
}
```

**验证：** `go build ./product/service/...`

---

## Task 5: gRPC Handler — 17 RPC

**Files:**
- Create: `product/grpc/product.go`

实现全部 17 个 RPC，含 domain ↔ proto 转换和 error 映射。

grpc handler 需实现的 RPC：
- CreateProduct, GetProduct, UpdateProduct, UpdateProductStatus, ListProducts, BatchGetProducts, DeleteProduct
- CreateCategory, UpdateCategory, ListCategories, DeleteCategory
- CreateBrand, UpdateBrand, ListBrands, DeleteBrand
- IncrSales

关键模式（参照 tenant/grpc/tenant.go）：
- 嵌入 `productv1.UnimplementedProductServiceServer`
- `Register(server *grpc.Server)` 注册服务
- `handleErr(err)` 映射 service 错误到 gRPC codes
- DTO 转换函数：`toProductDTO`, `toSKUDTO`, `toSpecDTO`, `toCategoryDTO`, `toBrandDTO`
- ListProducts 不返回 SKUs/Specs（性能），GetProduct 返回完整数据
- Category 的 ListCategories 递归转换树形结构

**验证：** `go build ./product/grpc/...`

---

## Task 6: IoC + Wire + Config + Main

**Files:**
- Create: `product/ioc/db.go`
- Create: `product/ioc/redis.go`
- Create: `product/ioc/kafka.go`
- Create: `product/ioc/logger.go`
- Create: `product/ioc/grpc.go`
- Create: `product/app.go`
- Create: `product/wire.go`
- Create: `product/config/dev.yaml`
- Create: `product/main.go`

### ioc/db.go — 同 tenant-svc，import 改为 product dao

### ioc/redis.go — 同 tenant-svc

### ioc/kafka.go — 同 tenant-svc，InitProducer 返回 product events.Producer

### ioc/logger.go — 同 tenant-svc

### ioc/grpc.go — 关键差异：需要 tenant-svc gRPC client

```go
package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
	igrpc "github.com/rermrf/mall/product/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
)

func InitGRPCServer(productServer *igrpc.ProductGRPCServer, l logger.Logger) *grpcx.Server {
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
	productServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "product",
		L:         l,
	}
}

func InitEtcdClient() *clientv3.Client {
	type Config struct {
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 etcd 配置失败: %w", err))
	}
	client, err := clientv3.New(clientv3.Config{Endpoints: cfg.EtcdAddrs})
	if err != nil {
		panic(fmt.Errorf("连接 etcd 失败: %w", err))
	}
	return client
}

func InitTenantClient(etcdClient *clientv3.Client) tenantv1.TenantServiceClient {
	etcdResolver, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(fmt.Errorf("创建 etcd resolver 失败: %w", err))
	}
	conn, err := grpc.NewClient(
		"etcd:///service/tenant",
		grpc.WithResolvers(etcdResolver),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(tenantx.GRPCUnaryClientInterceptor()),
	)
	if err != nil {
		panic(fmt.Errorf("连接 tenant 服务失败: %w", err))
	}
	return tenantv1.NewTenantServiceClient(conn)
}
```

### config/dev.yaml

```yaml
db:
  dsn: "root:wen123...@tcp(rermrf.icu:3306)/mall_product?charset=utf8mb4&parseTime=true&loc=Local"

redis:
  addr: "rermrf.icu:6379"
  password: ""
  db: 2

kafka:
  addrs:
    - "rermrf.icu:9094"

grpc:
  port: 8083
  etcdAddrs:
    - "rermrf.icu:2379"
```

### wire.go

```go
//go:build wireinject

package main

import (
	"github.com/google/wire"
	igrpc "github.com/rermrf/mall/product/grpc"
	"github.com/rermrf/mall/product/ioc"
	"github.com/rermrf/mall/product/repository"
	"github.com/rermrf/mall/product/repository/cache"
	"github.com/rermrf/mall/product/repository/dao"
	"github.com/rermrf/mall/product/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitTenantClient,
)

var productSet = wire.NewSet(
	dao.NewProductDAO,
	dao.NewSKUDAO,
	dao.NewSpecDAO,
	dao.NewCategoryDAO,
	dao.NewBrandDAO,
	cache.NewProductCache,
	repository.NewProductRepository,
	repository.NewCategoryRepository,
	repository.NewBrandRepository,
	service.NewProductService,
	igrpc.NewProductGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, productSet, wire.Struct(new(App), "*"))
	return new(App)
}
```

### app.go — 同 tenant-svc

### main.go — 同 tenant-svc（`product-svc 启动于...`）

**验证：**
1. `wire ./product/`
2. `go build ./product/...`
3. `go vet ./product/...`

---

## Task 7: 最终验证

```bash
go build ./product/...
go vet ./product/...
```

---

## 文件清单

| # | 文件路径 | 说明 |
|---|---------|------|
| 1 | `product/domain/product.go` | SPU/SKU/Spec 实体 + 枚举 |
| 2 | `product/domain/category.go` | 分类实体 + 枚举 |
| 3 | `product/domain/brand.go` | 品牌实体 + 枚举 |
| 4 | `product/repository/dao/product.go` | Product+SKU+Spec DAO |
| 5 | `product/repository/dao/category.go` | Category DAO |
| 6 | `product/repository/dao/brand.go` | Brand DAO |
| 7 | `product/repository/dao/init.go` | AutoMigrate 5 表 |
| 8 | `product/repository/cache/product.go` | Redis 缓存 |
| 9 | `product/repository/product.go` | Product CachedRepository |
| 10 | `product/repository/category.go` | Category Repository |
| 11 | `product/repository/brand.go` | Brand Repository |
| 12 | `product/events/types.go` | 事件 DTO |
| 13 | `product/events/producer.go` | Kafka Producer |
| 14 | `product/service/product.go` | 业务逻辑 |
| 15 | `product/grpc/product.go` | 17 RPC handler |
| 16 | `product/ioc/db.go` | MySQL 初始化 |
| 17 | `product/ioc/redis.go` | Redis 初始化 |
| 18 | `product/ioc/kafka.go` | Kafka 初始化 |
| 19 | `product/ioc/logger.go` | Logger 初始化 |
| 20 | `product/ioc/grpc.go` | gRPC server + tenant client |
| 21 | `product/config/dev.yaml` | 配置 |
| 22 | `product/app.go` | App 聚合 |
| 23 | `product/wire.go` | Wire DI |
| 24 | `product/main.go` | 入口 |

共 24 个文件。

## 参考文件

- 设计文档：`docs/plans/2026-03-07-product-svc-design.md`
- Proto 定义：`api/proto/product/v1/product.proto`
- 模式参考：`tenant/` 目录全部文件
- 通用包：`pkg/grpcx/`, `pkg/tenantx/`, `pkg/ginx/`
