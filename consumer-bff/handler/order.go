package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
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

type CreateOrderReq struct {
	Items     []CreateOrderItem `json:"items" binding:"required,min=1"`
	AddressID int64             `json:"addressId" binding:"required"`
	CouponID  int64             `json:"couponId"`
	Remark    string            `json:"remark"`
}

type CreateOrderItem struct {
	SkuID    int64 `json:"skuId" binding:"required"`
	Quantity int32 `json:"quantity" binding:"required,min=1"`
}

func (h *OrderHandler) CreateOrder(ctx *gin.Context, req CreateOrderReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	tenantId, _ := ctx.Get("tenant_id")
	items := make([]*orderv1.CreateOrderItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, &orderv1.CreateOrderItem{
			SkuId:    item.SkuID,
			Quantity: item.Quantity,
		})
	}
	resp, err := h.orderClient.CreateOrder(ctx.Request.Context(), &orderv1.CreateOrderRequest{
		BuyerId:   uid.(int64),
		TenantId:  tenantId.(int64),
		Items:     items,
		AddressId: req.AddressID,
		CouponId:  req.CouponID,
		Remark:    req.Remark,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建订单失败", ginx.OrderErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"orderNo":   resp.GetOrderNo(),
		"payAmount": resp.GetPayAmount(),
	}}, nil
}

type ListOrdersReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

func (h *OrderHandler) ListOrders(ctx *gin.Context, req ListOrdersReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.orderClient.ListOrders(ctx.Request.Context(), &orderv1.ListOrdersRequest{
		BuyerId:  uid.(int64),
		TenantId: tenantId.(int64),
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
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的订单号"})
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

func (h *OrderHandler) CancelOrder(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	uid, _ := ctx.Get("uid")
	_, err := h.orderClient.CancelOrder(ctx.Request.Context(), &orderv1.CancelOrderRequest{
		OrderNo: orderNo,
		BuyerId: uid.(int64),
	})
	if err != nil {
		h.l.Error("取消订单失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.OrderErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

func (h *OrderHandler) ConfirmReceive(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	uid, _ := ctx.Get("uid")
	_, err := h.orderClient.ConfirmReceive(ctx.Request.Context(), &orderv1.ConfirmReceiveRequest{
		OrderNo: orderNo,
		BuyerId: uid.(int64),
	})
	if err != nil {
		h.l.Error("确认收货失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.OrderErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

type ApplyRefundReq struct {
	Type         int32  `json:"type" binding:"required,oneof=1 2"`
	RefundAmount int64  `json:"refundAmount" binding:"required,min=1"`
	Reason       string `json:"reason" binding:"required"`
}

func (h *OrderHandler) ApplyRefund(ctx *gin.Context, req ApplyRefundReq) (ginx.Result, error) {
	orderNo := ctx.Param("orderNo")
	uid, _ := ctx.Get("uid")
	resp, err := h.orderClient.ApplyRefund(ctx.Request.Context(), &orderv1.ApplyRefundRequest{
		OrderNo:      orderNo,
		BuyerId:      uid.(int64),
		Type:         req.Type,
		RefundAmount: req.RefundAmount,
		Reason:       req.Reason,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "申请退款失败", ginx.OrderErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"refundNo": resp.GetRefundNo(),
	}}, nil
}

type ListRefundsReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

func (h *OrderHandler) ListRefundOrders(ctx *gin.Context, req ListRefundsReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.orderClient.ListRefundOrders(ctx.Request.Context(), &orderv1.ListRefundOrdersRequest{
		TenantId: tenantId.(int64),
		BuyerId:  uid.(int64),
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

func (h *OrderHandler) GetRefundOrder(ctx *gin.Context) {
	refundNo := ctx.Param("refundNo")
	if refundNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的退款单号"})
		return
	}
	resp, err := h.orderClient.GetRefundOrder(ctx.Request.Context(), &orderv1.GetRefundOrderRequest{
		RefundNo: refundNo,
	})
	if err != nil {
		h.l.Error("查询退款详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.OrderErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetRefundOrder()})
}
