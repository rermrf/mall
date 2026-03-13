package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rermrf/emo/logger"
	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

type TenantHandler struct {
	tenantClient tenantv1.TenantServiceClient
	l            logger.Logger
}

func NewTenantHandler(tenantClient tenantv1.TenantServiceClient, l logger.Logger) *TenantHandler {
	return &TenantHandler{tenantClient: tenantClient, l: l}
}

func (h *TenantHandler) GetShop(ctx *gin.Context) {
	// TenantResolve middleware already set "shop" in context
	shop, exists := ctx.Get("shop")
	if exists {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: shop})
		return
	}
	// Fallback: call tenant-svc
	tenantId := tenantx.GetTenantID(ctx.Request.Context())
	resp, err := h.tenantClient.GetShop(ctx.Request.Context(), &tenantv1.GetShopRequest{
		TenantId: tenantId,
	})
	if err != nil {
		h.l.Error("获取店铺信息失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetShop()})
}
