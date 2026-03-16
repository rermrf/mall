package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/validatorx"
)

type ProductHandler struct {
	productClient productv1.ProductServiceClient
	l             logger.Logger
}

func NewProductHandler(productClient productv1.ProductServiceClient, l logger.Logger) *ProductHandler {
	return &ProductHandler{
		productClient: productClient,
		l:             l,
	}
}

// ==================== 商品 ====================

type CreateProductReq struct {
	CategoryId  int64            `json:"categoryId" binding:"required"`
	BrandId     int64            `json:"brandId"`
	Name        string           `json:"name" binding:"required"`
	Subtitle    string           `json:"subtitle"`
	MainImage   string           `json:"mainImage"`
	Images      string           `json:"images"`
	Description string           `json:"description"`
	Status      int32            `json:"status"`
	Skus        []ProductSKUReq  `json:"skus"`
	Specs       []ProductSpecReq `json:"specs"`
}

type ProductSKUReq struct {
	SkuCode       string `json:"skuCode"`
	Price         int64  `json:"price"`
	OriginalPrice int64  `json:"originalPrice"`
	CostPrice     int64  `json:"costPrice"`
	BarCode       string `json:"barCode"`
	SpecValues    string `json:"specValues"`
	Status        int32  `json:"status"`
}

type ProductSpecReq struct {
	Name   string `json:"name"`
	Values string `json:"values"`
}

