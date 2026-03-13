package repository

import (
	"context"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/tenant/domain"
	"github.com/rermrf/mall/tenant/repository/dao"
)

func (r *CachedTenantRepository) CheckQuota(ctx context.Context, tenantId int64, quotaType string) (domain.QuotaUsage, error) {
	// Cache-Aside: try cache first
	q, err := r.cache.GetQuota(ctx, tenantId, quotaType)
	if err == nil {
		return q, nil
	}
	entity, err := r.quotaDAO.FindByTenantAndType(ctx, tenantId, quotaType)
	if err != nil {
		return domain.QuotaUsage{}, err
	}
	q = domain.QuotaUsage{
		QuotaType: entity.QuotaType,
		Used:      entity.Used,
		MaxLimit:  entity.MaxLimit,
	}
	// async set cache
	go func() {
		if er := r.cache.SetQuota(context.Background(), tenantId, quotaType, q); er != nil {
			r.l.Error("设置配额缓存失败", logger.Error(er), logger.Int64("tid", tenantId))
		}
	}()
	return q, nil
}

func (r *CachedTenantRepository) InitQuota(ctx context.Context, tenantId int64, quotaType string, maxLimit int32) error {
	return r.quotaDAO.Upsert(ctx, dao.TenantQuotaUsage{
		TenantId:  tenantId,
		QuotaType: quotaType,
		MaxLimit:  maxLimit,
	})
}

func (r *CachedTenantRepository) IncrQuota(ctx context.Context, tenantId int64, quotaType string, delta int32) error {
	err := r.quotaDAO.IncrUsed(ctx, tenantId, quotaType, delta)
	if err != nil {
		return err
	}
	return r.cache.DeleteQuota(ctx, tenantId, quotaType)
}

func (r *CachedTenantRepository) DecrQuota(ctx context.Context, tenantId int64, quotaType string, delta int32) error {
	err := r.quotaDAO.DecrUsed(ctx, tenantId, quotaType, delta)
	if err != nil {
		return err
	}
	return r.cache.DeleteQuota(ctx, tenantId, quotaType)
}
