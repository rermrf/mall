package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type TenantHandler struct {
	tenantClient tenantv1.TenantServiceClient
	l            logger.Logger
}

func NewTenantHandler(tenantClient tenantv1.TenantServiceClient, l logger.Logger) *TenantHandler {
	return &TenantHandler{
		tenantClient: tenantClient,
		l:            l,
	}
}

func (h *TenantHandler) GetShop(ctx *gin.Context) {
	tenantId, err := ginx.GetTenantID(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeForbidden, Msg: "需要商家身份"})
		return
	}
	resp, err := h.tenantClient.GetShop(ctx.Request.Context(), &tenantv1.GetShopRequest{
		TenantId: tenantId,
	})
	if err != nil {
		h.l.Error("获取店铺信息失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.TenantErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetShop()})
}

type UpdateShopReq struct {
	Name         string `json:"name"`
	Logo         string `json:"logo"`
	Description  string `json:"description"`
	Subdomain    string `json:"subdomain"`
	CustomDomain string `json:"customDomain"`
}

func (h *TenantHandler) UpdateShop(ctx *gin.Context, req UpdateShopReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	_, err := h.tenantClient.UpdateShop(ctx.Request.Context(), &tenantv1.UpdateShopRequest{
		Shop: &tenantv1.Shop{
			TenantId:     tenantId,
			Name:         req.Name,
			Logo:         req.Logo,
			Description:  req.Description,
			Subdomain:    req.Subdomain,
			CustomDomain: req.CustomDomain,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新店铺信息失败", ginx.TenantErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

func (h *TenantHandler) CheckQuota(ctx *gin.Context) {
	tenantId, err := ginx.GetTenantID(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeForbidden, Msg: "需要商家身份"})
		return
	}
	quotaType := ctx.Param("type")
	resp, err := h.tenantClient.CheckQuota(ctx.Request.Context(), &tenantv1.CheckQuotaRequest{
		TenantId:  tenantId,
		QuotaType: quotaType,
	})
	if err != nil {
		h.l.Error("查询配额失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.TenantErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp})
}
