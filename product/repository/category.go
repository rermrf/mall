package repository

import (
	"context"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/product/domain"
	"github.com/rermrf/mall/product/repository/cache"
	"github.com/rermrf/mall/product/repository/dao"
)

type CategoryRepository interface {
	CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error)
	UpdateCategory(ctx context.Context, c domain.Category) error
	ListCategories(ctx context.Context, tenantId int64) ([]domain.Category, error)
	DeleteCategory(ctx context.Context, id, tenantId int64) error
	CountChildren(ctx context.Context, parentId, tenantId int64) (int64, error)
}

type CachedCategoryRepository struct {
	categoryDAO dao.CategoryDAO
	cache       cache.ProductCache
	l           logger.Logger
}

func NewCategoryRepository(categoryDAO dao.CategoryDAO, cache cache.ProductCache, l logger.Logger) CategoryRepository {
	return &CachedCategoryRepository{categoryDAO: categoryDAO, cache: cache, l: l}
}

func (r *CachedCategoryRepository) CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error) {
	entity := r.toEntity(c)
	res, err := r.categoryDAO.Insert(ctx, entity)
	if err != nil {
		return domain.Category{}, err
	}
	_ = r.cache.DeleteCategoryTree(ctx, c.TenantID)
	return r.toDomain(res), nil
}

func (r *CachedCategoryRepository) UpdateCategory(ctx context.Context, c domain.Category) error {
	err := r.categoryDAO.Update(ctx, r.toEntity(c))
	if err != nil {
		return err
	}
	_ = r.cache.DeleteCategoryTree(ctx, c.TenantID)
	return nil
}

func (r *CachedCategoryRepository) ListCategories(ctx context.Context, tenantId int64) ([]domain.Category, error) {
	tree, err := r.cache.GetCategoryTree(ctx, tenantId)
	if err == nil {
		return tree, nil
	}
	entities, err := r.categoryDAO.FindAllByTenant(ctx, tenantId)
	if err != nil {
		return nil, err
	}
	all := make([]domain.Category, 0, len(entities))
	for _, e := range entities {
		all = append(all, r.toDomain(e))
	}
	tree = buildTree(all, 0)
	go func() {
		if er := r.cache.SetCategoryTree(context.Background(), tenantId, tree); er != nil {
			r.l.Error("设置分类树缓存失败", logger.Error(er), logger.Int64("tid", tenantId))
		}
	}()
	return tree, nil
}

func (r *CachedCategoryRepository) DeleteCategory(ctx context.Context, id, tenantId int64) error {
	err := r.categoryDAO.Delete(ctx, id, tenantId)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteCategoryTree(ctx, tenantId)
	return nil
}

func (r *CachedCategoryRepository) CountChildren(ctx context.Context, parentId, tenantId int64) (int64, error) {
	return r.categoryDAO.CountChildren(ctx, parentId, tenantId)
}

func buildTree(categories []domain.Category, parentId int64) []domain.Category {
	var tree []domain.Category
	for _, c := range categories {
		if c.ParentID == parentId {
			c.Children = buildTree(categories, c.ID)
			tree = append(tree, c)
		}
	}
	return tree
}

func (r *CachedCategoryRepository) toEntity(c domain.Category) dao.CategoryModel {
	return dao.CategoryModel{
		ID: c.ID, TenantID: c.TenantID, ParentID: c.ParentID,
		Name: c.Name, Level: c.Level, Sort: c.Sort, Icon: c.Icon, Status: uint8(c.Status),
	}
}

func (r *CachedCategoryRepository) toDomain(e dao.CategoryModel) domain.Category {
	return domain.Category{
		ID: e.ID, TenantID: e.TenantID, ParentID: e.ParentID,
		Name: e.Name, Level: e.Level, Sort: e.Sort, Icon: e.Icon, Status: domain.CategoryStatus(e.Status),
	}
}
