package repository

import (
	"context"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/product/domain"
	"github.com/rermrf/mall/product/repository/dao"
)

type BrandRepository interface {
	CreateBrand(ctx context.Context, b domain.Brand) (domain.Brand, error)
	UpdateBrand(ctx context.Context, b domain.Brand) error
	ListBrands(ctx context.Context, tenantId int64, offset, limit int) ([]domain.Brand, int64, error)
	DeleteBrand(ctx context.Context, id, tenantId int64) error
}

type CachedBrandRepository struct {
	brandDAO dao.BrandDAO
	l        logger.Logger
}

func NewBrandRepository(brandDAO dao.BrandDAO, l logger.Logger) BrandRepository {
	return &CachedBrandRepository{brandDAO: brandDAO, l: l}
}

func (r *CachedBrandRepository) CreateBrand(ctx context.Context, b domain.Brand) (domain.Brand, error) {
	entity := r.toEntity(b)
	res, err := r.brandDAO.Insert(ctx, entity)
	if err != nil {
		return domain.Brand{}, err
	}
	return r.toDomain(res), nil
}

func (r *CachedBrandRepository) UpdateBrand(ctx context.Context, b domain.Brand) error {
	return r.brandDAO.Update(ctx, r.toEntity(b))
}

func (r *CachedBrandRepository) ListBrands(ctx context.Context, tenantId int64, offset, limit int) ([]domain.Brand, int64, error) {
	entities, total, err := r.brandDAO.List(ctx, tenantId, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	brands := make([]domain.Brand, 0, len(entities))
	for _, e := range entities {
		brands = append(brands, r.toDomain(e))
	}
	return brands, total, nil
}

func (r *CachedBrandRepository) DeleteBrand(ctx context.Context, id, tenantId int64) error {
	return r.brandDAO.Delete(ctx, id, tenantId)
}

func (r *CachedBrandRepository) toEntity(b domain.Brand) dao.BrandModel {
	return dao.BrandModel{
		ID: b.ID, TenantID: b.TenantID, Name: b.Name, Logo: b.Logo, Status: uint8(b.Status),
	}
}

func (r *CachedBrandRepository) toDomain(e dao.BrandModel) domain.Brand {
	return domain.Brand{
		ID: e.ID, TenantID: e.TenantID, Name: e.Name, Logo: e.Logo, Status: domain.BrandStatus(e.Status),
	}
}
