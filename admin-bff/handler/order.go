package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

type OrderHandler struct {
	orderClient orderv1.OrderServiceClient
	l           logger.Logger
}

func NewOrderHandler(orderClient orderv1.OrderServiceClient, l logger.Logger) *OrderHandler {
	return &OrderHandler{
		orderClient: orderClient,
		l:           l,
	}
}

type AdminListOrdersReq struct {
	TenantId int64 `form:"tenant_id"`
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *OrderHandler) ListOrders(ctx *gin.Context, req AdminListOrdersReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.orderClient.ListOrders(rpcCtx, &orderv1.ListOrdersRequest{
		TenantId: req.TenantId,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询全平台订单列表失败", ginx.OrderErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"orders": resp.GetOrders(),
		"total":  resp.GetTotal(),
	}}, nil
}

func (h *OrderHandler) GetOrder(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	resp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{
		OrderNo: orderNo,
	})
	if err != nil {
		h.l.Error("查询订单详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.OrderErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetOrder()})
}
