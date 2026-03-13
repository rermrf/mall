package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Tenant struct {
	ID              int64  `gorm:"primaryKey;autoIncrement"`
	Name            string `gorm:"type:varchar(100);not null"`
	ContactName     string `gorm:"type:varchar(50)"`
	ContactPhone    string `gorm:"type:varchar(20)"`
	BusinessLicense string `gorm:"type:varchar(500)"`
	Status          uint8  `gorm:"default:1;not null"`
	PlanId          int64  `gorm:"not null"`
	PlanExpireTime  int64  `gorm:"not null"`
	Ctime           int64  `gorm:"not null"`
	Utime           int64  `gorm:"not null"`
}

type TenantDAO interface {
	Insert(ctx context.Context, t Tenant) (Tenant, error)
	FindById(ctx context.Context, id int64) (Tenant, error)
	Update(ctx context.Context, t Tenant) error
	UpdateStatus(ctx context.Context, id int64, status uint8) error
	List(ctx context.Context, offset, limit int, status uint8) ([]Tenant, int64, error)
}

type GORMTenantDAO struct {
	db *gorm.DB
}

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
	return d.db.WithContext(ctx).Where("id = ?", t.ID).Updates(&t).Error
}

func (d *GORMTenantDAO) UpdateStatus(ctx context.Context, id int64, status uint8) error {
	return d.db.WithContext(ctx).
		Model(&Tenant{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status": status,
			"utime":  time.Now().UnixMilli(),
		}).Error
}

func (d *GORMTenantDAO) List(ctx context.Context, offset, limit int, status uint8) ([]Tenant, int64, error) {
	db := d.db.WithContext(ctx).Model(&Tenant{})
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	var tenants []Tenant
	err = db.Offset(offset).Limit(limit).Order("id DESC").Find(&tenants).Error
	return tenants, total, err
}
