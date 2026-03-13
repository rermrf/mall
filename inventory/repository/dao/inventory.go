package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InventoryModel 库存表
type InventoryModel struct {
	ID             int64 `gorm:"primaryKey;autoIncrement"`
	TenantId       int64 `gorm:"uniqueIndex:uk_tenant_sku"`
	SkuId          int64 `gorm:"uniqueIndex:uk_tenant_sku"`
	Total          int32
	Available      int32
	Locked         int32
	Sold           int32
	AlertThreshold int32
	Ctime          int64
	Utime          int64
}

func (InventoryModel) TableName() string {
	return "inventories"
}

// InventoryLogModel 库存变更日志表
type InventoryLogModel struct {
	ID              int64 `gorm:"primaryKey;autoIncrement"`
	SkuId           int64 `gorm:"index:idx_sku"`
	OrderId         int64 `gorm:"index:idx_order"`
	Type            uint8
	Quantity        int32
	BeforeAvailable int32
	AfterAvailable  int32
	TenantId        int64
	Ctime           int64
}

func (InventoryLogModel) TableName() string {
	return "inventory_logs"
}

type InventoryDAO interface {
	Upsert(ctx context.Context, inv InventoryModel) error
	FindBySKUID(ctx context.Context, skuId int64) (InventoryModel, error)
	FindBySKUIDs(ctx context.Context, skuIds []int64) ([]InventoryModel, error)
	UpdateStockByConfirm(ctx context.Context, tx *gorm.DB, skuId int64, qty int32) error
	InsertLog(ctx context.Context, tx *gorm.DB, log InventoryLogModel) error
	ListLogs(ctx context.Context, tenantId, skuId int64, offset, limit int) ([]InventoryLogModel, int64, error)
	GetDB() *gorm.DB
}

type GORMInventoryDAO struct {
	db *gorm.DB
}

func NewInventoryDAO(db *gorm.DB) InventoryDAO {
	return &GORMInventoryDAO{db: db}
}

func (d *GORMInventoryDAO) GetDB() *gorm.DB {
	return d.db
}

func (d *GORMInventoryDAO) Upsert(ctx context.Context, inv InventoryModel) error {
	now := time.Now().UnixMilli()
	inv.Ctime = now
	inv.Utime = now
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "sku_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"total":           inv.Total,
			"available":       gorm.Expr("? - locked - sold", inv.Total),
			"alert_threshold": inv.AlertThreshold,
			"utime":           now,
		}),
	}).Create(&inv).Error
}

func (d *GORMInventoryDAO) FindBySKUID(ctx context.Context, skuId int64) (InventoryModel, error) {
	var inv InventoryModel
	err := d.db.WithContext(ctx).Where("sku_id = ?", skuId).First(&inv).Error
	return inv, err
}

func (d *GORMInventoryDAO) FindBySKUIDs(ctx context.Context, skuIds []int64) ([]InventoryModel, error) {
	var invs []InventoryModel
	err := d.db.WithContext(ctx).Where("sku_id IN ?", skuIds).Find(&invs).Error
	return invs, err
}

func (d *GORMInventoryDAO) UpdateStockByConfirm(ctx context.Context, tx *gorm.DB, skuId int64, qty int32) error {
	return tx.WithContext(ctx).Model(&InventoryModel{}).
		Where("sku_id = ? AND locked >= ?", skuId, qty).
		Updates(map[string]any{
			"locked": gorm.Expr("locked - ?", qty),
			"sold":   gorm.Expr("sold + ?", qty),
			"utime":  time.Now().UnixMilli(),
		}).Error
}

func (d *GORMInventoryDAO) InsertLog(ctx context.Context, tx *gorm.DB, log InventoryLogModel) error {
	log.Ctime = time.Now().UnixMilli()
	return tx.WithContext(ctx).Create(&log).Error
}

func (d *GORMInventoryDAO) ListLogs(ctx context.Context, tenantId, skuId int64, offset, limit int) ([]InventoryLogModel, int64, error) {
	var logs []InventoryLogModel
	var total int64
	query := d.db.WithContext(ctx).Model(&InventoryLogModel{}).Where("tenant_id = ?", tenantId)
	if skuId > 0 {
		query = query.Where("sku_id = ?", skuId)
	}
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = query.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}
