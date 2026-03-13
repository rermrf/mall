package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	marketingv1 "github.com/rermrf/mall/api/proto/gen/marketing/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

type MarketingHandler struct {
	marketingClient marketingv1.MarketingServiceClient
	l               logger.Logger
}

func NewMarketingHandler(marketingClient marketingv1.MarketingServiceClient, l logger.Logger) *MarketingHandler {
	return &MarketingHandler{
		marketingClient: marketingClient,
		l:               l,
	}
}

// ==================== 优惠券监管 ====================

type AdminListCouponsReq struct {
	TenantId int64 `form:"tenant_id"`
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *MarketingHandler) ListCoupons(ctx *gin.Context, req AdminListCouponsReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.marketingClient.ListCoupons(rpcCtx, &marketingv1.ListCouponsRequest{
		TenantId: req.TenantId,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询优惠券列表失败", ginx.MarketingErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"coupons": resp.GetCoupons(),
		"total":   resp.GetTotal(),
	}}, nil
}

// ==================== 秒杀活动监管 ====================

type AdminListSeckillReq struct {
	TenantId int64 `form:"tenant_id"`
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *MarketingHandler) ListSeckill(ctx *gin.Context, req AdminListSeckillReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.marketingClient.ListSeckillActivities(rpcCtx, &marketingv1.ListSeckillActivitiesRequest{
		TenantId: req.TenantId,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询秒杀活动列表失败", ginx.MarketingErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"activities": resp.GetActivities(),
		"total":      resp.GetTotal(),
	}}, nil
}

func (h *MarketingHandler) GetSeckill(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resp, err := h.marketingClient.GetSeckillActivity(ctx.Request.Context(), &marketingv1.GetSeckillActivityRequest{Id: id})
	if err != nil {
		h.l.Error("查询秒杀活动详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.MarketingErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetActivity()})
}

// ==================== 满减规则监管 ====================

type AdminListPromotionsReq struct {
	TenantId int64 `form:"tenant_id"`
	Status   int32 `form:"status"`
}

func (h *MarketingHandler) ListPromotions(ctx *gin.Context, req AdminListPromotionsReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.marketingClient.ListPromotionRules(rpcCtx, &marketingv1.ListPromotionRulesRequest{
		TenantId: req.TenantId,
		Status:   req.Status,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询满减规则列表失败", ginx.MarketingErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetRules()}, nil
}
