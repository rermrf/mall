package repository

import (
	"context"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/tenant/domain"
	"github.com/rermrf/mall/tenant/repository/dao"
)

func (r *CachedTenantRepository) CreateShop(ctx context.Context, s domain.Shop) (domain.Shop, error) {
	entity := r.shopDomainToEntity(s)
	res, err := r.shopDAO.Insert(ctx, entity)
	if err != nil {
		return domain.Shop{}, err
	}
	return r.shopEntityToDomain(res), nil
}

func (r *CachedTenantRepository) GetShop(ctx context.Context, tenantId int64) (domain.Shop, error) {
	// Cache-Aside: try cache first
	s, err := r.cache.GetShop(ctx, tenantId)
	if err == nil {
		return s, nil
	}
	entity, err := r.shopDAO.FindByTenantId(ctx, tenantId)
	if err != nil {
		return domain.Shop{}, err
	}
	s = r.shopEntityToDomain(entity)
	// async set cache
	go func() {
		if er := r.cache.SetShop(context.Background(), s); er != nil {
			r.l.Error("设置店铺缓存失败", logger.Error(er), logger.Int64("tid", tenantId))
		}
	}()
	return s, nil
}

func (r *CachedTenantRepository) UpdateShop(ctx context.Context, s domain.Shop) error {
	// Fetch old shop to invalidate previous domain cache keys
	old, err := r.shopDAO.FindByTenantId(ctx, s.TenantID)
	if err != nil {
		return err
	}
	entity := r.shopDomainToEntity(s)
	err = r.shopDAO.Update(ctx, entity)
	if err != nil {
		return err
	}
	// Invalidate shop info cache
	_ = r.cache.DeleteShop(ctx, s.TenantID)
	// Invalidate old domain caches
	if old.Subdomain != "" {
		_ = r.cache.DeleteShopByDomain(ctx, old.Subdomain)
	}
	if old.CustomDomain != "" {
		_ = r.cache.DeleteShopByDomain(ctx, old.CustomDomain)
	}
	return nil
}

func (r *CachedTenantRepository) GetShopByDomain(ctx context.Context, domainName string) (domain.Shop, error) {
	// Cache-Aside: try cache first
	s, err := r.cache.GetShopByDomain(ctx, domainName)
	if err == nil {
		return s, nil
	}
	entity, err := r.shopDAO.FindByDomain(ctx, domainName)
	if err != nil {
		return domain.Shop{}, err
	}
	s = r.shopEntityToDomain(entity)
	// async set cache
	go func() {
		if er := r.cache.SetShopByDomain(context.Background(), domainName, s); er != nil {
			r.l.Error("设置店铺域名缓存失败", logger.Error(er))
		}
	}()
	return s, nil
}

// ==================== Shop Converters ====================

func (r *CachedTenantRepository) shopEntityToDomain(e dao.Shop) domain.Shop {
	return domain.Shop{
		ID:           e.ID,
		TenantID:     e.TenantId,
		Name:         e.Name,
		Logo:         e.Logo,
		Description:  e.Description,
		Status:       domain.ShopStatus(e.Status),
		Rating:       e.Rating,
		Subdomain:    e.Subdomain,
		CustomDomain: e.CustomDomain,
		Ctime:        e.Ctime,
		Utime:        e.Utime,
	}
}

func (r *CachedTenantRepository) shopDomainToEntity(s domain.Shop) dao.Shop {
	return dao.Shop{
		ID:           s.ID,
		TenantId:     s.TenantID,
		Name:         s.Name,
		Logo:         s.Logo,
		Description:  s.Description,
		Status:       uint8(s.Status),
		Rating:       s.Rating,
		Subdomain:    s.Subdomain,
		CustomDomain: s.CustomDomain,
	}
}
