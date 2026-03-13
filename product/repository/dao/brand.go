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
