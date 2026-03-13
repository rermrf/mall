package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type InventoryHandler struct {
	inventoryClient inventoryv1.InventoryServiceClient
	l               logger.Logger
}

func NewInventoryHandler(inventoryClient inventoryv1.InventoryServiceClient, l logger.Logger) *InventoryHandler {
	return &InventoryHandler{
		inventoryClient: inventoryClient,
		l:               l,
	}
}

type SetStockReq struct {
	SkuId          int64 `json:"sku_id" binding:"required"`
	Total          int32 `json:"total" binding:"required,min=0"`
	AlertThreshold int32 `json:"alert_threshold" binding:"min=0"`
}

func (h *InventoryHandler) SetStock(ctx *gin.Context, req SetStockReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	_, err := h.inventoryClient.SetStock(ctx.Request.Context(), &inventoryv1.SetStockRequest{
		TenantId:       tenantId.(int64),
		SkuId:          req.SkuId,
		Total:          req.Total,
		AlertThreshold: req.AlertThreshold,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "设置库存失败")
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

func (h *InventoryHandler) GetStock(ctx *gin.Context) {
	skuIdStr := ctx.Param("skuId")
	skuId, err := strconv.ParseInt(skuIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的 skuId"})
		return
	}
	resp, err := h.inventoryClient.GetStock(ctx.Request.Context(), &inventoryv1.GetStockRequest{
		SkuId: skuId,
	})
	if err != nil {
		h.l.Error("查询库存失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetInventory()})
}

type BatchGetStockReq struct {
	SkuIds []int64 `json:"sku_ids" binding:"required,min=1"`
}

func (h *InventoryHandler) BatchGetStock(ctx *gin.Context, req BatchGetStockReq) (ginx.Result, error) {
	resp, err := h.inventoryClient.BatchGetStock(ctx.Request.Context(), &inventoryv1.BatchGetStockRequest{
		SkuIds: req.SkuIds,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "批量查询库存失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetInventories()}, nil
}

type ListLogsReq struct {
	SkuId    int64 `form:"sku_id"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *InventoryHandler) ListLogs(ctx *gin.Context, req ListLogsReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.inventoryClient.ListLogs(ctx.Request.Context(), &inventoryv1.ListLogsRequest{
		TenantId: tenantId.(int64),
		SkuId:    req.SkuId,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询库存日志失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"logs":  resp.GetLogs(),
		"total": resp.GetTotal(),
	}}, nil
}
