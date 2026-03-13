package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInsufficientQuota = errors.New("insufficient quota usage to decrement")

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

type QuotaDAO interface {
	FindByTenantAndType(ctx context.Context, tenantId int64, quotaType string) (TenantQuotaUsage, error)
	Upsert(ctx context.Context, q TenantQuotaUsage) error
	IncrUsed(ctx context.Context, tenantId int64, quotaType string, delta int32) error
	DecrUsed(ctx context.Context, tenantId int64, quotaType string, delta int32) error
}

type GORMQuotaDAO struct {
	db *gorm.DB
}

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
	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "quota_type"}},
			DoUpdates: clause.AssignmentColumns([]string{"max_limit", "utime"}),
		}).Create(&q).Error
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
	result := d.db.WithContext(ctx).
		Model(&TenantQuotaUsage{}).
		Where("tenant_id = ? AND quota_type = ? AND used >= ?", tenantId, quotaType, delta).
		Updates(map[string]any{
			"used":  gorm.Expr("used - ?", delta),
			"utime": time.Now().UnixMilli(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInsufficientQuota
	}
	return nil
}
