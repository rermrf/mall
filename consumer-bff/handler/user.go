package handler

import (
	"net/http"
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
	return &UserHandler{userClient: userClient, l: l}
}

// === Signup ===

type SignupReq struct {
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) Signup(ctx *gin.Context, req SignupReq) (ginx.Result, error) {
	tenantId := tenantx.GetTenantID(ctx.Request.Context())
	resp, err := h.userClient.Signup(ctx.Request.Context(), &userv1.SignupRequest{
		TenantId: tenantId,
		Phone:    req.Phone,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "注册失败", ginx.UserErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "注册成功", Data: map[string]any{"id": resp.GetId()}}, nil
}

// === SMS ===

type SendSmsCodeReq struct {
	Phone string `json:"phone"`
	Scene int32  `json:"scene"`
}

func (h *UserHandler) SendSmsCode(ctx *gin.Context, req SendSmsCodeReq) (ginx.Result, error) {
	tenantId := tenantx.GetTenantID(ctx.Request.Context())
	_, err := h.userClient.SendSmsCode(ctx.Request.Context(), &userv1.SendSmsCodeRequest{
		TenantId: tenantId,
		Phone:    req.Phone,
		Scene:    req.Scene,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "发送验证码失败", ginx.UserErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "发送成功"}, nil
}

// === Profile ===

func (h *UserHandler) GetProfile(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	resp, err := h.userClient.FindById(ctx.Request.Context(), &userv1.FindByIdRequest{
		Id: uid.(int64),
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
	uid, _ := ctx.Get("uid")
	_, err := h.userClient.UpdateProfile(ctx.Request.Context(), &userv1.UpdateProfileRequest{
		Id:       uid.(int64),
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新个人信息失败", ginx.UserErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

// === Addresses ===

type CreateAddressReq struct {
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Province  string `json:"province"`
	City      string `json:"city"`
	District  string `json:"district"`
	Detail    string `json:"detail"`
	IsDefault bool   `json:"is_default"`
}

func (h *UserHandler) CreateAddress(ctx *gin.Context, req CreateAddressReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	resp, err := h.userClient.CreateAddress(ctx.Request.Context(), &userv1.CreateAddressRequest{
		Address: &userv1.UserAddress{
			UserId:    uid.(int64),
			Name:      req.Name,
			Phone:     req.Phone,
			Province:  req.Province,
			City:      req.City,
			District:  req.District,
			Detail:    req.Detail,
			IsDefault: req.IsDefault,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建地址失败", ginx.OrderErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

func (h *UserHandler) ListAddresses(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	resp, err := h.userClient.ListAddresses(ctx.Request.Context(), &userv1.ListAddressesRequest{
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("获取地址列表失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.OrderErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetAddresses()})
}

type UpdateAddressReq struct {
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Province  string `json:"province"`
	City      string `json:"city"`
	District  string `json:"district"`
	Detail    string `json:"detail"`
	IsDefault bool   `json:"is_default"`
}

func (h *UserHandler) UpdateAddress(ctx *gin.Context, req UpdateAddressReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的地址 ID"}, nil
	}
	uid, _ := ctx.Get("uid")
	_, err = h.userClient.UpdateAddress(ctx.Request.Context(), &userv1.UpdateAddressRequest{
		Address: &userv1.UserAddress{
			Id:        id,
			UserId:    uid.(int64),
			Name:      req.Name,
			Phone:     req.Phone,
			Province:  req.Province,
			City:      req.City,
			District:  req.District,
			Detail:    req.Detail,
			IsDefault: req.IsDefault,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新地址失败", ginx.OrderErrMappings...)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

func (h *UserHandler) DeleteAddress(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的地址 ID"})
		return
	}
	uid, _ := ctx.Get("uid")
	_, err = h.userClient.DeleteAddress(ctx.Request.Context(), &userv1.DeleteAddressRequest{
		Id:     id,
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("删除地址失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.OrderErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}
