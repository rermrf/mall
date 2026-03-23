package handler

import (
	"encoding/json"
	"io"
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

type CreatePaymentReq struct {
	OrderID int64  `json:"orderId" binding:"required"`
	OrderNo string `json:"orderNo" binding:"required"`
	Channel string `json:"channel" binding:"required,oneof=mock wechat alipay"`
	Amount  int64  `json:"amount" binding:"required,min=1"`
}

func (h *PaymentHandler) CreatePayment(ctx *gin.Context, req CreatePaymentReq) (ginx.Result, error) {
	tenantId, tidErr := ginx.GetTenantID(ctx)
	if tidErr != nil {
		return ginx.Result{Code: 401001, Msg: "未登录"}, nil
	}
	resp, err := h.paymentClient.CreatePayment(ctx.Request.Context(), &paymentv1.CreatePaymentRequest{
		TenantId: tenantId,
		OrderId:  req.OrderID,
		OrderNo:  req.OrderNo,
		Channel:  req.Channel,
		Amount:   req.Amount,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建支付单失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"paymentNo": resp.GetPaymentNo(),
		"payUrl":    resp.GetPayUrl(),
	}}, nil
}

func (h *PaymentHandler) GetPayment(ctx *gin.Context) {
	paymentNo := ctx.Param("paymentNo")
	if paymentNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的支付单号"})
		return
	}
	resp, err := h.paymentClient.GetPayment(ctx.Request.Context(), &paymentv1.GetPaymentRequest{
		PaymentNo: paymentNo,
	})
	if err != nil {
		h.l.Error("查询支付状态失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetPayment()})
}

type HandleNotifyReq struct {
	Channel    string `json:"channel" binding:"required"`
	NotifyBody string `json:"notifyBody" binding:"required"`
}

func (h *PaymentHandler) HandleNotify(ctx *gin.Context, req HandleNotifyReq) (ginx.Result, error) {
	resp, err := h.paymentClient.HandleNotify(ctx.Request.Context(), &paymentv1.HandleNotifyRequest{
		Channel:    req.Channel,
		NotifyBody: req.NotifyBody,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "处理支付回调失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"success": resp.GetSuccess(),
	}}, nil
}

func (h *PaymentHandler) AlipayNotify(ctx *gin.Context) {
	if err := ctx.Request.ParseForm(); err != nil {
		ctx.String(http.StatusOK, "FAIL")
		return
	}

	data := make(map[string]string)
	for k, v := range ctx.Request.Form {
		if len(v) > 0 {
			data[k] = v[0]
		}
	}
	bodyBytes, _ := json.Marshal(data)

	_, err := h.paymentClient.HandleNotify(ctx.Request.Context(), &paymentv1.HandleNotifyRequest{
		Channel:    "alipay",
		NotifyBody: string(bodyBytes),
	})
	if err != nil {
		h.l.Error("支付宝异步回调处理失败", logger.Error(err))
		ctx.String(http.StatusOK, "FAIL")
		return
	}
	ctx.String(http.StatusOK, "success")
}

func (h *PaymentHandler) WechatNotify(ctx *gin.Context) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		h.l.Error("读取微信支付回调请求体失败", logger.Error(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "读取请求体失败"})
		return
	}

	data := map[string]string{
		"body":                string(body),
		"Wechatpay-Timestamp": ctx.GetHeader("Wechatpay-Timestamp"),
		"Wechatpay-Nonce":     ctx.GetHeader("Wechatpay-Nonce"),
		"Wechatpay-Signature": ctx.GetHeader("Wechatpay-Signature"),
		"Wechatpay-Serial":    ctx.GetHeader("Wechatpay-Serial"),
	}
	bodyBytes, _ := json.Marshal(data)

	_, err = h.paymentClient.HandleNotify(ctx.Request.Context(), &paymentv1.HandleNotifyRequest{
		Channel:    "wechat",
		NotifyBody: string(bodyBytes),
	})
	if err != nil {
		h.l.Error("微信支付异步回调处理失败", logger.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "处理失败"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "成功"})
}
