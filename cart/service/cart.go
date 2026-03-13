package service

import (
	"context"
	"fmt"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/cart/domain"
	"github.com/rermrf/mall/cart/repository"
)

type CartService interface {
	AddItem(ctx context.Context, item domain.CartItem) error
	UpdateItem(ctx context.Context, userId, skuId int64, quantity int32, selected bool, updateSelected bool) error
	RemoveItem(ctx context.Context, userId, skuId int64) error
	GetCart(ctx context.Context, userId int64) ([]domain.CartItem, error)
	ClearCart(ctx context.Context, userId int64) error
	BatchRemoveItems(ctx context.Context, userId int64, skuIds []int64) error
}

type cartService struct {
	repo repository.CartRepository
	l    logger.Logger
}

func NewCartService(repo repository.CartRepository, l logger.Logger) CartService {
	return &cartService{repo: repo, l: l}
}

func (s *cartService) AddItem(ctx context.Context, item domain.CartItem) error {
	if item.Quantity <= 0 {
		return fmt.Errorf("数量必须大于 0")
	}
	return s.repo.AddItem(ctx, item)
}

func (s *cartService) UpdateItem(ctx context.Context, userId, skuId int64, quantity int32, selected bool, updateSelected bool) error {
	updates := make(map[string]any)
	if quantity > 0 {
		updates["quantity"] = quantity
	}
	if updateSelected {
		updates["selected"] = selected
	}
	if len(updates) == 0 {
		return nil
	}
	return s.repo.UpdateItem(ctx, userId, skuId, updates)
}

func (s *cartService) RemoveItem(ctx context.Context, userId, skuId int64) error {
	return s.repo.RemoveItem(ctx, userId, skuId)
}

func (s *cartService) GetCart(ctx context.Context, userId int64) ([]domain.CartItem, error) {
	return s.repo.GetCart(ctx, userId)
}

func (s *cartService) ClearCart(ctx context.Context, userId int64) error {
	return s.repo.ClearCart(ctx, userId)
}

func (s *cartService) BatchRemoveItems(ctx context.Context, userId int64, skuIds []int64) error {
	if len(skuIds) == 0 {
		return nil
	}
	return s.repo.BatchRemoveItems(ctx, userId, skuIds)
}
