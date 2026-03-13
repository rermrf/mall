package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type TenantPlan struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	Name         string `gorm:"type:varchar(50);not null"`
	Price        int64  `gorm:"not null"`
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

type PlanDAO interface {
	Insert(ctx context.Context, p TenantPlan) (TenantPlan, error)
	FindById(ctx context.Context, id int64) (TenantPlan, error)
	Update(ctx context.Context, p TenantPlan) error
	ListAll(ctx context.Context) ([]TenantPlan, error)
}

type GORMPlanDAO struct {
	db *gorm.DB
}

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
	return d.db.WithContext(ctx).Where("id = ?", p.ID).Updates(&p).Error
}

func (d *GORMPlanDAO) ListAll(ctx context.Context) ([]TenantPlan, error) {
	var plans []TenantPlan
	err := d.db.WithContext(ctx).Order("id ASC").Find(&plans).Error
	return plans, err
}
