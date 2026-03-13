package grpc

import (
	"context"

	"google.golang.org/grpc"

	cartv1 "github.com/rermrf/mall/api/proto/gen/cart/v1"
	"github.com/rermrf/mall/cart/domain"
	"github.com/rermrf/mall/cart/service"
)

type CartGRPCServer struct {
	cartv1.UnimplementedCartServiceServer
	svc service.CartService
}

func NewCartGRPCServer(svc service.CartService) *CartGRPCServer {
	return &CartGRPCServer{svc: svc}
}

func (s *CartGRPCServer) Register(server *grpc.Server) {
	cartv1.RegisterCartServiceServer(server, s)
}

func (s *CartGRPCServer) AddItem(ctx context.Context, req *cartv1.AddItemRequest) (*cartv1.AddItemResponse, error) {
	err := s.svc.AddItem(ctx, domain.CartItem{
		UserID:    req.GetUserId(),
		SkuID:     req.GetSkuId(),
		ProductID: req.GetProductId(),
		TenantID:  req.GetTenantId(),
		Quantity:  req.GetQuantity(),
	})
	if err != nil {
		return nil, err
	}
	return &cartv1.AddItemResponse{}, nil
}

func (s *CartGRPCServer) UpdateItem(ctx context.Context, req *cartv1.UpdateItemRequest) (*cartv1.UpdateItemResponse, error) {
	err := s.svc.UpdateItem(ctx, req.GetUserId(), req.GetSkuId(), req.GetQuantity(), req.GetSelected(), req.GetUpdateSelected())
	if err != nil {
		return nil, err
	}
	return &cartv1.UpdateItemResponse{}, nil
}

func (s *CartGRPCServer) RemoveItem(ctx context.Context, req *cartv1.RemoveItemRequest) (*cartv1.RemoveItemResponse, error) {
	err := s.svc.RemoveItem(ctx, req.GetUserId(), req.GetSkuId())
	if err != nil {
		return nil, err
	}
	return &cartv1.RemoveItemResponse{}, nil
}

func (s *CartGRPCServer) GetCart(ctx context.Context, req *cartv1.GetCartRequest) (*cartv1.GetCartResponse, error) {
	items, err := s.svc.GetCart(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	pbItems := make([]*cartv1.CartItem, 0, len(items))
	for _, item := range items {
		pbItems = append(pbItems, s.toDTO(item))
	}
	return &cartv1.GetCartResponse{Items: pbItems}, nil
}

func (s *CartGRPCServer) ClearCart(ctx context.Context, req *cartv1.ClearCartRequest) (*cartv1.ClearCartResponse, error) {
	err := s.svc.ClearCart(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &cartv1.ClearCartResponse{}, nil
}

func (s *CartGRPCServer) BatchRemoveItems(ctx context.Context, req *cartv1.BatchRemoveItemsRequest) (*cartv1.BatchRemoveItemsResponse, error) {
	err := s.svc.BatchRemoveItems(ctx, req.GetUserId(), req.GetSkuIds())
	if err != nil {
		return nil, err
	}
	return &cartv1.BatchRemoveItemsResponse{}, nil
}

func (s *CartGRPCServer) toDTO(item domain.CartItem) *cartv1.CartItem {
	return &cartv1.CartItem{
		SkuId:     item.SkuID,
		ProductId: item.ProductID,
		TenantId:  item.TenantID,
		Quantity:  item.Quantity,
		Selected:  item.Selected,
	}
}
