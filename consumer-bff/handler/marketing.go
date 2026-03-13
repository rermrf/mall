package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	marketingv1 "github.com/rermrf/mall/api/proto/gen/marketing/v1"
	"github.com/rermrf/mall/pkg/ginx"
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

// ListAvailableCoupons 可领优惠券列表（公开）
func (h *MarketingHandler) ListAvailableCoupons(ctx *gin.Context) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.marketingClient.ListCoupons(ctx.Request.Context(), &marketingv1.ListCouponsRequest{
		TenantId: tenantId.(int64),
		Status:   2, // 进行中
		Page:     1,
		PageSize: 50,
	})
	if err != nil {
		h.l.Error("查询可领优惠券列表失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetCoupons()})
}

// ReceiveCoupon 领券（需登录）
func (h *MarketingHandler) ReceiveCoupon(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	idStr := ctx.Param("id")
	couponId, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.marketingClient.ReceiveCoupon(ctx.Request.Context(), &marketingv1.ReceiveCouponRequest{
		UserId:   uid.(int64),
		CouponId: couponId,
	})
	if err != nil {
		h.l.Error("领取优惠券失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

// ListMyCoupons 我的优惠券（需登录）
func (h *MarketingHandler) ListMyCoupons(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	tenantId, _ := ctx.Get("tenant_id")
	statusStr := ctx.DefaultQuery("status", "0")
	status, _ := strconv.ParseInt(statusStr, 10, 32)
	resp, err := h.marketingClient.ListUserCoupons(ctx.Request.Context(), &marketingv1.ListUserCouponsRequest{
		UserId:   uid.(int64),
		TenantId: tenantId.(int64),
		Status:   int32(status),
	})
	if err != nil {
		h.l.Error("查询我的优惠券失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetCoupons()})
}

// ListSeckillActivities 秒杀活动列表（公开）
func (h *MarketingHandler) ListSeckillActivities(ctx *gin.Context) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.marketingClient.ListSeckillActivities(ctx.Request.Context(), &marketingv1.ListSeckillActivitiesRequest{
		TenantId: tenantId.(int64),
		Status:   2, // 进行中
		Page:     1,
		PageSize: 50,
	})
	if err != nil {
		h.l.Error("查询秒杀活动列表失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetActivities()})
}

// Seckill 秒杀抢购（需登录）
func (h *MarketingHandler) Seckill(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	itemIdStr := ctx.Param("itemId")
	itemId, _ := strconv.ParseInt(itemIdStr, 10, 64)
	resp, err := h.marketingClient.Seckill(ctx.Request.Context(), &marketingv1.SeckillRequest{
		UserId: uid.(int64),
		ItemId: itemId,
	})
	if err != nil {
		h.l.Error("秒杀请求失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统繁忙"})
		return
	}
	if !resp.GetSuccess() {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: resp.GetMessage()})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"message":  resp.GetMessage(),
		"order_no": resp.GetOrderNo(),
	}})
}
