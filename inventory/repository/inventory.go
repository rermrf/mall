package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/rermrf/mall/inventory/domain"
	"github.com/rermrf/mall/inventory/repository/cache"
	"github.com/rermrf/mall/inventory/repository/dao"
)

type InventoryRepository interface {
	SetStock(ctx context.Context, inv domain.Inventory) error
	GetStock(ctx context.Context, skuId int64) (domain.Inventory, error)
	BatchGetStock(ctx context.Context, skuIds []int64) ([]domain.Inventory, error)
	Deduct(ctx context.Context, items map[int64]int32) (bool, string, error)
	GetDeductRecord(ctx context.Context, orderId int64) (map[int64]int32, error)
	SetDeductRecord(ctx context.Context, orderId int64, items map[int64]int32) error
	DeleteDeductRecord(ctx context.Context, orderId int64) error
	RollbackStock(ctx context.Context, items map[int64]int32) error
	ConfirmStock(ctx context.Context, items map[int64]int32, orderId, tenantId int64) error
	InsertLog(ctx context.Context, log domain.InventoryLog) error
	ListLogs(ctx context.Context, tenantId, skuId int64, page, pageSize int32) ([]domain.InventoryLog, int64, error)
}

type inventoryRepository struct {
	dao   dao.InventoryDAO
	cache cache.InventoryCache
}

func NewInventoryRepository(d dao.InventoryDAO, c cache.InventoryCache) InventoryRepository {
	return &inventoryRepository{dao: d, cache: c}
}

func (r *inventoryRepository) SetStock(ctx context.Context, inv domain.Inventory) error {
	err := r.dao.Upsert(ctx, r.domainToEntity(inv))
	if err != nil {
		return err
	}
	entity, err := r.dao.FindBySKUID(ctx, inv.SKUID)
	if err != nil {
		return err
	}
	return r.cache.SetStock(ctx, entity.SkuId, entity.Total, entity.Available, entity.Locked, entity.Sold, entity.AlertThreshold)
}

func (r *inventoryRepository) GetStock(ctx context.Context, skuId int64) (domain.Inventory, error) {
	total, available, locked, sold, alertThreshold, err := r.cache.GetStock(ctx, skuId)
	if err == nil {
		return domain.Inventory{
			SKUID:          skuId,
			Total:          total,
			Available:      available,
			Locked:         locked,
			Sold:           sold,
			AlertThreshold: alertThreshold,
		}, nil
	}
	if err != redis.Nil {
		return domain.Inventory{}, err
	}
	entity, err := r.dao.FindBySKUID(ctx, skuId)
	if err != nil {
		return domain.Inventory{}, err
	}
	inv := r.entityToDomain(entity)
	_ = r.cache.SetStock(ctx, entity.SkuId, entity.Total, entity.Available, entity.Locked, entity.Sold, entity.AlertThreshold)
	return inv, nil
}

func (r *inventoryRepository) BatchGetStock(ctx context.Context, skuIds []int64) ([]domain.Inventory, error) {
	entities, err := r.dao.FindBySKUIDs(ctx, skuIds)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Inventory, 0, len(entities))
	for _, e := range entities {
		result = append(result, r.entityToDomain(e))
	}
	return result, nil
}

func (r *inventoryRepository) Deduct(ctx context.Context, items map[int64]int32) (bool, string, error) {
	return r.cache.Deduct(ctx, items)
}

func (r *inventoryRepository) GetDeductRecord(ctx context.Context, orderId int64) (map[int64]int32, error) {
	return r.cache.GetDeductRecord(ctx, orderId)
}

func (r *inventoryRepository) SetDeductRecord(ctx context.Context, orderId int64, items map[int64]int32) error {
	return r.cache.SetDeductRecord(ctx, orderId, items)
}

func (r *inventoryRepository) DeleteDeductRecord(ctx context.Context, orderId int64) error {
	return r.cache.DeleteDeductRecord(ctx, orderId)
}

func (r *inventoryRepository) RollbackStock(ctx context.Context, items map[int64]int32) error {
	return r.cache.Rollback(ctx, items)
}

func (r *inventoryRepository) ConfirmStock(ctx context.Context, items map[int64]int32, orderId, tenantId int64) error {
	db := r.dao.GetDB()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for skuId, qty := range items {
			var inv dao.InventoryModel
			if err := tx.Where("sku_id = ?", skuId).First(&inv).Error; err != nil {
				return err
			}
			err := r.dao.UpdateStockByConfirm(ctx, tx, skuId, qty)
			if err != nil {
				return err
			}
			err = r.dao.InsertLog(ctx, tx, dao.InventoryLogModel{
				SkuId:           skuId,
				OrderId:         orderId,
				Type:            uint8(domain.LogTypeConfirm),
				Quantity:        qty,
				BeforeAvailable: inv.Available,
				AfterAvailable:  inv.Available,
				TenantId:        tenantId,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *inventoryRepository) InsertLog(ctx context.Context, log domain.InventoryLog) error {
	db := r.dao.GetDB()
	return r.dao.InsertLog(ctx, db, dao.InventoryLogModel{
		SkuId:           log.SKUID,
		OrderId:         log.OrderID,
		Type:            uint8(log.Type),
		Quantity:        log.Quantity,
		BeforeAvailable: log.BeforeAvailable,
		AfterAvailable:  log.AfterAvailable,
		TenantId:        log.TenantID,
	})
}

func (r *inventoryRepository) ListLogs(ctx context.Context, tenantId, skuId int64, page, pageSize int32) ([]domain.InventoryLog, int64, error) {
	offset := int((page - 1) * pageSize)
	entities, total, err := r.dao.ListLogs(ctx, tenantId, skuId, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	logs := make([]domain.InventoryLog, 0, len(entities))
	for _, e := range entities {
		logs = append(logs, domain.InventoryLog{
			ID:              e.ID,
			SKUID:           e.SkuId,
			OrderID:         e.OrderId,
			Type:            domain.LogType(e.Type),
			Quantity:        e.Quantity,
			BeforeAvailable: e.BeforeAvailable,
			AfterAvailable:  e.AfterAvailable,
			TenantID:        e.TenantId,
			Ctime:           time.UnixMilli(e.Ctime),
		})
	}
	return logs, total, nil
}

func (r *inventoryRepository) domainToEntity(inv domain.Inventory) dao.InventoryModel {
	return dao.InventoryModel{
		ID:             inv.ID,
		TenantId:       inv.TenantID,
		SkuId:          inv.SKUID,
		Total:          inv.Total,
		Available:      inv.Available,
		Locked:         inv.Locked,
		Sold:           inv.Sold,
		AlertThreshold: inv.AlertThreshold,
	}
}

func (r *inventoryRepository) entityToDomain(e dao.InventoryModel) domain.Inventory {
	return domain.Inventory{
		ID:             e.ID,
		TenantID:       e.TenantId,
		SKUID:          e.SkuId,
		Total:          e.Total,
		Available:      e.Available,
		Locked:         e.Locked,
		Sold:           e.Sold,
		AlertThreshold: e.AlertThreshold,
		Ctime:          time.UnixMilli(e.Ctime),
		Utime:          time.UnixMilli(e.Utime),
	}
}
