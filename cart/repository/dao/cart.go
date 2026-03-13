package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CartItemModel struct {
	ID        int64 `gorm:"primaryKey;autoIncrement"`
	UserId    int64 `gorm:"uniqueIndex:uk_user_sku;index:idx_user"`
	SkuId     int64 `gorm:"uniqueIndex:uk_user_sku"`
	ProductId int64
	TenantId  int64
	Quantity  int32
	Selected  bool
	Ctime     int64
	Utime     int64
}

func (CartItemModel) TableName() string { return "cart_items" }

type CartDAO interface {
	Upsert(ctx context.Context, item CartItemModel) error
	Update(ctx context.Context, userId, skuId int64, updates map[string]any) error
	Delete(ctx context.Context, userId, skuId int64) error
	FindByUser(ctx context.Context, userId int64) ([]CartItemModel, error)
	DeleteByUser(ctx context.Context, userId int64) error
	BatchDelete(ctx context.Context, userId int64, skuIds []int64) error
}

type GORMCartDAO struct {
	db *gorm.DB
}

func NewCartDAO(db *gorm.DB) CartDAO {
	return &GORMCartDAO{db: db}
}

func (d *GORMCartDAO) Upsert(ctx context.Context, item CartItemModel) error {
	now := time.Now().UnixMilli()
	item.Ctime = now
	item.Utime = now
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "sku_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"quantity": gorm.Expr("quantity + ?", item.Quantity),
			"utime":    now,
		}),
	}).Create(&item).Error
}

func (d *GORMCartDAO) Update(ctx context.Context, userId, skuId int64, updates map[string]any) error {
	updates["utime"] = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Model(&CartItemModel{}).
		Where("user_id = ? AND sku_id = ?", userId, skuId).
		Updates(updates).Error
}

func (d *GORMCartDAO) Delete(ctx context.Context, userId, skuId int64) error {
	return d.db.WithContext(ctx).
		Where("user_id = ? AND sku_id = ?", userId, skuId).
		Delete(&CartItemModel{}).Error
}

func (d *GORMCartDAO) FindByUser(ctx context.Context, userId int64) ([]CartItemModel, error) {
	var items []CartItemModel
	err := d.db.WithContext(ctx).Where("user_id = ?", userId).
		Order("id DESC").Find(&items).Error
	return items, err
}

func (d *GORMCartDAO) DeleteByUser(ctx context.Context, userId int64) error {
	return d.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Delete(&CartItemModel{}).Error
}

func (d *GORMCartDAO) BatchDelete(ctx context.Context, userId int64, skuIds []int64) error {
	return d.db.WithContext(ctx).
		Where("user_id = ? AND sku_id IN ?", userId, skuIds).
		Delete(&CartItemModel{}).Error
}
