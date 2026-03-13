package handler

import (
	"fmt"
	"net/http"
	"strconv"

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

type CreateTenantReq struct {
	Name            string `json:"name"`
	ContactName     string `json:"contact_name"`
	ContactPhone    string `json:"contact_phone"`
	BusinessLicense string `json:"business_license"`
	PlanId          int64  `json:"plan_id"`
}

func (h *TenantHandler) CreateTenant(ctx *gin.Context, req CreateTenantReq) (ginx.Result, error) {
	resp, err := h.tenantClient.CreateTenant(ctx.Request.Context(), &tenantv1.CreateTenantRequest{
		Tenant: &tenantv1.Tenant{
			Name:            req.Name,
			ContactName:     req.ContactName,
			ContactPhone:    req.ContactPhone,
			BusinessLicense: req.BusinessLicense,
			PlanId:          req.PlanId,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用租户服务创建租户失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
		Data: resp,
	}, nil
}

type ListTenantsReq struct {
	Page     int32 `form:"page"`
	PageSize int32 `form:"page_size"`
	Status   int32 `form:"status"`
}

func (h *TenantHandler) ListTenants(ctx *gin.Context, req ListTenantsReq) (ginx.Result, error) {
	resp, err := h.tenantClient.ListTenants(ctx.Request.Context(), &tenantv1.ListTenantsRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   req.Status,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用租户服务列表失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
		Data: resp,
	}, nil
}

func (h *TenantHandler) GetTenant(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的租户 ID"})
		return
	}

	resp, err := h.tenantClient.GetTenant(ctx.Request.Context(), &tenantv1.GetTenantRequest{
		Id: id,
	})
	if err != nil {
		h.l.Error("调用租户服务获取租户失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}

	ctx.JSON(http.StatusOK, ginx.Result{
		Code: 0,
		Msg:  "success",
		Data: resp.GetTenant(),
	})
}

type ApproveTenantReq struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

func (h *TenantHandler) ApproveTenant(ctx *gin.Context, req ApproveTenantReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的租户 ID"}, nil
	}

	_, err = h.tenantClient.ApproveTenant(ctx.Request.Context(), &tenantv1.ApproveTenantRequest{
		Id:       id,
		Approved: req.Approved,
		Reason:   req.Reason,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用租户服务审核租户失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
	}, nil
}

type FreezeTenantReq struct {
	Freeze bool `json:"freeze"`
}

func (h *TenantHandler) FreezeTenant(ctx *gin.Context, req FreezeTenantReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的租户 ID"}, nil
	}

	_, err = h.tenantClient.FreezeTenant(ctx.Request.Context(), &tenantv1.FreezeTenantRequest{
		Id:     id,
		Freeze: req.Freeze,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用租户服务冻结租户失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
	}, nil
}

type ListPlansReq struct{}

func (h *TenantHandler) ListPlans(ctx *gin.Context, _ ListPlansReq) (ginx.Result, error) {
	resp, err := h.tenantClient.ListPlans(ctx.Request.Context(), &tenantv1.ListPlansRequest{})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用租户服务套餐列表失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
		Data: resp,
	}, nil
}

type CreatePlanReq struct {
	Name         string `json:"name"`
	Price        int64  `json:"price"`
	DurationDays int32  `json:"duration_days"`
	MaxProducts  int32  `json:"max_products"`
	MaxStaff     int32  `json:"max_staff"`
	Features     string `json:"features"`
}

func (h *TenantHandler) CreatePlan(ctx *gin.Context, req CreatePlanReq) (ginx.Result, error) {
	resp, err := h.tenantClient.CreatePlan(ctx.Request.Context(), &tenantv1.CreatePlanRequest{
		Plan: &tenantv1.TenantPlan{
			Name:         req.Name,
			Price:        req.Price,
			DurationDays: req.DurationDays,
			MaxProducts:  req.MaxProducts,
			MaxStaff:     req.MaxStaff,
			Features:     req.Features,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用租户服务创建套餐失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
		Data: resp,
	}, nil
}

type UpdatePlanReq struct {
	Name         string `json:"name"`
	Price        int64  `json:"price"`
	DurationDays int32  `json:"duration_days"`
	MaxProducts  int32  `json:"max_products"`
	MaxStaff     int32  `json:"max_staff"`
	Features     string `json:"features"`
}

func (h *TenantHandler) UpdatePlan(ctx *gin.Context, req UpdatePlanReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的套餐 ID"}, nil
	}

	_, err = h.tenantClient.UpdatePlan(ctx.Request.Context(), &tenantv1.UpdatePlanRequest{
		Plan: &tenantv1.TenantPlan{
			Id:           id,
			Name:         req.Name,
			Price:        req.Price,
			DurationDays: req.DurationDays,
			MaxProducts:  req.MaxProducts,
			MaxStaff:     req.MaxStaff,
			Features:     req.Features,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用租户服务更新套餐失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
	}, nil
}
