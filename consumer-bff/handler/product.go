package handler

import (
	"net/http"
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
	return &ProductHandler{productClient: productClient, l: l}
}

func (h *ProductHandler) GetProduct(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的商品ID"})
		return
	}
	resp, err := h.productClient.GetProduct(ctx.Request.Context(), &productv1.GetProductRequest{
		Id: id,
	})
	if err != nil {
		h.l.Error("查询商品详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetProduct()})
}

func (h *ProductHandler) ListCategories(ctx *gin.Context) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.productClient.ListCategories(ctx.Request.Context(), &productv1.ListCategoriesRequest{
		TenantId: tenantId.(int64),
	})
	if err != nil {
		h.l.Error("查询分类列表失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetCategories()})
}

type ListProductsReq struct {
	CategoryId int64 `form:"categoryId"`
	Page       int32 `form:"page" binding:"required,min=1"`
	PageSize   int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

func (h *ProductHandler) ListProducts(ctx *gin.Context, req ListProductsReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.productClient.ListProducts(ctx.Request.Context(), &productv1.ListProductsRequest{
		TenantId:   tenantId.(int64),
		CategoryId: req.CategoryId,
		Status:     2, // 只返回上架商品
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询商品列表失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"products": resp.GetProducts(),
		"total":    resp.GetTotal(),
	}}, nil
}
