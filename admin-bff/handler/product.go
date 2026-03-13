package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/ginx"
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

// ==================== 平台分类管理 ====================

type AdminCreateCategoryReq struct {
	ParentId int64  `json:"parent_id"`
	Name     string `json:"name" binding:"required"`
	Level    int32  `json:"level"`
	Sort     int32  `json:"sort"`
	Icon     string `json:"icon"`
	Status   int32  `json:"status"`
}

func (h *ProductHandler) CreateCategory(ctx *gin.Context, req AdminCreateCategoryReq) (ginx.Result, error) {
	resp, err := h.productClient.CreateCategory(ctx.Request.Context(), &productv1.CreateCategoryRequest{
		Category: &productv1.Category{
			TenantId: 0,
			ParentId: req.ParentId,
			Name:     req.Name,
			Level:    req.Level,
			Sort:     req.Sort,
			Icon:     req.Icon,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建平台分类失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type AdminUpdateCategoryReq struct {
	ParentId int64  `json:"parent_id"`
	Name     string `json:"name"`
	Level    int32  `json:"level"`
	Sort     int32  `json:"sort"`
	Icon     string `json:"icon"`
	Status   int32  `json:"status"`
}

func (h *ProductHandler) UpdateCategory(ctx *gin.Context, req AdminUpdateCategoryReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.productClient.UpdateCategory(ctx.Request.Context(), &productv1.UpdateCategoryRequest{
		Category: &productv1.Category{
			Id:       id,
			TenantId: 0,
			ParentId: req.ParentId,
			Name:     req.Name,
			Level:    req.Level,
			Sort:     req.Sort,
			Icon:     req.Icon,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新平台分类失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type AdminListCategoriesReq struct{}

func (h *ProductHandler) ListCategories(ctx *gin.Context, _ AdminListCategoriesReq) (ginx.Result, error) {
	resp, err := h.productClient.ListCategories(ctx.Request.Context(), &productv1.ListCategoriesRequest{
		TenantId: 0,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询平台分类列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetCategories()}, nil
}

// ==================== 平台品牌管理 ====================

type AdminCreateBrandReq struct {
	Name   string `json:"name" binding:"required"`
	Logo   string `json:"logo"`
	Status int32  `json:"status"`
}

func (h *ProductHandler) CreateBrand(ctx *gin.Context, req AdminCreateBrandReq) (ginx.Result, error) {
	resp, err := h.productClient.CreateBrand(ctx.Request.Context(), &productv1.CreateBrandRequest{
		Brand: &productv1.Brand{
			TenantId: 0,
			Name:     req.Name,
			Logo:     req.Logo,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建平台品牌失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type AdminUpdateBrandReq struct {
	Name   string `json:"name"`
	Logo   string `json:"logo"`
	Status int32  `json:"status"`
}

func (h *ProductHandler) UpdateBrand(ctx *gin.Context, req AdminUpdateBrandReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.productClient.UpdateBrand(ctx.Request.Context(), &productv1.UpdateBrandRequest{
		Brand: &productv1.Brand{
			Id:       id,
			TenantId: 0,
			Name:     req.Name,
			Logo:     req.Logo,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新平台品牌失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type AdminListBrandsReq struct {
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *ProductHandler) ListBrands(ctx *gin.Context, req AdminListBrandsReq) (ginx.Result, error) {
	resp, err := h.productClient.ListBrands(ctx.Request.Context(), &productv1.ListBrandsRequest{
		TenantId: 0,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询平台品牌列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"brands": resp.GetBrands(),
		"total":  resp.GetTotal(),
	}}, nil
}