func (h *ProductHandler) CreateProduct(ctx *gin.Context, req CreateProductReq) (ginx.Result, error) {
	v := validatorx.New()
	for i, s := range req.Skus {
		field := fmt.Sprintf("skus[%d].price", i)
		v.CheckPositive(field, s.Price)
	}
	if v.HasErrors() {
		return v.ToResult(), nil
	}
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	skus := make([]*productv1.ProductSKU, 0, len(req.Skus))
	for _, s := range req.Skus {
		skus = append(skus, &productv1.ProductSKU{
			SkuCode:       s.SkuCode,
			Price:         s.Price,
			OriginalPrice: s.OriginalPrice,
			CostPrice:     s.CostPrice,
			BarCode:       s.BarCode,
			SpecValues:    s.SpecValues,
			Status:        s.Status,
		})
	}
	specs := make([]*productv1.ProductSpec, 0, len(req.Specs))
	for _, s := range req.Specs {
		specs = append(specs, &productv1.ProductSpec{
			Name:   s.Name,
			Values: s.Values,
		})
	}
	resp, err := h.productClient.CreateProduct(ctx.Request.Context(), &productv1.CreateProductRequest{
		Product: &productv1.Product{
			TenantId:    tenantId,
			CategoryId:  req.CategoryId,
			BrandId:     req.BrandId,
			Name:        req.Name,
			Subtitle:    req.Subtitle,
			MainImage:   req.MainImage,
			Images:      req.Images,
			Description: req.Description,
			Status:      req.Status,
			Skus:        skus,
			Specs:       specs,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建商品失败", ginx.ProductErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateProductReq struct {
	CategoryId  int64            `json:"categoryId"`
	BrandId     int64            `json:"brandId"`
	Name        string           `json:"name"`
	Subtitle    string           `json:"subtitle"`
	MainImage   string           `json:"mainImage"`
	Images      string           `json:"images"`
	Description string           `json:"description"`
	Skus        []ProductSKUReq  `json:"skus"`
	Specs       []ProductSpecReq `json:"specs"`
}

func (h *ProductHandler) UpdateProduct(ctx *gin.Context, req UpdateProductReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的商品 ID"}, nil
	}
	v := validatorx.New()
	for i, s := range req.Skus {
		field := fmt.Sprintf("skus[%d].price", i)
		v.CheckPositive(field, s.Price)
	}
	if v.HasErrors() {
		return v.ToResult(), nil
	}
	skus := make([]*productv1.ProductSKU, 0, len(req.Skus))
	for _, s := range req.Skus {
		skus = append(skus, &productv1.ProductSKU{
			SkuCode:       s.SkuCode,
			Price:         s.Price,
			OriginalPrice: s.OriginalPrice,
			CostPrice:     s.CostPrice,
			BarCode:       s.BarCode,
			SpecValues:    s.SpecValues,
			Status:        s.Status,
		})
	}
	specs := make([]*productv1.ProductSpec, 0, len(req.Specs))
	for _, s := range req.Specs {
		specs = append(specs, &productv1.ProductSpec{
			Name:   s.Name,
			Values: s.Values,
		})
	}
	_, err = h.productClient.UpdateProduct(ctx.Request.Context(), &productv1.UpdateProductRequest{
		Product: &productv1.Product{
			Id:          id,
			TenantId:    tenantId,
			CategoryId:  req.CategoryId,
			BrandId:     req.BrandId,
			Name:        req.Name,
			Subtitle:    req.Subtitle,
			MainImage:   req.MainImage,
			Images:      req.Images,
			Description: req.Description,
			Skus:        skus,
			Specs:       specs,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新商品失败", ginx.ProductErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

func (h *ProductHandler) GetProduct(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的商品 ID"})
		return
	}
	resp, err := h.productClient.GetProduct(ctx.Request.Context(), &productv1.GetProductRequest{Id: id})
	if err != nil {
		h.l.Error("查询商品详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.ProductErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetProduct()})
}

type ListProductsReq struct {
	CategoryId int64 `form:"categoryId"`
	Status     int32 `form:"status"`
	Page       int32 `form:"page" binding:"required,min=1"`
	PageSize   int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

func (h *ProductHandler) ListProducts(ctx *gin.Context, req ListProductsReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.productClient.ListProducts(ctx.Request.Context(), &productv1.ListProductsRequest{
		TenantId:   tenantId,
		CategoryId: req.CategoryId,
		Status:     req.Status,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询商品列表失败", ginx.ProductErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"products": resp.GetProducts(),
		"total":    resp.GetTotal(),
	}}, nil
}

type UpdateProductStatusReq struct {
	Status int32 `json:"status" binding:"required"`
}

func (h *ProductHandler) UpdateProductStatus(ctx *gin.Context, req UpdateProductStatusReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的商品 ID"}, nil
	}
	_, err = h.productClient.UpdateProductStatus(ctx.Request.Context(), &productv1.UpdateProductStatusRequest{
		Id:       id,
		TenantId: tenantId,
		Status:   req.Status,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新商品状态失败", ginx.ProductErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

// ==================== 分类 ====================

type CreateCategoryReq struct {
	ParentId int64  `json:"parentId"`
	Name     string `json:"name" binding:"required"`
	Level    int32  `json:"level"`
	Sort     int32  `json:"sort"`
	Icon     string `json:"icon"`
	Status   int32  `json:"status"`
}

func (h *ProductHandler) CreateCategory(ctx *gin.Context, req CreateCategoryReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.productClient.CreateCategory(ctx.Request.Context(), &productv1.CreateCategoryRequest{
		Category: &productv1.Category{
			TenantId: tenantId,
			ParentId: req.ParentId,
			Name:     req.Name,
			Level:    req.Level,
			Sort:     req.Sort,
			Icon:     req.Icon,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建分类失败", ginx.ProductErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateCategoryReq struct {
	ParentId int64  `json:"parentId"`
	Name     string `json:"name"`
	Level    int32  `json:"level"`
	Sort     int32  `json:"sort"`
	Icon     string `json:"icon"`
	Status   int32  `json:"status"`
}

func (h *ProductHandler) UpdateCategory(ctx *gin.Context, req UpdateCategoryReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的分类 ID"}, nil
	}
	_, err = h.productClient.UpdateCategory(ctx.Request.Context(), &productv1.UpdateCategoryRequest{
		Category: &productv1.Category{
			Id:       id,
			TenantId: tenantId,
			ParentId: req.ParentId,
			Name:     req.Name,
			Level:    req.Level,
			Sort:     req.Sort,
			Icon:     req.Icon,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新分类失败", ginx.ProductErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListCategoriesReq struct{}

func (h *ProductHandler) ListCategories(ctx *gin.Context, _ ListCategoriesReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.productClient.ListCategories(ctx.Request.Context(), &productv1.ListCategoriesRequest{
		TenantId: tenantId,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询分类列表失败", ginx.ProductErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetCategories()}, nil
}

// ==================== 品牌 ====================

type CreateBrandReq struct {
	Name   string `json:"name" binding:"required"`
	Logo   string `json:"logo"`
	Status int32  `json:"status"`
}

func (h *ProductHandler) CreateBrand(ctx *gin.Context, req CreateBrandReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.productClient.CreateBrand(ctx.Request.Context(), &productv1.CreateBrandRequest{
		Brand: &productv1.Brand{
			TenantId: tenantId,
			Name:     req.Name,
			Logo:     req.Logo,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建品牌失败", ginx.ProductErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateBrandReq struct {
	Name   string `json:"name"`
	Logo   string `json:"logo"`
	Status int32  `json:"status"`
}

func (h *ProductHandler) UpdateBrand(ctx *gin.Context, req UpdateBrandReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的品牌 ID"}, nil
	}
	_, err = h.productClient.UpdateBrand(ctx.Request.Context(), &productv1.UpdateBrandRequest{
		Brand: &productv1.Brand{
			Id:       id,
			TenantId: tenantId,
			Name:     req.Name,
			Logo:     req.Logo,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新品牌失败", ginx.ProductErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListBrandsReq struct {
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

func (h *ProductHandler) ListBrands(ctx *gin.Context, req ListBrandsReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.productClient.ListBrands(ctx.Request.Context(), &productv1.ListBrandsRequest{
		TenantId: tenantId,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询品牌列表失败", ginx.ProductErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"brands": resp.GetBrands(),
		"total":  resp.GetTotal(),
	}}, nil
}
