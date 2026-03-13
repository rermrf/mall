package service

import (
	"context"
	"errors"

	"github.com/rermrf/emo/logger"
	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/product/domain"
	"github.com/rermrf/mall/product/events"
	"github.com/rermrf/mall/product/repository"
)

var (
	ErrProductNotFound    = errors.New("商品不存在")
	ErrCategoryNotFound   = errors.New("分类不存在")
	ErrBrandNotFound      = errors.New("品牌不存在")
	ErrQuotaExceeded      = errors.New("商品配额已超限")
	ErrCategoryHasChild   = errors.New("该分类下有子分类，不能删除")
	ErrCategoryHasProduct = errors.New("该分类下有商品，不能删除")
	ErrBrandHasProduct    = errors.New("该品牌下有商品，不能删除")
	ErrCategoryLevelLimit = errors.New("分类层级不能超过3级")
)

type ProductService interface {
	CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error)
	GetProduct(ctx context.Context, id int64) (domain.Product, error)
	UpdateProduct(ctx context.Context, p domain.Product) error
	UpdateProductStatus(ctx context.Context, id, tenantId int64, status int32) error
	ListProducts(ctx context.Context, tenantId, categoryId int64, status int32, page, pageSize int32) ([]domain.Product, int64, error)
	BatchGetProducts(ctx context.Context, ids []int64) ([]domain.Product, error)
	DeleteProduct(ctx context.Context, id, tenantId int64) error

	CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error)
	UpdateCategory(ctx context.Context, c domain.Category) error
	ListCategories(ctx context.Context, tenantId int64) ([]domain.Category, error)
	DeleteCategory(ctx context.Context, id, tenantId int64) error

	CreateBrand(ctx context.Context, b domain.Brand) (domain.Brand, error)
	UpdateBrand(ctx context.Context, b domain.Brand) error
	ListBrands(ctx context.Context, tenantId int64, page, pageSize int32) ([]domain.Brand, int64, error)
	DeleteBrand(ctx context.Context, id, tenantId int64) error

	IncrSales(ctx context.Context, productId, tenantId int64, count int32) error
}

type productService struct {
	productRepo  repository.ProductRepository
	categoryRepo repository.CategoryRepository
	brandRepo    repository.BrandRepository
	tenantClient tenantv1.TenantServiceClient
	producer     events.Producer
	l            logger.Logger
}

func NewProductService(
	productRepo repository.ProductRepository,
	categoryRepo repository.CategoryRepository,
	brandRepo repository.BrandRepository,
	tenantClient tenantv1.TenantServiceClient,
	producer events.Producer,
	l logger.Logger,
) ProductService {
	return &productService{
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		brandRepo:    brandRepo,
		tenantClient: tenantClient,
		producer:     producer,
		l:            l,
	}
}

func (s *productService) CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	resp, err := s.tenantClient.CheckQuota(ctx, &tenantv1.CheckQuotaRequest{
		TenantId:  p.TenantID,
		QuotaType: "product_count",
	})
	if err != nil {
		return domain.Product{}, err
	}
	if !resp.GetAllowed() {
		return domain.Product{}, ErrQuotaExceeded
	}

	p.Status = domain.ProductStatusDraft
	product, err := s.productRepo.CreateProduct(ctx, p)
	if err != nil {
		return domain.Product{}, err
	}

	_, _ = s.tenantClient.IncrQuota(ctx, &tenantv1.IncrQuotaRequest{
		TenantId:  p.TenantID,
		QuotaType: "product_count",
		Delta:     1,
	})

	go func() {
		if er := s.producer.ProduceProductUpdated(context.Background(), events.ProductUpdatedEvent{
			ProductId: product.ID,
			TenantId:  product.TenantID,
		}); er != nil {
			s.l.Error("发送商品更新事件失败", logger.Error(er), logger.Int64("pid", product.ID))
		}
	}()

	return product, nil
}

func (s *productService) GetProduct(ctx context.Context, id int64) (domain.Product, error) {
	return s.productRepo.GetProduct(ctx, id)
}

