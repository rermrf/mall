package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/pkg/ginx"
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

func (h *UserHandler) GetProfile(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	resp, err := h.userClient.FindById(ctx.Request.Context(), &userv1.FindByIdRequest{
		Id: uid.(int64),
	})
	if err != nil {
		h.l.Error("获取个人信息失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetUser()})
}

type UpdateProfileReq struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

func (h *UserHandler) UpdateProfile(ctx *gin.Context, req UpdateProfileReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	_, err := h.userClient.UpdateProfile(ctx.Request.Context(), &userv1.UpdateProfileRequest{
		Id:       uid.(int64),
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新个人信息失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListStaffReq struct {
	Page     int32 `form:"page"`
	PageSize int32 `form:"page_size"`
}

func (h *UserHandler) ListStaff(ctx *gin.Context, req ListStaffReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.userClient.ListUsers(ctx.Request.Context(), &userv1.ListUsersRequest{
		TenantId: tenantId.(int64),
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("获取员工列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp}, nil
}

type ListRolesReq struct{}

func (h *UserHandler) ListRoles(ctx *gin.Context, _ ListRolesReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.userClient.ListRoles(ctx.Request.Context(), &userv1.ListRolesRequest{
		TenantId: tenantId.(int64),
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("获取角色列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp}, nil
}

type CreateRoleReq struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

func (h *UserHandler) CreateRole(ctx *gin.Context, req CreateRoleReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.userClient.CreateRole(ctx.Request.Context(), &userv1.CreateRoleRequest{
		Role: &userv1.Role{
			TenantId:    tenantId.(int64),
			Name:        req.Name,
			Code:        req.Code,
			Description: req.Description,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建角色失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp}, nil
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
		return ginx.Result{}, fmt.Errorf("更新角色失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type AssignRoleReq struct {
	RoleId int64 `json:"role_id"`
}

func (h *UserHandler) AssignRole(ctx *gin.Context, req AssignRoleReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	userId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的员工 ID"}, nil
	}

	tenantId, _ := ctx.Get("tenant_id")
	_, err = h.userClient.AssignRole(ctx.Request.Context(), &userv1.AssignRoleRequest{
		UserId:   userId,
		TenantId: tenantId.(int64),
		RoleId:   req.RoleId,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("分配角色失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}
