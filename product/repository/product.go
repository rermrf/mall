package repository

import (
	"context"
	"time"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/product/domain"
	"github.com/rermrf/mall/product/repository/cache"
	"github.com/rermrf/mall/product/repository/dao"
	"gorm.io/gorm"
)

type ProductRepository interface {
	CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error)
	GetProduct(ctx context.Context, id int64) (domain.Product, error)
	UpdateProduct(ctx context.Context, p domain.Product) error
	UpdateProductStatus(ctx context.Context, id, tenantId int64, status domain.ProductStatus) error
	ListProducts(ctx context.Context, tenantId, categoryId int64, status uint8, offset, limit int) ([]domain.Product, int64, error)
	BatchGetProducts(ctx context.Context, ids []int64) ([]domain.Product, error)
	DeleteProduct(ctx context.Context, id, tenantId int64) error
	IncrSales(ctx context.Context, id, tenantId int64, delta int32) error
	CountByCategory(ctx context.Context, categoryId, tenantId int64) (int64, error)
	CountByBrand(ctx context.Context, brandId, tenantId int64) (int64, error)
}

type CachedProductRepository struct {
	productDAO dao.ProductDAO
	skuDAO     dao.SKUDAO
	specDAO    dao.SpecDAO
	cache      cache.ProductCache
	db         *gorm.DB
	l          logger.Logger
}

func NewProductRepository(
	productDAO dao.ProductDAO,
	skuDAO dao.SKUDAO,
	specDAO dao.SpecDAO,
	cache cache.ProductCache,
	db *gorm.DB,
	l logger.Logger,
) ProductRepository {
	return &CachedProductRepository{
		productDAO: productDAO,
		skuDAO:     skuDAO,
		specDAO:    specDAO,
		cache:      cache,
		db:         db,
		l:          l,
	}
}

func (r *CachedProductRepository) CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	var result domain.Product
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		productEntity := r.productToEntity(p)
		created, err := dao.NewProductDAO(tx).Insert(ctx, productEntity)
		if err != nil {
			return err
		}

		if len(p.SKUs) > 0 {
			skuEntities := make([]dao.ProductSKU, 0, len(p.SKUs))
			for _, sku := range p.SKUs {
				e := r.skuToEntity(sku)
				e.ProductID = created.ID
				e.TenantID = p.TenantID
				skuEntities = append(skuEntities, e)
			}
			if err := dao.NewSKUDAO(tx).BatchInsert(ctx, skuEntities); err != nil {
				return err
			}
		}

		if len(p.Specs) > 0 {
			specEntities := make([]dao.ProductSpec, 0, len(p.Specs))
			for _, spec := range p.Specs {
				e := r.specToEntity(spec)
				e.ProductID = created.ID
				e.TenantID = p.TenantID
				specEntities = append(specEntities, e)
			}
			if err := dao.NewSpecDAO(tx).BatchInsert(ctx, specEntities); err != nil {
				return err
			}
		}

		result = r.productToDomain(created)
		return nil
	})
	return result, err
}

func (r *CachedProductRepository) GetProduct(ctx context.Context, id int64) (domain.Product, error) {
	p, err := r.cache.GetProduct(ctx, id)
	if err == nil {
		return p, nil
	}

	entity, err := r.productDAO.FindById(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}
	p = r.productToDomain(entity)

	skuEntities, err := r.skuDAO.FindByProductId(ctx, id)
	if err == nil {
		p.SKUs = make([]domain.SKU, 0, len(skuEntities))
		for _, e := range skuEntities {
			p.SKUs = append(p.SKUs, r.skuToDomain(e))
		}
	}

	specEntities, err := r.specDAO.FindByProductId(ctx, id)
	if err == nil {
		p.Specs = make([]domain.ProductSpec, 0, len(specEntities))
		for _, e := range specEntities {
			p.Specs = append(p.Specs, r.specToDomain(e))
		}
	}

	go func() {
		if er := r.cache.SetProduct(context.Background(), p); er != nil {
			r.l.Error("设置商品缓存失败", logger.Error(er), logger.Int64("pid", id))
		}
	}()
	return p, nil
}

func (r *CachedProductRepository) UpdateProduct(ctx context.Context, p domain.Product) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dao.NewProductDAO(tx).Update(ctx, r.productToEntity(p)); err != nil {
			return err
		}
		skuDAO := dao.NewSKUDAO(tx)
		if err := skuDAO.DeleteByProductId(ctx, p.ID); err != nil {
			return err
		}
		if len(p.SKUs) > 0 {
			skuEntities := make([]dao.ProductSKU, 0, len(p.SKUs))
			for _, sku := range p.SKUs {
				e := r.skuToEntity(sku)
				e.ProductID = p.ID
				e.TenantID = p.TenantID
				skuEntities = append(skuEntities, e)
			}
			if err := skuDAO.BatchInsert(ctx, skuEntities); err != nil {
				return err
			}
		}
		specDAO := dao.NewSpecDAO(tx)
		if err := specDAO.DeleteByProductId(ctx, p.ID); err != nil {
			return err
		}
		if len(p.Specs) > 0 {
			specEntities := make([]dao.ProductSpec, 0, len(p.Specs))
			for _, spec := range p.Specs {
				e := r.specToEntity(spec)
				e.ProductID = p.ID
				e.TenantID = p.TenantID
				specEntities = append(specEntities, e)
			}
			if err := specDAO.BatchInsert(ctx, specEntities); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	_ = r.cache.DeleteProduct(ctx, p.ID)
	return nil
}

