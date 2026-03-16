package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rermrf/emo/logger"
	notificationv1 "github.com/rermrf/mall/api/proto/gen/notification/v1"
	"github.com/rermrf/mall/pkg/ginx"
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

type ListNotificationsReq struct {
	Channel    int32 `form:"channel"`
	UnreadOnly bool  `form:"unreadOnly"`
	Page       int32 `form:"page"`
	PageSize   int32 `form:"pageSize"`
}

func (h *NotificationHandler) ListNotifications(ctx *gin.Context, req ListNotificationsReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	resp, err := h.notificationClient.ListNotifications(ctx.Request.Context(), &notificationv1.ListNotificationsRequest{
		UserId:     uid.(int64),
		Channel:    req.Channel,
		UnreadOnly: req.UnreadOnly,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		h.l.Error("查询通知列表失败", logger.Error(err))
		return ginx.HandleGRPCError(err, "查询通知列表失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"notifications": resp.GetNotifications(),
		"total":         resp.GetTotal(),
	}}, nil
}

func (h *NotificationHandler) GetUnreadCount(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	resp, err := h.notificationClient.GetUnreadCount(ctx.Request.Context(), &notificationv1.GetUnreadCountRequest{
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("查询未读数量失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetCount()})
}

func (h *NotificationHandler) MarkRead(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.notificationClient.MarkRead(ctx.Request.Context(), &notificationv1.MarkReadRequest{
		Id:     id,
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("标记已读失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

func (h *NotificationHandler) MarkAllRead(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	_, err := h.notificationClient.MarkAllRead(ctx.Request.Context(), &notificationv1.MarkAllReadRequest{
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("全部标记已读失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

func (h *NotificationHandler) DeleteNotification(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.notificationClient.DeleteNotification(ctx.Request.Context(), &notificationv1.DeleteNotificationRequest{
		Id:     id,
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("删除通知失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}
