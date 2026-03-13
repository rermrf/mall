package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/rermrf/mall/cart/domain"
	"github.com/rermrf/mall/cart/repository/cache"
	"github.com/rermrf/mall/cart/repository/dao"
)

type CartRepository interface {
	AddItem(ctx context.Context, item domain.CartItem) error
	UpdateItem(ctx context.Context, userId, skuId int64, updates map[string]any) error
	RemoveItem(ctx context.Context, userId, skuId int64) error
	GetCart(ctx context.Context, userId int64) ([]domain.CartItem, error)
	ClearCart(ctx context.Context, userId int64) error
	BatchRemoveItems(ctx context.Context, userId int64, skuIds []int64) error
}

type cartRepository struct {
	dao   dao.CartDAO
	cache cache.CartCache
}

func NewCartRepository(d dao.CartDAO, c cache.CartCache) CartRepository {
	return &cartRepository{dao: d, cache: c}
}

func (r *cartRepository) AddItem(ctx context.Context, item domain.CartItem) error {
	err := r.dao.Upsert(ctx, r.toModel(item))
	if err != nil {
		return err
	}
	_ = r.cache.Delete(ctx, item.UserID)
	return nil
}

func (r *cartRepository) UpdateItem(ctx context.Context, userId, skuId int64, updates map[string]any) error {
	err := r.dao.Update(ctx, userId, skuId, updates)
	if err != nil {
		return err
	}
	_ = r.cache.Delete(ctx, userId)
	return nil
}

func (r *cartRepository) RemoveItem(ctx context.Context, userId, skuId int64) error {
	err := r.dao.Delete(ctx, userId, skuId)
	if err != nil {
		return err
	}
	_ = r.cache.Delete(ctx, userId)
	return nil
}

func (r *cartRepository) GetCart(ctx context.Context, userId int64) ([]domain.CartItem, error) {
	data, err := r.cache.Get(ctx, userId)
	if err == nil {
		var items []domain.CartItem
		if json.Unmarshal(data, &items) == nil {
			return items, nil
		}
	}
	if err != nil && err != redis.Nil {
		// Redis 错误只记录，不阻塞
	}
	models, err := r.dao.FindByUser(ctx, userId)
	if err != nil {
		return nil, err
	}
	items := make([]domain.CartItem, 0, len(models))
	for _, m := range models {
		items = append(items, r.toDomain(m))
	}
	if jsonData, e := json.Marshal(items); e == nil {
		_ = r.cache.Set(ctx, userId, jsonData)
	}
	return items, nil
}

func (r *cartRepository) ClearCart(ctx context.Context, userId int64) error {
	err := r.dao.DeleteByUser(ctx, userId)
	if err != nil {
		return err
	}
	_ = r.cache.Delete(ctx, userId)
	return nil
}

func (r *cartRepository) BatchRemoveItems(ctx context.Context, userId int64, skuIds []int64) error {
	err := r.dao.BatchDelete(ctx, userId, skuIds)
	if err != nil {
		return err
	}
	_ = r.cache.Delete(ctx, userId)
	return nil
}

func (r *cartRepository) toModel(item domain.CartItem) dao.CartItemModel {
	return dao.CartItemModel{
		UserId:    item.UserID,
		SkuId:     item.SkuID,
		ProductId: item.ProductID,
		TenantId:  item.TenantID,
		Quantity:  item.Quantity,
		Selected:  item.Selected,
	}
}

func (r *cartRepository) toDomain(m dao.CartItemModel) domain.CartItem {
	return domain.CartItem{
		ID:        m.ID,
		UserID:    m.UserId,
		SkuID:     m.SkuId,
		ProductID: m.ProductId,
		TenantID:  m.TenantId,
		Quantity:  m.Quantity,
		Selected:  m.Selected,
		Ctime:     time.UnixMilli(m.Ctime),
		Utime:     time.UnixMilli(m.Utime),
	}
}
