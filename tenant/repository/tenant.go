package repository

import (
	"context"
	"time"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/tenant/domain"
	"github.com/rermrf/mall/tenant/repository/cache"
	"github.com/rermrf/mall/tenant/repository/dao"
)

type TenantRepository interface {
	// Tenant CRUD
	CreateTenant(ctx context.Context, t domain.Tenant) (domain.Tenant, error)
	GetTenant(ctx context.Context, id int64) (domain.Tenant, error)
	UpdateTenant(ctx context.Context, t domain.Tenant) error
	UpdateTenantStatus(ctx context.Context, id int64, status domain.TenantStatus) error
	ListTenants(ctx context.Context, offset, limit int, status uint8) ([]domain.Tenant, int64, error)

	// Plan
	CreatePlan(ctx context.Context, p domain.TenantPlan) (domain.TenantPlan, error)
	GetPlan(ctx context.Context, id int64) (domain.TenantPlan, error)
	UpdatePlan(ctx context.Context, p domain.TenantPlan) error
	ListPlans(ctx context.Context) ([]domain.TenantPlan, error)

	// Quota
	CheckQuota(ctx context.Context, tenantId int64, quotaType string) (domain.QuotaUsage, error)
	InitQuota(ctx context.Context, tenantId int64, quotaType string, maxLimit int32) error
	IncrQuota(ctx context.Context, tenantId int64, quotaType string, delta int32) error
	DecrQuota(ctx context.Context, tenantId int64, quotaType string, delta int32) error

	// Shop
	CreateShop(ctx context.Context, s domain.Shop) (domain.Shop, error)
	GetShop(ctx context.Context, tenantId int64) (domain.Shop, error)
	UpdateShop(ctx context.Context, s domain.Shop) error
	GetShopByDomain(ctx context.Context, domain string) (domain.Shop, error)
}

type CachedTenantRepository struct {
	tenantDAO dao.TenantDAO
	planDAO   dao.PlanDAO
	quotaDAO  dao.QuotaDAO
	shopDAO   dao.ShopDAO
	cache     cache.TenantCache
	l         logger.Logger
}

func NewTenantRepository(
	tenantDAO dao.TenantDAO,
	planDAO dao.PlanDAO,
	quotaDAO dao.QuotaDAO,
	shopDAO dao.ShopDAO,
	cache cache.TenantCache,
	l logger.Logger,
) TenantRepository {
	return &CachedTenantRepository{
		tenantDAO: tenantDAO,
		planDAO:   planDAO,
		quotaDAO:  quotaDAO,
		shopDAO:   shopDAO,
		cache:     cache,
		l:         l,
	}
}

// ==================== Tenant CRUD ====================

func (r *CachedTenantRepository) CreateTenant(ctx context.Context, t domain.Tenant) (domain.Tenant, error) {
	entity := r.domainToEntity(t)
	res, err := r.tenantDAO.Insert(ctx, entity)
	if err != nil {
		return domain.Tenant{}, err
	}
	return r.entityToDomain(res), nil
}

func (r *CachedTenantRepository) GetTenant(ctx context.Context, id int64) (domain.Tenant, error) {
	// Cache-Aside: try cache first
	t, err := r.cache.GetTenant(ctx, id)
	if err == nil {
		return t, nil
	}
	entity, err := r.tenantDAO.FindById(ctx, id)
	if err != nil {
		return domain.Tenant{}, err
	}
	t = r.entityToDomain(entity)
	// async set cache, don't block
	go func() {
		if er := r.cache.SetTenant(context.Background(), t); er != nil {
			r.l.Error("设置租户缓存失败", logger.Error(er), logger.Int64("tid", id))
		}
	}()
	return t, nil
}

func (r *CachedTenantRepository) UpdateTenant(ctx context.Context, t domain.Tenant) error {
	entity := r.domainToEntity(t)
	err := r.tenantDAO.Update(ctx, entity)
	if err != nil {
		return err
	}
	return r.cache.DeleteTenant(ctx, t.ID)
}

func (r *CachedTenantRepository) UpdateTenantStatus(ctx context.Context, id int64, status domain.TenantStatus) error {
	err := r.tenantDAO.UpdateStatus(ctx, id, uint8(status))
	if err != nil {
		return err
	}
	return r.cache.DeleteTenant(ctx, id)
}

func (r *CachedTenantRepository) ListTenants(ctx context.Context, offset, limit int, status uint8) ([]domain.Tenant, int64, error) {
	entities, total, err := r.tenantDAO.List(ctx, offset, limit, status)
	if err != nil {
		return nil, 0, err
	}
	tenants := make([]domain.Tenant, 0, len(entities))
	for _, e := range entities {
		tenants = append(tenants, r.entityToDomain(e))
	}
	return tenants, total, nil
}

// ==================== Tenant Converters ====================

func (r *CachedTenantRepository) entityToDomain(e dao.Tenant) domain.Tenant {
	return domain.Tenant{
		ID:              e.ID,
		Name:            e.Name,
		ContactName:     e.ContactName,
		ContactPhone:    e.ContactPhone,
		BusinessLicense: e.BusinessLicense,
		Status:          domain.TenantStatus(e.Status),
		PlanID:          e.PlanId,
		PlanExpireTime:  e.PlanExpireTime,
		Ctime:           time.UnixMilli(e.Ctime),
		Utime:           time.UnixMilli(e.Utime),
	}
}

func (r *CachedTenantRepository) domainToEntity(t domain.Tenant) dao.Tenant {
	return dao.Tenant{
		ID:              t.ID,
		Name:            t.Name,
		ContactName:     t.ContactName,
		ContactPhone:    t.ContactPhone,
		BusinessLicense: t.BusinessLicense,
		Status:          uint8(t.Status),
		PlanId:          t.PlanID,
		PlanExpireTime:  t.PlanExpireTime,
	}
}
