package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/validatorx"
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
	uid, err := ginx.GetUID(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeUnauthorized, Msg: "未授权"})
		return
	}
	resp, err := h.userClient.FindById(ctx.Request.Context(), &userv1.FindByIdRequest{
		Id: uid,
	})
	if err != nil {
		h.l.Error("获取个人信息失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.UserErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetUser()})
}

type UpdateProfileReq struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

func (h *UserHandler) UpdateProfile(ctx *gin.Context, req UpdateProfileReq) (ginx.Result, error) {
	uid, errResult := ginx.MustGetUID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	_, err := h.userClient.UpdateProfile(ctx.Request.Context(), &userv1.UpdateProfileRequest{
		Id:       uid,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新个人信息失败", ginx.UserErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListStaffReq struct {
	Page     int32 `form:"page"`
	PageSize int32 `form:"pageSize"`
}

func (h *UserHandler) ListStaff(ctx *gin.Context, req ListStaffReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.userClient.ListUsers(ctx.Request.Context(), &userv1.ListUsersRequest{
		TenantId: tenantId,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "获取员工列表失败", ginx.UserErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp}, nil
}

type ListRolesReq struct{}

func (h *UserHandler) ListRoles(ctx *gin.Context, _ ListRolesReq) (ginx.Result, error) {
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.userClient.ListRoles(ctx.Request.Context(), &userv1.ListRolesRequest{
		TenantId: tenantId,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "获取角色列表失败", ginx.UserErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetRoles()}, nil
}

type CreateRoleReq struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

func (h *UserHandler) CreateRole(ctx *gin.Context, req CreateRoleReq) (ginx.Result, error) {
	v := validatorx.New()
	v.CheckNotBlank("name", req.Name)
	v.CheckNotBlank("code", req.Code)
	if v.HasErrors() {
		return v.ToResult(), nil
	}
	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	resp, err := h.userClient.CreateRole(ctx.Request.Context(), &userv1.CreateRoleRequest{
		Role: &userv1.Role{
			TenantId:    tenantId,
			Name:        req.Name,
			Code:        req.Code,
			Description: req.Description,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建角色失败", ginx.UserErrMappings...)
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
		return ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的角色 ID"}, nil
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
		return ginx.HandleGRPCError(err, "更新角色失败", ginx.UserErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type AssignRoleReq struct {
	RoleId int64 `json:"roleId"`
}

func (h *UserHandler) AssignRole(ctx *gin.Context, req AssignRoleReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	userId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的员工 ID"}, nil
	}

	tenantId, errResult := ginx.MustGetTenantID(ctx)
	if errResult != nil {
		return *errResult, nil
	}
	_, err = h.userClient.AssignRole(ctx.Request.Context(), &userv1.AssignRoleRequest{
		UserId:   userId,
		TenantId: tenantId,
		RoleId:   req.RoleId,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "分配角色失败", ginx.UserErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}
