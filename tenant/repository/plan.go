package repository

import (
	"context"

	"github.com/rermrf/mall/tenant/domain"
	"github.com/rermrf/mall/tenant/repository/dao"
)

func (r *CachedTenantRepository) CreatePlan(ctx context.Context, p domain.TenantPlan) (domain.TenantPlan, error) {
	entity := r.planDomainToEntity(p)
	res, err := r.planDAO.Insert(ctx, entity)
	if err != nil {
		return domain.TenantPlan{}, err
	}
	return r.planEntityToDomain(res), nil
}

func (r *CachedTenantRepository) GetPlan(ctx context.Context, id int64) (domain.TenantPlan, error) {
	entity, err := r.planDAO.FindById(ctx, id)
	if err != nil {
		return domain.TenantPlan{}, err
	}
	return r.planEntityToDomain(entity), nil
}

func (r *CachedTenantRepository) UpdatePlan(ctx context.Context, p domain.TenantPlan) error {
	entity := r.planDomainToEntity(p)
	return r.planDAO.Update(ctx, entity)
}

func (r *CachedTenantRepository) ListPlans(ctx context.Context) ([]domain.TenantPlan, error) {
	entities, err := r.planDAO.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	plans := make([]domain.TenantPlan, 0, len(entities))
	for _, e := range entities {
		plans = append(plans, r.planEntityToDomain(e))
	}
	return plans, nil
}

// ==================== Plan Converters ====================

func (r *CachedTenantRepository) planEntityToDomain(e dao.TenantPlan) domain.TenantPlan {
	return domain.TenantPlan{
		ID:           e.ID,
		Name:         e.Name,
		Price:        e.Price,
		DurationDays: e.DurationDays,
		MaxProducts:  e.MaxProducts,
		MaxStaff:     e.MaxStaff,
		Features:     e.Features,
		Status:       domain.PlanStatus(e.Status),
		Ctime:        e.Ctime,
		Utime:        e.Utime,
	}
}

func (r *CachedTenantRepository) planDomainToEntity(p domain.TenantPlan) dao.TenantPlan {
	return dao.TenantPlan{
		ID:           p.ID,
		Name:         p.Name,
		Price:        p.Price,
		DurationDays: p.DurationDays,
		MaxProducts:  p.MaxProducts,
		MaxStaff:     p.MaxStaff,
		Features:     p.Features,
		Status:       uint8(p.Status),
	}
}
