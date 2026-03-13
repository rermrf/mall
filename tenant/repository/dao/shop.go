package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

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

type ShopDAO interface {
	Insert(ctx context.Context, s Shop) (Shop, error)
	FindByTenantId(ctx context.Context, tenantId int64) (Shop, error)
	Update(ctx context.Context, s Shop) error
	FindByDomain(ctx context.Context, domain string) (Shop, error)
}

type GORMShopDAO struct {
	db *gorm.DB
}

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
	return d.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", s.ID, s.TenantId).
		Updates(&s).Error
}

func (d *GORMShopDAO) FindByDomain(ctx context.Context, domain string) (Shop, error) {
	var s Shop
	err := d.db.WithContext(ctx).
		Where("subdomain = ? OR custom_domain = ?", domain, domain).
		First(&s).Error
	return s, err
}
