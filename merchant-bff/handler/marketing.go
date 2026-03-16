package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	marketingv1 "github.com/rermrf/mall/api/proto/gen/marketing/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/validatorx"
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

// ==================== 优惠券 ====================

type CreateCouponReq struct {
	Name          string `json:"name" binding:"required"`
	Type          int32  `json:"type" binding:"required"`
	Threshold     int64  `json:"threshold"`
	DiscountValue int64  `json:"discount_value" binding:"required"`
	TotalCount    int32  `json:"total_count" binding:"required"`
	PerLimit      int32  `json:"per_limit"`
	StartTime     int64  `json:"start_time" binding:"required"`
	EndTime       int64  `json:"end_time" binding:"required"`
	ScopeType     int32  `json:"scope_type"`
	ScopeIDs      string `json:"scope_ids"`
	Status        int32  `json:"status"`
}

func (h *MarketingHandler) CreateCoupon(ctx *gin.Context, req CreateCouponReq) (ginx.Result, error) {
	v := validatorx.New()
	v.CheckPositive("discount_value", req.DiscountValue)
	v.CheckTimeRange("start_time", "end_time", req.StartTime, req.EndTime)
	if v.HasErrors() {
		return v.ToResult(), nil
	}
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.marketingClient.CreateCoupon(ctx.Request.Context(), &marketingv1.CreateCouponRequest{
		Coupon: &marketingv1.Coupon{
			TenantId:      tenantId,
			Name:          req.Name,
			Type:          req.Type,
			Threshold:     req.Threshold,
			DiscountValue: req.DiscountValue,
			TotalCount:    req.TotalCount,
			PerLimit:      req.PerLimit,
			StartTime:     req.StartTime,
			EndTime:       req.EndTime,
			ScopeType:     req.ScopeType,
			ScopeIds:      req.ScopeIDs,
			Status:        req.Status,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建优惠券失败", ginx.MarketingErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateCouponReq struct {
	Name          string `json:"name"`
	Type          int32  `json:"type"`
	Threshold     int64  `json:"threshold"`
	DiscountValue int64  `json:"discount_value"`
	TotalCount    int32  `json:"total_count"`
	PerLimit      int32  `json:"per_limit"`
	StartTime     int64  `json:"start_time"`
	EndTime       int64  `json:"end_time"`
	ScopeType     int32  `json:"scope_type"`
	ScopeIDs      string `json:"scope_ids"`
	Status        int32  `json:"status"`
}

func (h *MarketingHandler) UpdateCoupon(ctx *gin.Context, req UpdateCouponReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的优惠券 ID"}, nil
	}
	v := validatorx.New()
	if req.DiscountValue != 0 {
		v.CheckPositive("discount_value", req.DiscountValue)
	}
	if req.StartTime != 0 && req.EndTime != 0 {
		v.CheckTimeRange("start_time", "end_time", req.StartTime, req.EndTime)
	}
	if v.HasErrors() {
		return v.ToResult(), nil
	}
	_, err = h.marketingClient.UpdateCoupon(ctx.Request.Context(), &marketingv1.UpdateCouponRequest{
		Coupon: &marketingv1.Coupon{
			Id:            id,
			TenantId:      tenantId,
			Name:          req.Name,
			Type:          req.Type,
			Threshold:     req.Threshold,
			DiscountValue: req.DiscountValue,
			TotalCount:    req.TotalCount,
			PerLimit:      req.PerLimit,
			StartTime:     req.StartTime,
			EndTime:       req.EndTime,
			ScopeType:     req.ScopeType,
			ScopeIds:      req.ScopeIDs,
			Status:        req.Status,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新优惠券失败", ginx.MarketingErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListCouponsReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *MarketingHandler) ListCoupons(ctx *gin.Context, req ListCouponsReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.marketingClient.ListCoupons(ctx.Request.Context(), &marketingv1.ListCouponsRequest{
		TenantId: tenantId,
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

func (h *MarketingHandler) GetCoupon(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的优惠券 ID"})
		return
	}
	resp, err := h.marketingClient.GetCoupon(ctx.Request.Context(), &marketingv1.GetCouponRequest{Id: id})
	if err != nil {
		h.l.Error("查询优惠券详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.MarketingErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetCoupon()})
}

// ==================== 秒杀 ====================

type CreateSeckillReq struct {
	Name      string              `json:"name" binding:"required"`
	StartTime int64               `json:"start_time" binding:"required"`
	EndTime   int64               `json:"end_time" binding:"required"`
	Status    int32               `json:"status"`
	Items     []CreateSeckillItem `json:"items" binding:"required,min=1"`
}

type CreateSeckillItem struct {
	SkuID        int64 `json:"sku_id" binding:"required"`
	SeckillPrice int64 `json:"seckill_price" binding:"required"`
	SeckillStock int32 `json:"seckill_stock" binding:"required"`
	PerLimit     int32 `json:"per_limit"`
}

func (h *MarketingHandler) CreateSeckill(ctx *gin.Context, req CreateSeckillReq) (ginx.Result, error) {
	v := validatorx.New()
	v.CheckTimeRange("start_time", "end_time", req.StartTime, req.EndTime)
	for i, item := range req.Items {
		v.CheckPositive(fmt.Sprintf("items[%d].seckill_price", i), item.SeckillPrice)
		v.CheckPositiveInt32(fmt.Sprintf("items[%d].seckill_stock", i), item.SeckillStock)
	}
	if v.HasErrors() {
		return v.ToResult(), nil
	}
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	items := make([]*marketingv1.SeckillItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, &marketingv1.SeckillItem{
			SkuId:        item.SkuID,
			SeckillPrice: item.SeckillPrice,
			SeckillStock: item.SeckillStock,
			PerLimit:     item.PerLimit,
		})
	}
	resp, err := h.marketingClient.CreateSeckillActivity(ctx.Request.Context(), &marketingv1.CreateSeckillActivityRequest{
		Activity: &marketingv1.SeckillActivity{
			TenantId:  tenantId,
			Name:      req.Name,
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
			Status:    req.Status,
			Items:     items,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建秒杀活动失败", ginx.MarketingErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateSeckillReq struct {
	Name      string              `json:"name"`
	StartTime int64               `json:"start_time"`
	EndTime   int64               `json:"end_time"`
	Status    int32               `json:"status"`
	Items     []CreateSeckillItem `json:"items"`
}

func (h *MarketingHandler) UpdateSeckill(ctx *gin.Context, req UpdateSeckillReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的秒杀活动 ID"}, nil
	}
	v := validatorx.New()
	if req.StartTime != 0 && req.EndTime != 0 {
		v.CheckTimeRange("start_time", "end_time", req.StartTime, req.EndTime)
	}
	for i, item := range req.Items {
		if item.SeckillPrice != 0 {
			v.CheckPositive(fmt.Sprintf("items[%d].seckill_price", i), item.SeckillPrice)
		}
		if item.SeckillStock != 0 {
			v.CheckPositiveInt32(fmt.Sprintf("items[%d].seckill_stock", i), item.SeckillStock)
		}
	}
	if v.HasErrors() {
		return v.ToResult(), nil
	}
	items := make([]*marketingv1.SeckillItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, &marketingv1.SeckillItem{
			SkuId:        item.SkuID,
			SeckillPrice: item.SeckillPrice,
			SeckillStock: item.SeckillStock,
			PerLimit:     item.PerLimit,
		})
	}
	_, err = h.marketingClient.UpdateSeckillActivity(ctx.Request.Context(), &marketingv1.UpdateSeckillActivityRequest{
		Activity: &marketingv1.SeckillActivity{
			Id:        id,
			TenantId:  tenantId,
			Name:      req.Name,
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
			Status:    req.Status,
			Items:     items,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新秒杀活动失败", ginx.MarketingErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListSeckillReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *MarketingHandler) ListSeckill(ctx *gin.Context, req ListSeckillReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.marketingClient.ListSeckillActivities(ctx.Request.Context(), &marketingv1.ListSeckillActivitiesRequest{
		TenantId: tenantId,
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
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的秒杀活动 ID"})
		return
	}
	resp, err := h.marketingClient.GetSeckillActivity(ctx.Request.Context(), &marketingv1.GetSeckillActivityRequest{Id: id})
	if err != nil {
		h.l.Error("查询秒杀活动详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.MarketingErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetActivity()})
}

// ==================== 满减 ====================

type CreatePromotionReq struct {
	Name          string `json:"name" binding:"required"`
	Type          int32  `json:"type" binding:"required"`
	Threshold     int64  `json:"threshold" binding:"required"`
	DiscountValue int64  `json:"discount_value" binding:"required"`
	StartTime     int64  `json:"start_time" binding:"required"`
	EndTime       int64  `json:"end_time" binding:"required"`
	Status        int32  `json:"status"`
}

func (h *MarketingHandler) CreatePromotion(ctx *gin.Context, req CreatePromotionReq) (ginx.Result, error) {
	v := validatorx.New()
	v.CheckPositive("discount_value", req.DiscountValue)
	v.CheckTimeRange("start_time", "end_time", req.StartTime, req.EndTime)
	if v.HasErrors() {
		return v.ToResult(), nil
	}
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.marketingClient.CreatePromotionRule(ctx.Request.Context(), &marketingv1.CreatePromotionRuleRequest{
		Rule: &marketingv1.PromotionRule{
			TenantId:      tenantId,
			Name:          req.Name,
			Type:          req.Type,
			Threshold:     req.Threshold,
			DiscountValue: req.DiscountValue,
			StartTime:     req.StartTime,
			EndTime:       req.EndTime,
			Status:        req.Status,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建满减规则失败", ginx.MarketingErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdatePromotionReq struct {
	Name          string `json:"name"`
	Type          int32  `json:"type"`
	Threshold     int64  `json:"threshold"`
	DiscountValue int64  `json:"discount_value"`
	StartTime     int64  `json:"start_time"`
	EndTime       int64  `json:"end_time"`
	Status        int32  `json:"status"`
}

func (h *MarketingHandler) UpdatePromotion(ctx *gin.Context, req UpdatePromotionReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的满减规则 ID"}, nil
	}
	v := validatorx.New()
	if req.DiscountValue != 0 {
		v.CheckPositive("discount_value", req.DiscountValue)
	}
	if req.StartTime != 0 && req.EndTime != 0 {
		v.CheckTimeRange("start_time", "end_time", req.StartTime, req.EndTime)
	}
	if v.HasErrors() {
		return v.ToResult(), nil
	}
	_, err = h.marketingClient.UpdatePromotionRule(ctx.Request.Context(), &marketingv1.UpdatePromotionRuleRequest{
		Rule: &marketingv1.PromotionRule{
			Id:            id,
			TenantId:      tenantId,
			Name:          req.Name,
			Type:          req.Type,
			Threshold:     req.Threshold,
			DiscountValue: req.DiscountValue,
			StartTime:     req.StartTime,
			EndTime:       req.EndTime,
			Status:        req.Status,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新满减规则失败", ginx.MarketingErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListPromotionsReq struct {
	Status int32 `form:"status"`
}

func (h *MarketingHandler) ListPromotions(ctx *gin.Context, req ListPromotionsReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.marketingClient.ListPromotionRules(ctx.Request.Context(), &marketingv1.ListPromotionRulesRequest{
		TenantId: tenantId,
		Status:   req.Status,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询满减规则列表失败", ginx.MarketingErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetRules()}, nil
}
