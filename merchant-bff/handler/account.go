package handler

import (
	"net/http"

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

// GetAccount 获取商家账户信息
func (h *AccountHandler) GetAccount(ctx *gin.Context) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		ctx.JSON(http.StatusOK, *errResult)
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

// UpdateBankInfoReq 更新银行信息请求
type UpdateBankInfoReq struct {
	BankAccountName string `json:"bank_account_name" binding:"required"`
	BankAccountNo   string `json:"bank_account_no" binding:"required"`
	BankName        string `json:"bank_name" binding:"required"`
}

// UpdateBankInfo 更新银行信息
func (h *AccountHandler) UpdateBankInfo(ctx *gin.Context, req UpdateBankInfoReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	_, err := h.accountClient.UpdateBankInfo(ctx.Request.Context(), &accountv1.UpdateBankInfoRequest{
		TenantId:        tenantId,
		BankAccountName: req.BankAccountName,
		BankAccountNo:   req.BankAccountNo,
		BankName:        req.BankName,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新银行信息失败")
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

// ListSettlementsReq 结算列表请求
type ListSettlementsReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

// ListSettlements 查询结算记录
func (h *AccountHandler) ListSettlements(ctx *gin.Context, req ListSettlementsReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.accountClient.ListSettlements(ctx.Request.Context(), &accountv1.ListSettlementsRequest{
		TenantId: tenantId,
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

// RequestWithdrawalReq 提现申请请求
type RequestWithdrawalReq struct {
	Amount int64 `json:"amount" binding:"required,min=1"`
}

// RequestWithdrawal 申请提现
func (h *AccountHandler) RequestWithdrawal(ctx *gin.Context, req RequestWithdrawalReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.accountClient.RequestWithdrawal(ctx.Request.Context(), &accountv1.RequestWithdrawalRequest{
		TenantId: tenantId,
		Amount:   req.Amount,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "申请提现失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"withdrawalNo": resp.GetWithdrawalNo(),
	}}, nil
}

// ListWithdrawalsReq 提现列表请求
type ListWithdrawalsReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

// ListWithdrawals 查询提现记录
func (h *AccountHandler) ListWithdrawals(ctx *gin.Context, req ListWithdrawalsReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.accountClient.ListWithdrawals(ctx.Request.Context(), &accountv1.ListWithdrawalsRequest{
		TenantId: tenantId,
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

// ListTransactionsReq 流水列表请求
type ListTransactionsReq struct {
	Type     int32 `form:"type"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

// ListTransactions 查询交易流水
func (h *AccountHandler) ListTransactions(ctx *gin.Context, req ListTransactionsReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.accountClient.ListTransactions(ctx.Request.Context(), &accountv1.ListTransactionsRequest{
		TenantId: tenantId,
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

// GetAccountSummary 获取账户概览
func (h *AccountHandler) GetAccountSummary(ctx *gin.Context) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		ctx.JSON(http.StatusOK, *errResult)
		return
	}
	resp, err := h.accountClient.GetAccountSummary(ctx.Request.Context(), &accountv1.GetAccountSummaryRequest{
		TenantId: tenantId,
	})
	if err != nil {
		h.l.Error("查询账户概览失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetSummary()})
}
