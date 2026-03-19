package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	accountv1 "github.com/rermrf/mall/api/proto/gen/account/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type AccountHandler struct {
	accountClient accountv1.AccountServiceClient
	l             logger.Logger
}

func NewAccountHandler(accountClient accountv1.AccountServiceClient, l logger.Logger) *AccountHandler {
	return &AccountHandler{
		accountClient: accountClient,
		l:             l,
	}
}

// AdminListAccountsReq 账户列表请求
type AdminListAccountsReq struct {
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

// ListAccounts 查询所有商家账户
func (h *AccountHandler) ListAccounts(ctx *gin.Context, req AdminListAccountsReq) (ginx.Result, error) {
	resp, err := h.accountClient.ListAccounts(ctx.Request.Context(), &accountv1.ListAccountsRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询账户列表失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"accounts": resp.GetAccounts(),
		"total":    resp.GetTotal(),
	}}, nil
}

// GetAccount 查询指定商家账户
func (h *AccountHandler) GetAccount(ctx *gin.Context) {
	tenantIdStr := ctx.Param("tenantId")
	tenantId, err := strconv.ParseInt(tenantIdStr, 10, 64)
	if err != nil || tenantId <= 0 {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的商家ID"})
		return
	}
	resp, err := h.accountClient.GetAccount(ctx.Request.Context(), &accountv1.GetAccountRequest{
		TenantId: tenantId,
	})
	if err != nil {
		h.l.Error("查询商家账户失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetAccount()})
}

// AdminListSettlementsReq 管理端结算列表请求
type AdminListSettlementsReq struct {
	TenantId int64 `form:"tenantId"`
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

// ListSettlements 查询结算记录
func (h *AccountHandler) ListSettlements(ctx *gin.Context, req AdminListSettlementsReq) (ginx.Result, error) {
	resp, err := h.accountClient.ListSettlements(ctx.Request.Context(), &accountv1.ListSettlementsRequest{
		TenantId: req.TenantId,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询结算列表失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"settlements": resp.GetSettlements(),
		"total":       resp.GetTotal(),
	}}, nil
}

// AdminListWithdrawalsReq 管理端提现列表请求
type AdminListWithdrawalsReq struct {
	TenantId int64 `form:"tenantId"`
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

// ListWithdrawals 查询提现记录
func (h *AccountHandler) ListWithdrawals(ctx *gin.Context, req AdminListWithdrawalsReq) (ginx.Result, error) {
	resp, err := h.accountClient.ListWithdrawals(ctx.Request.Context(), &accountv1.ListWithdrawalsRequest{
		TenantId: req.TenantId,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询提现列表失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"withdrawals": resp.GetWithdrawals(),
		"total":       resp.GetTotal(),
	}}, nil
}

// AdminReviewWithdrawalReq 审核提现请求
type AdminReviewWithdrawalReq struct {
	Approved     bool   `json:"approved"`
	RejectReason string `json:"reject_reason"`
}

// ReviewWithdrawal 审核提现申请
func (h *AccountHandler) ReviewWithdrawal(ctx *gin.Context, req AdminReviewWithdrawalReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的提现ID"}, nil
	}
	_, err = h.accountClient.ReviewWithdrawal(ctx.Request.Context(), &accountv1.ReviewWithdrawalRequest{
		Id:           id,
		Approved:     req.Approved,
		RejectReason: req.RejectReason,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "审核提现失败")
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

// ConfirmWithdrawalPaid 确认提现已打款
func (h *AccountHandler) ConfirmWithdrawalPaid(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的提现ID"})
		return
	}
	_, err = h.accountClient.ConfirmWithdrawalPaid(ctx.Request.Context(), &accountv1.ConfirmWithdrawalPaidRequest{
		Id: id,
	})
	if err != nil {
		h.l.Error("确认提现打款失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

// AdminListTransactionsReq 管理端流水列表请求
type AdminListTransactionsReq struct {
	TenantId int64 `form:"tenantId"`
	Type     int32 `form:"type"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

// ListTransactions 查询交易流水
func (h *AccountHandler) ListTransactions(ctx *gin.Context, req AdminListTransactionsReq) (ginx.Result, error) {
	resp, err := h.accountClient.ListTransactions(ctx.Request.Context(), &accountv1.ListTransactionsRequest{
		TenantId: req.TenantId,
		Type:     req.Type,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询交易流水失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"transactions": resp.GetTransactions(),
		"total":        resp.GetTotal(),
	}}, nil
}
