package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type ReconciliationHandler struct {
	paymentClient paymentv1.PaymentServiceClient
	l             logger.Logger
}

func NewReconciliationHandler(paymentClient paymentv1.PaymentServiceClient, l logger.Logger) *ReconciliationHandler {
	return &ReconciliationHandler{
		paymentClient: paymentClient,
		l:             l,
	}
}

// RunReconciliationReq 触发对账请求
type RunReconciliationReq struct {
	Channel  string `json:"channel" binding:"required"`
	BillDate string `json:"bill_date" binding:"required"`
}

// RunReconciliation 手动触发对账
func (h *ReconciliationHandler) RunReconciliation(ctx *gin.Context, req RunReconciliationReq) (ginx.Result, error) {
	if _, err := time.Parse("2006-01-02", req.BillDate); err != nil {
		return ginx.Result{Code: 4, Msg: "日期格式错误，应为 YYYY-MM-DD"}, nil
	}
	resp, err := h.paymentClient.RunReconciliation(ctx.Request.Context(), &paymentv1.RunReconciliationRequest{
		Channel:  req.Channel,
		BillDate: req.BillDate,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "触发对账失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"batch_id": resp.GetBatchId(),
	}}, nil
}

// ListBatchesReq 对账批次列表请求
type ListBatchesReq struct {
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

// ListBatches 查询对账批次列表
func (h *ReconciliationHandler) ListBatches(ctx *gin.Context, req ListBatchesReq) (ginx.Result, error) {
	resp, err := h.paymentClient.ListReconciliationBatches(ctx.Request.Context(), &paymentv1.ListReconciliationBatchesRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询对账批次列表失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"batches": resp.GetBatches(),
		"total":   resp.GetTotal(),
	}}, nil
}

// GetBatchDetail 查询对账批次详情（含差异明细）
func (h *ReconciliationHandler) GetBatchDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	batchId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || batchId <= 0 {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的批次ID"})
		return
	}

	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 32)
	pageSize, _ := strconv.ParseInt(ctx.DefaultQuery("pageSize", "20"), 10, 32)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	resp, err := h.paymentClient.GetReconciliationBatchDetail(ctx.Request.Context(), &paymentv1.GetReconciliationBatchDetailRequest{
		BatchId:  batchId,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		h.l.Error("查询对账批次详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"batch":   resp.GetBatch(),
		"details": resp.GetDetails(),
		"total":   resp.GetTotal(),
	}})
}
