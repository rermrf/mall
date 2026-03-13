package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/product/domain"
	"github.com/rermrf/mall/product/service"
)

type ProductGRPCServer struct {
	productv1.UnimplementedProductServiceServer
	svc service.ProductService
}

func NewProductGRPCServer(svc service.ProductService) *ProductGRPCServer {
	return &ProductGRPCServer{svc: svc}
}

func (s *ProductGRPCServer) Register(server *grpc.Server) {
	productv1.RegisterProductServiceServer(server, s)
}

// ==================== Product ====================

func (s *ProductGRPCServer) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.CreateProductResponse, error) {
	p := req.GetProduct()
	skus := make([]domain.SKU, 0, len(p.GetSkus()))
	for _, sku := range p.GetSkus() {
		skus = append(skus, toDomainSKU(sku))
	}
	specs := make([]domain.ProductSpec, 0, len(p.GetSpecs()))
	for _, spec := range p.GetSpecs() {
		specs = append(specs, toDomainSpec(spec))
	}
	product, err := s.svc.CreateProduct(ctx, domain.Product{
		TenantID:    p.GetTenantId(),
		CategoryID:  p.GetCategoryId(),
		BrandID:     p.GetBrandId(),
		Name:        p.GetName(),
		Subtitle:    p.GetSubtitle(),
		MainImage:   p.GetMainImage(),
		Images:      p.GetImages(),
		Description: p.GetDescription(),
		SKUs:        skus,
		Specs:       specs,
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.CreateProductResponse{Id: product.ID}, nil
}

func (s *ProductGRPCServer) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.GetProductResponse, error) {
	product, err := s.svc.GetProduct(ctx, req.GetId())
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.GetProductResponse{Product: toProductDTO(product)}, nil
}

func (s *ProductGRPCServer) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.UpdateProductResponse, error) {
	p := req.GetProduct()
	skus := make([]domain.SKU, 0, len(p.GetSkus()))
	for _, sku := range p.GetSkus() {
		skus = append(skus, toDomainSKU(sku))
	}
	specs := make([]domain.ProductSpec, 0, len(p.GetSpecs()))
	for _, spec := range p.GetSpecs() {
		specs = append(specs, toDomainSpec(spec))
	}
	err := s.svc.UpdateProduct(ctx, domain.Product{
		ID:          p.GetId(),
		TenantID:    p.GetTenantId(),
		CategoryID:  p.GetCategoryId(),
		BrandID:     p.GetBrandId(),
		Name:        p.GetName(),
		Subtitle:    p.GetSubtitle(),
		MainImage:   p.GetMainImage(),
		Images:      p.GetImages(),
		Description: p.GetDescription(),
		Status:      domain.ProductStatus(p.GetStatus()),
		SKUs:        skus,
		Specs:       specs,
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.UpdateProductResponse{}, nil
}

func (s *ProductGRPCServer) UpdateProductStatus(ctx context.Context, req *productv1.UpdateProductStatusRequest) (*productv1.UpdateProductStatusResponse, error) {
	err := s.svc.UpdateProductStatus(ctx, req.GetId(), req.GetTenantId(), req.GetStatus())
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.UpdateProductStatusResponse{}, nil
}

func (s *ProductGRPCServer) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	products, total, err := s.svc.ListProducts(ctx, req.GetTenantId(), req.GetCategoryId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, handleErr(err)
	}
	dtos := make([]*productv1.Product, 0, len(products))
	for _, p := range products {
		dtos = append(dtos, toProductListDTO(p))
	}
	return &productv1.ListProductsResponse{Products: dtos, Total: total}, nil
}

func (s *ProductGRPCServer) BatchGetProducts(ctx context.Context, req *productv1.BatchGetProductsRequest) (*productv1.BatchGetProductsResponse, error) {
	products, err := s.svc.BatchGetProducts(ctx, req.GetIds())
	if err != nil {
		return nil, handleErr(err)
	}
	dtos := make([]*productv1.Product, 0, len(products))
	for _, p := range products {
		dtos = append(dtos, toBatchProductDTO(p))
	}
	return &productv1.BatchGetProductsResponse{Products: dtos}, nil
}

func (s *ProductGRPCServer) DeleteProduct(ctx context.Context, req *productv1.DeleteProductRequest) (*productv1.DeleteProductResponse, error) {
	err := s.svc.DeleteProduct(ctx, req.GetId(), req.GetTenantId())
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.DeleteProductResponse{}, nil
}

// ==================== Category ====================

func (s *ProductGRPCServer) CreateCategory(ctx context.Context, req *productv1.CreateCategoryRequest) (*productv1.CreateCategoryResponse, error) {
	c := req.GetCategory()
	category, err := s.svc.CreateCategory(ctx, domain.Category{
		TenantID: c.GetTenantId(),
		ParentID: c.GetParentId(),
		Name:     c.GetName(),
		Level:    c.GetLevel(),
		Sort:     c.GetSort(),
		Icon:     c.GetIcon(),
		Status:   domain.CategoryStatus(c.GetStatus()),
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.CreateCategoryResponse{Id: category.ID}, nil
}

func (s *ProductGRPCServer) UpdateCategory(ctx context.Context, req *productv1.UpdateCategoryRequest) (*productv1.UpdateCategoryResponse, error) {
	c := req.GetCategory()
	err := s.svc.UpdateCategory(ctx, domain.Category{
		ID:       c.GetId(),
		TenantID: c.GetTenantId(),
		ParentID: c.GetParentId(),
		Name:     c.GetName(),
		Level:    c.GetLevel(),
		Sort:     c.GetSort(),
		Icon:     c.GetIcon(),
		Status:   domain.CategoryStatus(c.GetStatus()),
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.UpdateCategoryResponse{}, nil
}

func (s *ProductGRPCServer) ListCategories(ctx context.Context, req *productv1.ListCategoriesRequest) (*productv1.ListCategoriesResponse, error) {
	categories, err := s.svc.ListCategories(ctx, req.GetTenantId())
	if err != nil {
		return nil, handleErr(err)
	}
	dtos := make([]*productv1.Category, 0, len(categories))
	for _, c := range categories {
		dtos = append(dtos, toCategoryDTO(c))
	}
	return &productv1.ListCategoriesResponse{Categories: dtos}, nil
}

func (s *ProductGRPCServer) DeleteCategory(ctx context.Context, req *productv1.DeleteCategoryRequest) (*productv1.DeleteCategoryResponse, error) {
	err := s.svc.DeleteCategory(ctx, req.GetId(), req.GetTenantId())
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.DeleteCategoryResponse{}, nil
}

// ==================== Brand ====================

func (s *ProductGRPCServer) CreateBrand(ctx context.Context, req *productv1.CreateBrandRequest) (*productv1.CreateBrandResponse, error) {
	b := req.GetBrand()
	brand, err := s.svc.CreateBrand(ctx, domain.Brand{
		TenantID: b.GetTenantId(),
		Name:     b.GetName(),
		Logo:     b.GetLogo(),
		Status:   domain.BrandStatus(b.GetStatus()),
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.CreateBrandResponse{Id: brand.ID}, nil
}

func (s *ProductGRPCServer) UpdateBrand(ctx context.Context, req *productv1.UpdateBrandRequest) (*productv1.UpdateBrandResponse, error) {
	b := req.GetBrand()
	err := s.svc.UpdateBrand(ctx, domain.Brand{
		ID:       b.GetId(),
		TenantID: b.GetTenantId(),
		Name:     b.GetName(),
		Logo:     b.GetLogo(),
		Status:   domain.BrandStatus(b.GetStatus()),
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.UpdateBrandResponse{}, nil
}

func (s *ProductGRPCServer) ListBrands(ctx context.Context, req *productv1.ListBrandsRequest) (*productv1.ListBrandsResponse, error) {
	brands, total, err := s.svc.ListBrands(ctx, req.GetTenantId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, handleErr(err)
	}
	dtos := make([]*productv1.Brand, 0, len(brands))
	for _, b := range brands {
		dtos = append(dtos, toBrandDTO(b))
	}
	return &productv1.ListBrandsResponse{Brands: dtos, Total: total}, nil
}

func (s *ProductGRPCServer) DeleteBrand(ctx context.Context, req *productv1.DeleteBrandRequest) (*productv1.DeleteBrandResponse, error) {
	err := s.svc.DeleteBrand(ctx, req.GetId(), req.GetTenantId())
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.DeleteBrandResponse{}, nil
}

// ==================== Sales ====================

func (s *ProductGRPCServer) IncrSales(ctx context.Context, req *productv1.IncrSalesRequest) (*productv1.IncrSalesResponse, error) {
	err := s.svc.IncrSales(ctx, req.GetProductId(), req.GetTenantId(), req.GetCount())
	if err != nil {
		return nil, handleErr(err)
	}
	return &productv1.IncrSalesResponse{}, nil
}

// ==================== DTO 转换 ====================

// toProductDTO converts a domain product to proto with full details (SKUs + Specs).
// Used by GetProduct.
func toProductDTO(p domain.Product) *productv1.Product {
	skus := make([]*productv1.ProductSKU, 0, len(p.SKUs))
	for _, sku := range p.SKUs {
		skus = append(skus, toSKUDTO(sku))
	}
	specs := make([]*productv1.ProductSpec, 0, len(p.Specs))
	for _, spec := range p.Specs {
		specs = append(specs, toSpecDTO(spec))
	}
	return &productv1.Product{
		Id:          p.ID,
		TenantId:    p.TenantID,
		CategoryId:  p.CategoryID,
		BrandId:     p.BrandID,
		Name:        p.Name,
		Subtitle:    p.Subtitle,
		MainImage:   p.MainImage,
		Images:      p.Images,
		Description: p.Description,
		Status:      int32(p.Status),
		Sales:       p.Sales,
		Skus:        skus,
		Specs:       specs,
		Ctime:       timestamppb.New(p.Ctime),
		Utime:       timestamppb.New(p.Utime),
	}
}

// toProductListDTO converts a domain product to proto without SKUs/Specs.
// Used by ListProducts.
func toProductListDTO(p domain.Product) *productv1.Product {
	return &productv1.Product{
		Id:          p.ID,
		TenantId:    p.TenantID,
		CategoryId:  p.CategoryID,
		BrandId:     p.BrandID,
		Name:        p.Name,
		Subtitle:    p.Subtitle,
		MainImage:   p.MainImage,
		Images:      p.Images,
		Description: p.Description,
		Status:      int32(p.Status),
		Sales:       p.Sales,
		Ctime:       timestamppb.New(p.Ctime),
		Utime:       timestamppb.New(p.Utime),
	}
}

// toBatchProductDTO converts a domain product to proto with SKUs but no Specs.
// Used by BatchGetProducts.
func toBatchProductDTO(p domain.Product) *productv1.Product {
	skus := make([]*productv1.ProductSKU, 0, len(p.SKUs))
	for _, sku := range p.SKUs {
		skus = append(skus, toSKUDTO(sku))
	}
	return &productv1.Product{
		Id:          p.ID,
		TenantId:    p.TenantID,
		CategoryId:  p.CategoryID,
		BrandId:     p.BrandID,
		Name:        p.Name,
		Subtitle:    p.Subtitle,
		MainImage:   p.MainImage,
		Images:      p.Images,
		Description: p.Description,
		Status:      int32(p.Status),
		Sales:       p.Sales,
		Skus:        skus,
		Ctime:       timestamppb.New(p.Ctime),
		Utime:       timestamppb.New(p.Utime),
	}
}

func toSKUDTO(sku domain.SKU) *productv1.ProductSKU {
	return &productv1.ProductSKU{
		Id:            sku.ID,
		TenantId:      sku.TenantID,
		ProductId:     sku.ProductID,
		SpecValues:    sku.SpecValues,
		Price:         sku.Price,
		OriginalPrice: sku.OriginalPrice,
		CostPrice:     sku.CostPrice,
		SkuCode:       sku.SKUCode,
		BarCode:       sku.BarCode,
		Status:        int32(sku.Status),
	}
}

func toSpecDTO(spec domain.ProductSpec) *productv1.ProductSpec {
	return &productv1.ProductSpec{
		Id:        spec.ID,
		ProductId: spec.ProductID,
		Name:      spec.Name,
		Values:    spec.Values,
		TenantId:  spec.TenantID,
	}
}

// toCategoryDTO recursively converts domain category tree to proto.
func toCategoryDTO(c domain.Category) *productv1.Category {
	children := make([]*productv1.Category, 0, len(c.Children))
	for _, child := range c.Children {
		children = append(children, toCategoryDTO(child))
	}
	return &productv1.Category{
		Id:       c.ID,
		TenantId: c.TenantID,
		ParentId: c.ParentID,
		Name:     c.Name,
		Level:    c.Level,
		Sort:     c.Sort,
		Icon:     c.Icon,
		Status:   int32(c.Status),
		Children: children,
	}
}

func toBrandDTO(b domain.Brand) *productv1.Brand {
	return &productv1.Brand{
		Id:       b.ID,
		TenantId: b.TenantID,
		Name:     b.Name,
		Logo:     b.Logo,
		Status:   int32(b.Status),
	}
}

func toDomainSKU(sku *productv1.ProductSKU) domain.SKU {
	return domain.SKU{
		ID:            sku.GetId(),
		TenantID:      sku.GetTenantId(),
		ProductID:     sku.GetProductId(),
		SpecValues:    sku.GetSpecValues(),
		Price:         sku.GetPrice(),
		OriginalPrice: sku.GetOriginalPrice(),
		CostPrice:     sku.GetCostPrice(),
		SKUCode:       sku.GetSkuCode(),
		BarCode:       sku.GetBarCode(),
		Status:        domain.SKUStatus(sku.GetStatus()),
	}
}

func toDomainSpec(spec *productv1.ProductSpec) domain.ProductSpec {
	return domain.ProductSpec{
		ID:        spec.GetId(),
		ProductID: spec.GetProductId(),
		TenantID:  spec.GetTenantId(),
		Name:      spec.GetName(),
		Values:    spec.GetValues(),
	}
}

// ==================== 错误处理 ====================

func handleErr(err error) error {
	switch {
	case errors.Is(err, service.ErrProductNotFound),
		errors.Is(err, service.ErrCategoryNotFound),
		errors.Is(err, service.ErrBrandNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrQuotaExceeded):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, service.ErrCategoryHasChild),
		errors.Is(err, service.ErrCategoryHasProduct),
		errors.Is(err, service.ErrBrandHasProduct),
		errors.Is(err, service.ErrCategoryLevelLimit):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Errorf(codes.Internal, "内部错误: %v", err)
	}
}
