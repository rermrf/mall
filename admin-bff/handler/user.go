package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

type UserHandler struct {
	userClient userv1.UserServiceClient
	l          logger.Logger
}

func NewUserHandler(userClient userv1.UserServiceClient, l logger.Logger) *UserHandler {
	return &UserHandler{
		userClient: userClient,
		l:          l,
	}
}

type ListUsersReq struct {
	TenantId int64  `form:"tenant_id"`
	Page     int32  `form:"page"`
	PageSize int32  `form:"page_size"`
	Status   int32  `form:"status"`
	Keyword  string `form:"keyword"`
}

func (h *UserHandler) ListUsers(ctx *gin.Context, req ListUsersReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.userClient.ListUsers(rpcCtx, &userv1.ListUsersRequest{
		TenantId: req.TenantId,
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   req.Status,
		Keyword:  req.Keyword,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用用户服务列表失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
		Data: resp,
	}, nil
}

type UpdateUserStatusReq struct {
	Status int32 `json:"status"`
}

func (h *UserHandler) UpdateUserStatus(ctx *gin.Context, req UpdateUserStatusReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的用户 ID"}, nil
	}

	_, err = h.userClient.UpdateUserStatus(ctx.Request.Context(), &userv1.UpdateUserStatusRequest{
		Id:     id,
		Status: req.Status,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用用户服务更新状态失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
	}, nil
}

type ListRolesReq struct {
	TenantId int64 `form:"tenant_id"`
}

func (h *UserHandler) ListRoles(ctx *gin.Context, req ListRolesReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.userClient.ListRoles(rpcCtx, &userv1.ListRolesRequest{
		TenantId: req.TenantId,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用用户服务角色列表失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
		Data: resp,
	}, nil
}

type CreateRoleReq struct {
	TenantId    int64  `json:"tenant_id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

func (h *UserHandler) CreateRole(ctx *gin.Context, req CreateRoleReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.userClient.CreateRole(rpcCtx, &userv1.CreateRoleRequest{
		Role: &userv1.Role{
			TenantId:    req.TenantId,
			Name:        req.Name,
			Code:        req.Code,
			Description: req.Description,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用用户服务创建角色失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
		Data: resp,
	}, nil
}

type UpdateRoleReq struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

func (h *UserHandler) UpdateRole(ctx *gin.Context, req UpdateRoleReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的角色 ID"}, nil
	}

	_, err = h.userClient.UpdateRole(ctx.Request.Context(), &userv1.UpdateRoleRequest{
		Role: &userv1.Role{
			Id:          id,
			Name:        req.Name,
			Code:        req.Code,
			Description: req.Description,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用用户服务更新角色失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "success",
	}, nil
}
