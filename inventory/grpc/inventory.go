package grpc

import (
	"context"

	"google.golang.org/grpc"

	inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"
	"github.com/rermrf/mall/inventory/domain"
	"github.com/rermrf/mall/inventory/service"
)

type InventoryGRPCServer struct {
	inventoryv1.UnimplementedInventoryServiceServer
	svc service.InventoryService
}

func NewInventoryGRPCServer(svc service.InventoryService) *InventoryGRPCServer {
	return &InventoryGRPCServer{svc: svc}
}

func (s *InventoryGRPCServer) Register(server *grpc.Server) {
	inventoryv1.RegisterInventoryServiceServer(server, s)
}

func (s *InventoryGRPCServer) SetStock(ctx context.Context, req *inventoryv1.SetStockRequest) (*inventoryv1.SetStockResponse, error) {
	err := s.svc.SetStock(ctx, req.GetTenantId(), req.GetSkuId(), req.GetTotal(), req.GetAlertThreshold())
	if err != nil {
		return nil, err
	}
	return &inventoryv1.SetStockResponse{}, nil
}

func (s *InventoryGRPCServer) GetStock(ctx context.Context, req *inventoryv1.GetStockRequest) (*inventoryv1.GetStockResponse, error) {
	inv, err := s.svc.GetStock(ctx, req.GetSkuId())
	if err != nil {
		return nil, err
	}
	return &inventoryv1.GetStockResponse{
		Inventory: s.toDTO(inv),
	}, nil
}

func (s *InventoryGRPCServer) BatchGetStock(ctx context.Context, req *inventoryv1.BatchGetStockRequest) (*inventoryv1.BatchGetStockResponse, error) {
	invs, err := s.svc.BatchGetStock(ctx, req.GetSkuIds())
	if err != nil {
		return nil, err
	}
	dtos := make([]*inventoryv1.Inventory, 0, len(invs))
	for _, inv := range invs {
		dtos = append(dtos, s.toDTO(inv))
	}
	return &inventoryv1.BatchGetStockResponse{
		Inventories: dtos,
	}, nil
}

func (s *InventoryGRPCServer) Deduct(ctx context.Context, req *inventoryv1.DeductRequest) (*inventoryv1.DeductResponse, error) {
	items := make([]domain.DeductItem, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		items = append(items, domain.DeductItem{
			SKUID:    item.GetSkuId(),
			Quantity: item.GetQuantity(),
		})
	}
	success, msg, err := s.svc.Deduct(ctx, req.GetOrderId(), req.GetTenantId(), items)
	if err != nil {
		return nil, err
	}
	return &inventoryv1.DeductResponse{
		Success: success,
		Message: msg,
	}, nil
}

func (s *InventoryGRPCServer) Confirm(ctx context.Context, req *inventoryv1.ConfirmRequest) (*inventoryv1.ConfirmResponse, error) {
	err := s.svc.Confirm(ctx, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &inventoryv1.ConfirmResponse{}, nil
}

func (s *InventoryGRPCServer) Rollback(ctx context.Context, req *inventoryv1.RollbackRequest) (*inventoryv1.RollbackResponse, error) {
	err := s.svc.Rollback(ctx, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &inventoryv1.RollbackResponse{}, nil
}

func (s *InventoryGRPCServer) ListLogs(ctx context.Context, req *inventoryv1.ListLogsRequest) (*inventoryv1.ListLogsResponse, error) {
	logs, total, err := s.svc.ListLogs(ctx, req.GetTenantId(), req.GetSkuId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*inventoryv1.InventoryLog, 0, len(logs))
	for _, log := range logs {
		dtos = append(dtos, s.toLogDTO(log))
	}
	return &inventoryv1.ListLogsResponse{
		Logs:  dtos,
		Total: total,
	}, nil
}

func (s *InventoryGRPCServer) toDTO(inv domain.Inventory) *inventoryv1.Inventory {
	return &inventoryv1.Inventory{
		Id:             inv.ID,
		TenantId:       inv.TenantID,
		SkuId:          inv.SKUID,
		Total:          inv.Total,
		Available:      inv.Available,
		Locked:         inv.Locked,
		Sold:           inv.Sold,
		AlertThreshold: inv.AlertThreshold,
		Ctime:          inv.Ctime.UnixMilli(),
		Utime:          inv.Utime.UnixMilli(),
	}
}

func (s *InventoryGRPCServer) toLogDTO(log domain.InventoryLog) *inventoryv1.InventoryLog {
	return &inventoryv1.InventoryLog{
		Id:              log.ID,
		SkuId:           log.SKUID,
		OrderId:         log.OrderID,
		Type:            int32(log.Type),
		Quantity:        log.Quantity,
		BeforeAvailable: log.BeforeAvailable,
		AfterAvailable:  log.AfterAvailable,
		TenantId:        log.TenantID,
		Ctime:           log.Ctime.UnixMilli(),
	}
}
