package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/merchant-bff/domain"
	"github.com/rermrf/mall/pkg/ginx"
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

type ListOrdersReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

func (h *OrderHandler) ListOrders(ctx *gin.Context, req ListOrdersReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.orderClient.ListOrders(ctx.Request.Context(), &orderv1.ListOrdersRequest{
		TenantId: tenantId,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询订单列表失败", ginx.OrderErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"orders": resp.GetOrders(),
		"total":  resp.GetTotal(),
	}}, nil
}

func (h *OrderHandler) GetOrder(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	if orderNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的订单号"})
		return
	}
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

func (h *OrderHandler) ShipOrder(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	uid, err := ginx.GetUID(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeUnauthorized, Msg: "未授权"})
		return
	}
	_, err = h.orderClient.UpdateOrderStatus(ctx.Request.Context(), &orderv1.UpdateOrderStatusRequest{
		OrderNo:      orderNo,
		Status:       int32(domain.OrderStatusShipped),
		OperatorId:   uid,
		OperatorType: int32(domain.OperatorTypeMerchant),
		Remark:       "商家发货",
	})
	if err != nil {
		h.l.Error("发货失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.OrderErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

type HandleRefundReq struct {
	RefundNo string `json:"refundNo" binding:"required"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

func (h *OrderHandler) HandleRefund(ctx *gin.Context, req HandleRefundReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	_, err := h.orderClient.HandleRefund(ctx.Request.Context(), &orderv1.HandleRefundRequest{
		RefundNo: req.RefundNo,
		TenantId: tenantId,
		Approved: req.Approved,
		Reason:   req.Reason,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "处理退款失败", ginx.OrderErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListRefundsReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

func (h *OrderHandler) ListRefundOrders(ctx *gin.Context, req ListRefundsReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.orderClient.ListRefundOrders(ctx.Request.Context(), &orderv1.ListRefundOrdersRequest{
		TenantId: tenantId,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询退款列表失败", ginx.OrderErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"refundOrders": resp.GetRefundOrders(),
		"total":         resp.GetTotal(),
	}}, nil
}