func (r *CachedProductRepository) UpdateProductStatus(ctx context.Context, id, tenantId int64, status domain.ProductStatus) error {
	err := r.productDAO.UpdateStatus(ctx, id, tenantId, uint8(status))
	if err != nil {
		return err
	}
	_ = r.cache.DeleteProduct(ctx, id)
	return nil
}

func (r *CachedProductRepository) ListProducts(ctx context.Context, tenantId, categoryId int64, status uint8, offset, limit int) ([]domain.Product, int64, error) {
	entities, total, err := r.productDAO.List(ctx, tenantId, categoryId, status, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	products := make([]domain.Product, 0, len(entities))
	for _, e := range entities {
		products = append(products, r.productToDomain(e))
	}
	return products, total, nil
}

func (r *CachedProductRepository) BatchGetProducts(ctx context.Context, ids []int64) ([]domain.Product, error) {
	entities, err := r.productDAO.BatchFindByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	products := make([]domain.Product, 0, len(entities))
	for _, e := range entities {
		p := r.productToDomain(e)
		skuEntities, er := r.skuDAO.FindByProductId(ctx, e.ID)
		if er == nil {
			p.SKUs = make([]domain.SKU, 0, len(skuEntities))
			for _, se := range skuEntities {
				p.SKUs = append(p.SKUs, r.skuToDomain(se))
			}
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *CachedProductRepository) DeleteProduct(ctx context.Context, id, tenantId int64) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dao.NewSKUDAO(tx).DeleteByProductId(ctx, id); err != nil {
			return err
		}
		if err := dao.NewSpecDAO(tx).DeleteByProductId(ctx, id); err != nil {
			return err
		}
		return dao.NewProductDAO(tx).Delete(ctx, id, tenantId)
	})
	if err != nil {
		return err
	}
	_ = r.cache.DeleteProduct(ctx, id)
	return nil
}

func (r *CachedProductRepository) IncrSales(ctx context.Context, id, tenantId int64, delta int32) error {
	err := r.productDAO.IncrSales(ctx, id, tenantId, delta)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteProduct(ctx, id)
	return nil
}

func (r *CachedProductRepository) CountByCategory(ctx context.Context, categoryId, tenantId int64) (int64, error) {
	return r.productDAO.CountByCategory(ctx, categoryId, tenantId)
}

func (r *CachedProductRepository) CountByBrand(ctx context.Context, brandId, tenantId int64) (int64, error) {
	return r.productDAO.CountByBrand(ctx, brandId, tenantId)
}

// ==================== Converters ====================

func (r *CachedProductRepository) productToEntity(p domain.Product) dao.Product {
	return dao.Product{
		ID: p.ID, TenantID: p.TenantID, CategoryID: p.CategoryID, BrandID: p.BrandID,
		Name: p.Name, Subtitle: p.Subtitle, MainImage: p.MainImage, Images: p.Images,
		Description: p.Description, Status: uint8(p.Status), Sales: p.Sales,
	}
}

func (r *CachedProductRepository) productToDomain(e dao.Product) domain.Product {
	return domain.Product{
		ID: e.ID, TenantID: e.TenantID, CategoryID: e.CategoryID, BrandID: e.BrandID,
		Name: e.Name, Subtitle: e.Subtitle, MainImage: e.MainImage, Images: e.Images,
		Description: e.Description, Status: domain.ProductStatus(e.Status), Sales: e.Sales,
		Ctime: time.UnixMilli(e.Ctime), Utime: time.UnixMilli(e.Utime),
	}
}

func (r *CachedProductRepository) skuToEntity(s domain.SKU) dao.ProductSKU {
	return dao.ProductSKU{
		ID: s.ID, TenantID: s.TenantID, ProductID: s.ProductID,
		SpecValues: s.SpecValues, Price: s.Price, OriginalPrice: s.OriginalPrice,
		CostPrice: s.CostPrice, SKUCode: s.SKUCode, BarCode: s.BarCode, Status: uint8(s.Status),
	}
}

func (r *CachedProductRepository) skuToDomain(e dao.ProductSKU) domain.SKU {
	return domain.SKU{
		ID: e.ID, TenantID: e.TenantID, ProductID: e.ProductID,
		SpecValues: e.SpecValues, Price: e.Price, OriginalPrice: e.OriginalPrice,
		CostPrice: e.CostPrice, SKUCode: e.SKUCode, BarCode: e.BarCode, Status: domain.SKUStatus(e.Status),
		Ctime: time.UnixMilli(e.Ctime), Utime: time.UnixMilli(e.Utime),
	}
}

func (r *CachedProductRepository) specToEntity(s domain.ProductSpec) dao.ProductSpec {
	return dao.ProductSpec{
		ID: s.ID, ProductID: s.ProductID, TenantID: s.TenantID, Name: s.Name, Values: s.Values,
	}
}

func (r *CachedProductRepository) specToDomain(e dao.ProductSpec) domain.ProductSpec {
	return domain.ProductSpec{
		ID: e.ID, ProductID: e.ProductID, TenantID: e.TenantID, Name: e.Name, Values: e.Values,
	}
}
