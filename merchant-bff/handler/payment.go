package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type PaymentHandler struct {
	paymentClient paymentv1.PaymentServiceClient
	l             logger.Logger
}

func NewPaymentHandler(paymentClient paymentv1.PaymentServiceClient, l logger.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentClient: paymentClient,
		l:             l,
	}
}

type ListPaymentsReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

func (h *PaymentHandler) ListPayments(ctx *gin.Context, req ListPaymentsReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.paymentClient.ListPayments(ctx.Request.Context(), &paymentv1.ListPaymentsRequest{
		TenantId: tenantId,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询支付列表失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"payments": resp.GetPayments(),
		"total":    resp.GetTotal(),
	}}, nil
}

func (h *PaymentHandler) GetPayment(ctx *gin.Context) {
	paymentNo := ctx.Param("paymentNo")
	if paymentNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的支付单号"})
		return
	}
	tenantId, tidErr := ginx.GetTenantID(ctx)
	if tidErr != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeForbidden, Msg: "需要商家身份"})
		return
	}
	resp, err := h.paymentClient.GetPayment(ctx.Request.Context(), &paymentv1.GetPaymentRequest{
		PaymentNo: paymentNo,
	})
	if err != nil {
		h.l.Error("查询支付详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	payment := resp.GetPayment()
	if payment.GetTenantId() != tenantId {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeForbidden, Msg: "无权查看此支付单"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: payment})
}

type RefundReq struct {
	Amount int64  `json:"amount" binding:"required,min=1"`
	Reason string `json:"reason" binding:"required"`
}

func (h *PaymentHandler) Refund(ctx *gin.Context, req RefundReq) (ginx.Result, error) {
	paymentNo := ctx.Param("paymentNo")
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	// Verify payment belongs to this tenant before refunding
	getResp, err := h.paymentClient.GetPayment(ctx.Request.Context(), &paymentv1.GetPaymentRequest{
		PaymentNo: paymentNo,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询支付单失败")
	}
	if getResp.GetPayment().GetTenantId() != tenantId {
		return ginx.Result{Code: ginx.CodeForbidden, Msg: "无权操作此支付单"}, nil
	}
	resp, err := h.paymentClient.Refund(ctx.Request.Context(), &paymentv1.RefundRequest{
		PaymentNo: paymentNo,
		Amount:    req.Amount,
		Reason:    req.Reason,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "发起退款失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"refundNo": resp.GetRefundNo(),
	}}, nil
}

func (h *PaymentHandler) GetRefund(ctx *gin.Context) {
	refundNo := ctx.Param("refundNo")
	if refundNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的退款单号"})
		return
	}
	tenantId, tidErr := ginx.GetTenantID(ctx)
	if tidErr != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeForbidden, Msg: "需要商家身份"})
		return
	}
	resp, err := h.paymentClient.GetRefund(ctx.Request.Context(), &paymentv1.GetRefundRequest{
		RefundNo: refundNo,
	})
	if err != nil {
		h.l.Error("查询退款详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	refund := resp.GetRefund()
	if refund.GetTenantId() != tenantId {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeForbidden, Msg: "无权查看此退款单"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: refund})
}
