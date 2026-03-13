package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	notificationv1 "github.com/rermrf/mall/api/proto/gen/notification/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

type NotificationHandler struct {
	notificationClient notificationv1.NotificationServiceClient
	l                  logger.Logger
}

func NewNotificationHandler(notificationClient notificationv1.NotificationServiceClient, l logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		notificationClient: notificationClient,
		l:                  l,
	}
}

// ==================== 通知模板管理 ====================

type CreateTemplateReq struct {
	TenantId int64  `json:"tenant_id"`
	Code     string `json:"code" binding:"required"`
	Channel  int32  `json:"channel" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Status   int32  `json:"status"`
}

func (h *NotificationHandler) CreateTemplate(ctx *gin.Context, req CreateTemplateReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.notificationClient.CreateTemplate(rpcCtx, &notificationv1.CreateTemplateRequest{
		Template: &notificationv1.NotificationTemplate{
			TenantId: req.TenantId,
			Code:     req.Code,
			Channel:  req.Channel,
			Title:    req.Title,
			Content:  req.Content,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建通知模板失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateTemplateReq struct {
	TenantId int64  `json:"tenant_id"`
	Code     string `json:"code"`
	Channel  int32  `json:"channel"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Status   int32  `json:"status"`
}

func (h *NotificationHandler) UpdateTemplate(ctx *gin.Context, req UpdateTemplateReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	_, err := h.notificationClient.UpdateTemplate(rpcCtx, &notificationv1.UpdateTemplateRequest{
		Template: &notificationv1.NotificationTemplate{
			Id:       id,
			TenantId: req.TenantId,
			Code:     req.Code,
			Channel:  req.Channel,
			Title:    req.Title,
			Content:  req.Content,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新通知模板失败")
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListTemplatesReq struct {
	TenantId int64 `form:"tenant_id"`
	Channel  int32 `form:"channel"`
}

func (h *NotificationHandler) ListTemplates(ctx *gin.Context, req ListTemplatesReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.notificationClient.ListTemplates(rpcCtx, &notificationv1.ListTemplatesRequest{
		TenantId: req.TenantId,
		Channel:  req.Channel,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询通知模板列表失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetTemplates()}, nil
}

func (h *NotificationHandler) DeleteTemplate(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.notificationClient.DeleteTemplate(ctx.Request.Context(), &notificationv1.DeleteTemplateRequest{
		Id: id,
	})
	if err != nil {
		h.l.Error("删除通知模板失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

// ==================== 发送通知 ====================

type SendNotificationReq struct {
	UserId       int64             `json:"user_id" binding:"required"`
	TenantId     int64             `json:"tenant_id"`
	TemplateCode string            `json:"template_code" binding:"required"`
	Channel      int32             `json:"channel" binding:"required"`
	Params       map[string]string `json:"params"`
}

func (h *NotificationHandler) SendNotification(ctx *gin.Context, req SendNotificationReq) (ginx.Result, error) {
	rpcCtx := ctx.Request.Context()
	if req.TenantId > 0 {
		rpcCtx = tenantx.WithTenantID(rpcCtx, req.TenantId)
	}
	resp, err := h.notificationClient.SendNotification(rpcCtx, &notificationv1.SendNotificationRequest{
		UserId:       req.UserId,
		TenantId:     req.TenantId,
		TemplateCode: req.TemplateCode,
		Channel:      req.Channel,
		Params:       req.Params,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "发送通知失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}
