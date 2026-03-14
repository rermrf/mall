package service

import (
	"context"
	"fmt"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/inventory/domain"
	"github.com/rermrf/mall/inventory/events"
	"github.com/rermrf/mall/inventory/repository"
)

type InventoryService interface {
	SetStock(ctx context.Context, tenantId, skuId int64, total, alertThreshold int32) error
	GetStock(ctx context.Context, skuId int64) (domain.Inventory, error)
	BatchGetStock(ctx context.Context, skuIds []int64) ([]domain.Inventory, error)
	Deduct(ctx context.Context, orderId, tenantId int64, items []domain.DeductItem) (bool, string, error)
	Confirm(ctx context.Context, orderId int64) error
	Rollback(ctx context.Context, orderId int64) error
	ListLogs(ctx context.Context, tenantId, skuId int64, page, pageSize int32) ([]domain.InventoryLog, int64, error)
}

type inventoryService struct {
	repo     repository.InventoryRepository
	producer events.Producer
	l        logger.Logger
}

func NewInventoryService(
	repo repository.InventoryRepository,
	producer events.Producer,
	l logger.Logger,
) InventoryService {
	return &inventoryService{repo: repo, producer: producer, l: l}
}

func (s *inventoryService) SetStock(ctx context.Context, tenantId, skuId int64, total, alertThreshold int32) error {
	if total < 0 {
		return fmt.Errorf("库存总量不能为负数")
	}
	inv := domain.Inventory{
		TenantID:       tenantId,
		SKUID:          skuId,
		Total:          total,
		Available:      total,
		AlertThreshold: alertThreshold,
	}
	return s.repo.SetStock(ctx, inv)
}

func (s *inventoryService) GetStock(ctx context.Context, skuId int64) (domain.Inventory, error) {
	return s.repo.GetStock(ctx, skuId)
}

func (s *inventoryService) BatchGetStock(ctx context.Context, skuIds []int64) ([]domain.Inventory, error) {
	return s.repo.BatchGetStock(ctx, skuIds)
}

// Deduct 预扣库存：Redis Lua 原子预扣 → 存 Redis Hash → 发 go-delay 延迟消息
func (s *inventoryService) Deduct(ctx context.Context, orderId, tenantId int64, items []domain.DeductItem) (bool, string, error) {
	for _, item := range items {
		if item.Quantity <= 0 {
			return false, fmt.Sprintf("SKU %d 的扣减数量必须大于0", item.SKUID), nil
		}
	}
	itemMap := make(map[int64]int32, len(items))
	for _, item := range items {
		itemMap[item.SKUID] = item.Quantity
	}

	// 1. Redis Lua 原子预扣
	success, msg, err := s.repo.Deduct(ctx, itemMap)
	if err != nil {
		return false, "", err
	}
	if !success {
		return false, msg, nil
	}

	// 2. 存 Redis Hash 预扣记录
	err = s.repo.SetDeductRecord(ctx, orderId, itemMap)
	if err != nil {
		if rollbackErr := s.repo.RollbackStock(ctx, itemMap); rollbackErr != nil {
			s.l.Error("回滚库存失败", logger.Int64("orderId", orderId), logger.Error(rollbackErr))
		}
		return false, "系统错误", err
	}

	// 3. 发 go-delay 延迟消息（30min 后触发回滚检查）
	err = s.producer.ProduceDelay(ctx, orderId)
	if err != nil {
		s.l.Error("发送延迟消息失败",
			logger.Int64("orderId", orderId),
			logger.Error(err),
		)
	}

	return true, "", nil
}

// Confirm 确认扣减（幂等）：读 Redis Hash → MySQL 事务更新 → 删 Redis Hash
func (s *inventoryService) Confirm(ctx context.Context, orderId int64) error {
	items, err := s.repo.GetDeductRecord(ctx, orderId)
	if err != nil {
		return err
	}
	if items == nil {
		return nil
	}

	var tenantId int64
	for skuId := range items {
		inv, e := s.repo.GetStock(ctx, skuId)
		if e == nil {
			tenantId = inv.TenantID
			break
		}
	}

	err = s.repo.ConfirmStock(ctx, items, orderId, tenantId)
	if err != nil {
		return err
	}

	if deleteErr := s.repo.DeleteDeductRecord(ctx, orderId); deleteErr != nil {
		s.l.Error("删除预扣记录失败", logger.Int64("orderId", orderId), logger.Error(deleteErr))
	}

	for skuId := range items {
		inv, e := s.repo.GetStock(ctx, skuId)
		if e == nil && inv.Available < inv.AlertThreshold {
			if alertErr := s.producer.ProduceAlert(ctx, events.InventoryAlertEvent{
				TenantID:  inv.TenantID,
				SKUID:     skuId,
				Available: inv.Available,
				Threshold: inv.AlertThreshold,
			}); alertErr != nil {
				s.l.Error("发送库存预警失败", logger.Int64("skuId", skuId), logger.Error(alertErr))
			}
		}
	}

	return nil
}

// Rollback 回滚库存（幂等）：读 Redis Hash → Redis Lua 回滚 → MySQL 写日志 → 删 Redis Hash
func (s *inventoryService) Rollback(ctx context.Context, orderId int64) error {
	items, err := s.repo.GetDeductRecord(ctx, orderId)
	if err != nil {
		return err
	}
	if items == nil {
		return nil
	}

	err = s.repo.RollbackStock(ctx, items)
	if err != nil {
		return err
	}

	var tenantId int64
	for skuId := range items {
		inv, e := s.repo.GetStock(ctx, skuId)
		if e == nil {
			tenantId = inv.TenantID
			break
		}
	}

	for skuId, qty := range items {
		if logErr := s.repo.InsertLog(ctx, domain.InventoryLog{
			SKUID:    skuId,
			OrderID:  orderId,
			Type:     domain.LogTypeRollback,
			Quantity: qty,
			TenantID: tenantId,
		}); logErr != nil {
			s.l.Error("写入库存日志失败", logger.Int64("orderId", orderId), logger.Int64("skuId", skuId), logger.Error(logErr))
		}
	}

	if deleteErr := s.repo.DeleteDeductRecord(ctx, orderId); deleteErr != nil {
		s.l.Error("删除预扣记录失败", logger.Int64("orderId", orderId), logger.Error(deleteErr))
	}
	return nil
}

func (s *inventoryService) ListLogs(ctx context.Context, tenantId, skuId int64, page, pageSize int32) ([]domain.InventoryLog, int64, error) {
	return s.repo.ListLogs(ctx, tenantId, skuId, page, pageSize)
}