func (s *productService) UpdateProduct(ctx context.Context, p domain.Product) error {
	err := s.productRepo.UpdateProduct(ctx, p)
	if err != nil {
		return err
	}
	go func() {
		if er := s.producer.ProduceProductUpdated(context.Background(), events.ProductUpdatedEvent{
			ProductId: p.ID,
			TenantId:  p.TenantID,
		}); er != nil {
			s.l.Error("发送商品更新事件失败", logger.Error(er), logger.Int64("pid", p.ID))
		}
	}()
	return nil
}

func (s *productService) UpdateProductStatus(ctx context.Context, id, tenantId int64, status int32) error {
	old, err := s.productRepo.GetProduct(ctx, id)
	if err != nil {
		return err
	}
	err = s.productRepo.UpdateProductStatus(ctx, id, tenantId, domain.ProductStatus(status))
	if err != nil {
		return err
	}
	go func() {
		if er := s.producer.ProduceProductStatusChanged(context.Background(), events.ProductStatusChangedEvent{
			ProductId: id,
			TenantId:  tenantId,
			OldStatus: int32(old.Status),
			NewStatus: status,
		}); er != nil {
			s.l.Error("发送商品状态变更事件失败", logger.Error(er), logger.Int64("pid", id))
		}
	}()
	return nil
}

func (s *productService) ListProducts(ctx context.Context, tenantId, categoryId int64, status int32, page, pageSize int32) ([]domain.Product, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := int((page - 1) * pageSize)
	limit := int(pageSize)
	return s.productRepo.ListProducts(ctx, tenantId, categoryId, uint8(status), offset, limit)
}

func (s *productService) BatchGetProducts(ctx context.Context, ids []int64) ([]domain.Product, error) {
	return s.productRepo.BatchGetProducts(ctx, ids)
}

func (s *productService) DeleteProduct(ctx context.Context, id, tenantId int64) error {
	err := s.productRepo.DeleteProduct(ctx, id, tenantId)
	if err != nil {
		return err
	}
	_, _ = s.tenantClient.DecrQuota(ctx, &tenantv1.DecrQuotaRequest{
		TenantId:  tenantId,
		QuotaType: "product_count",
		Delta:     1,
	})
	return nil
}

func (s *productService) CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error) {
	if c.Level > 3 {
		return domain.Category{}, ErrCategoryLevelLimit
	}
	return s.categoryRepo.CreateCategory(ctx, c)
}

func (s *productService) UpdateCategory(ctx context.Context, c domain.Category) error {
	return s.categoryRepo.UpdateCategory(ctx, c)
}

func (s *productService) ListCategories(ctx context.Context, tenantId int64) ([]domain.Category, error) {
	return s.categoryRepo.ListCategories(ctx, tenantId)
}

func (s *productService) DeleteCategory(ctx context.Context, id, tenantId int64) error {
	childCount, err := s.categoryRepo.CountChildren(ctx, id, tenantId)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return ErrCategoryHasChild
	}
	productCount, err := s.productRepo.CountByCategory(ctx, id, tenantId)
	if err != nil {
		return err
	}
	if productCount > 0 {
		return ErrCategoryHasProduct
	}
	return s.categoryRepo.DeleteCategory(ctx, id, tenantId)
}

func (s *productService) CreateBrand(ctx context.Context, b domain.Brand) (domain.Brand, error) {
	return s.brandRepo.CreateBrand(ctx, b)
}

func (s *productService) UpdateBrand(ctx context.Context, b domain.Brand) error {
	return s.brandRepo.UpdateBrand(ctx, b)
}

func (s *productService) ListBrands(ctx context.Context, tenantId int64, page, pageSize int32) ([]domain.Brand, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := int((page - 1) * pageSize)
	limit := int(pageSize)
	return s.brandRepo.ListBrands(ctx, tenantId, offset, limit)
}

func (s *productService) DeleteBrand(ctx context.Context, id, tenantId int64) error {
	productCount, err := s.productRepo.CountByBrand(ctx, id, tenantId)
	if err != nil {
		return err
	}
	if productCount > 0 {
		return ErrBrandHasProduct
	}
	return s.brandRepo.DeleteBrand(ctx, id, tenantId)
}

func (s *productService) IncrSales(ctx context.Context, productId, tenantId int64, count int32) error {
	return s.productRepo.IncrSales(ctx, productId, tenantId, count)
}
