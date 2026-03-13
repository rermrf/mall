package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
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
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetInventory()})
}

type AdminBatchGetStockReq struct {
	SkuIds []int64 `json:"sku_ids" binding:"required,min=1"`
}

func (h *InventoryHandler) BatchGetStock(ctx *gin.Context, req AdminBatchGetStockReq) (ginx.Result, error) {
	resp, err := h.inventoryClient.BatchGetStock(ctx.Request.Context(), &inventoryv1.BatchGetStockRequest{
		SkuIds: req.SkuIds,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("批量查询库存失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetInventories()}, nil
}

type AdminListLogsReq struct {
	TenantId int64 `form:"tenant_id"`
	SkuId    int64 `form:"sku_id"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *InventoryHandler) ListLogs(ctx *gin.Context, req AdminListLogsReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.inventoryClient.ListLogs(rpcCtx, &inventoryv1.ListLogsRequest{
		TenantId: req.TenantId,
		SkuId:    req.SkuId,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询库存日志失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"logs":  resp.GetLogs(),
		"total": resp.GetTotal(),
	}}, nil
}
