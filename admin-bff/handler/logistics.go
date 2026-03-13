package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	logisticsv1 "github.com/rermrf/mall/api/proto/gen/logistics/v1"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

type LogisticsHandler struct {
	logisticsClient logisticsv1.LogisticsServiceClient
	orderClient     orderv1.OrderServiceClient
	l               logger.Logger
}

func NewLogisticsHandler(
	logisticsClient logisticsv1.LogisticsServiceClient,
	orderClient orderv1.OrderServiceClient,
	l logger.Logger,
) *LogisticsHandler {
	return &LogisticsHandler{
		logisticsClient: logisticsClient,
		orderClient:     orderClient,
		l:               l,
	}
}

// ==================== 运费模板监管 ====================

type AdminListFreightTemplatesReq struct {
	TenantId int64 `form:"tenant_id"`
}

func (h *LogisticsHandler) ListFreightTemplates(ctx *gin.Context, req AdminListFreightTemplatesReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.logisticsClient.ListFreightTemplates(rpcCtx, &logisticsv1.ListFreightTemplatesRequest{
		TenantId: req.TenantId,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询运费模板列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetTemplates()}, nil
}

func (h *LogisticsHandler) GetFreightTemplate(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resp, err := h.logisticsClient.GetFreightTemplate(ctx.Request.Context(), &logisticsv1.GetFreightTemplateRequest{Id: id})
	if err != nil {
		h.l.Error("查询运费模板详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetTemplate()})
}

// ==================== 物流查询 ====================

func (h *LogisticsHandler) GetShipment(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resp, err := h.logisticsClient.GetShipment(ctx.Request.Context(), &logisticsv1.GetShipmentRequest{Id: id})
	if err != nil {
		h.l.Error("查询物流单失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetShipment()})
}

func (h *LogisticsHandler) GetOrderLogistics(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	orderResp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{OrderNo: orderNo})
	if err != nil {
		h.l.Error("查询订单失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	resp, err := h.logisticsClient.GetShipmentByOrder(ctx.Request.Context(), &logisticsv1.GetShipmentByOrderRequest{
		OrderId: orderResp.GetOrder().GetId(),
	})
	if err != nil {
		h.l.Error("查询物流信息失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetShipment()})
}
