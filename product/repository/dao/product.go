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
	TenantID      int64  `gorm:"not null;uniqueIndex:uk_sku_code"`
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
