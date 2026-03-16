package service

import (
	"bytes"
	"context"
	"errors"
	"text/template"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/notification/domain"
	"github.com/rermrf/mall/notification/repository"
	"github.com/rermrf/mall/notification/service/provider"
)

var (
	ErrTemplateNotFound = errors.New("通知模板不存在")
	ErrTemplateDisabled = errors.New("通知模板已停用")
)

type NotificationService interface {
	SendNotification(ctx context.Context, userId, tenantId int64, templateCode string, channel int32, params map[string]string) (domain.Notification, error)
	BatchSendNotification(ctx context.Context, userIds []int64, tenantId int64, templateCode string, channel int32, params map[string]string) (int32, int32, error)
	ListNotifications(ctx context.Context, userId int64, channel int32, unreadOnly bool, page, pageSize int32) ([]domain.Notification, int64, error)
	MarkRead(ctx context.Context, id, userId int64) error
	MarkAllRead(ctx context.Context, userId int64) error
	GetUnreadCount(ctx context.Context, userId int64) (int64, error)
	DeleteNotification(ctx context.Context, id, userId int64) error
	CreateTemplate(ctx context.Context, t domain.NotificationTemplate) (domain.NotificationTemplate, error)
	UpdateTemplate(ctx context.Context, t domain.NotificationTemplate) error
	DeleteTemplate(ctx context.Context, id, tenantId int64) error
	ListTemplates(ctx context.Context, tenantId int64, channel int32) ([]domain.NotificationTemplate, error)
}

type notificationService struct {
	repo          repository.NotificationRepository
	smsProvider   provider.SmsProvider
	emailProvider provider.EmailProvider
	l             logger.Logger
	smsSignName   string
}

func NewNotificationService(
	repo repository.NotificationRepository,
	smsProvider provider.SmsProvider,
	emailProvider provider.EmailProvider,
	l logger.Logger,
) NotificationService {
	return &notificationService{
		repo: repo, smsProvider: smsProvider, emailProvider: emailProvider,
		l: l, smsSignName: "商城",
	}
}

func (s *notificationService) SendNotification(ctx context.Context, userId, tenantId int64, templateCode string, channel int32, params map[string]string) (domain.Notification, error) {
	tmpl, err := s.findTemplate(ctx, tenantId, templateCode, channel)
	if err != nil {
		return domain.Notification{}, err
	}
	title, _ := renderTemplate(tmpl.Title, params)
	content, _ := renderTemplate(tmpl.Content, params)

	status := int32(2) // 已发送
	switch channel {
	case 1: // SMS
		if phone, ok := params["phone"]; ok {
			if err = s.smsProvider.Send(ctx, phone, s.smsSignName, templateCode, params); err != nil {
				s.l.Error("发送短信失败", logger.Error(err))
				status = 3
			}
		}
	case 2: // Email
		if email, ok := params["email"]; ok {
			if err = s.emailProvider.Send(ctx, email, title, content); err != nil {
				s.l.Error("发送邮件失败", logger.Error(err))
				status = 3
			}
		}
	case 3: // 站内信
	}

	n, err := s.repo.CreateNotification(ctx, domain.Notification{
		UserID: userId, TenantID: tenantId, Channel: channel,
		Title: title, Content: content, Status: status,
	})
	if err != nil {
		return domain.Notification{}, err
	}
	return n, nil
}

func (s *notificationService) BatchSendNotification(ctx context.Context, userIds []int64, tenantId int64, templateCode string, channel int32, params map[string]string) (int32, int32, error) {
	var successCount, failCount int32
	for _, uid := range userIds {
		_, err := s.SendNotification(ctx, uid, tenantId, templateCode, channel, params)
		if err != nil {
			s.l.Error("批量发送通知失败", logger.Error(err), logger.Int64("userId", uid))
			failCount++
		} else {
			successCount++
		}
	}
	return successCount, failCount, nil
}

func (s *notificationService) ListNotifications(ctx context.Context, userId int64, channel int32, unreadOnly bool, page, pageSize int32) ([]domain.Notification, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return s.repo.ListNotifications(ctx, userId, channel, unreadOnly, page, pageSize)
}

func (s *notificationService) MarkRead(ctx context.Context, id, userId int64) error {
	return s.repo.MarkRead(ctx, id, userId)
}

func (s *notificationService) MarkAllRead(ctx context.Context, userId int64) error {
	return s.repo.MarkAllRead(ctx, userId)
}

func (s *notificationService) GetUnreadCount(ctx context.Context, userId int64) (int64, error) {
	return s.repo.GetUnreadCount(ctx, userId)
}

func (s *notificationService) DeleteNotification(ctx context.Context, id, userId int64) error {
	return s.repo.DeleteNotification(ctx, id, userId)
}

func (s *notificationService) CreateTemplate(ctx context.Context, t domain.NotificationTemplate) (domain.NotificationTemplate, error) {
	return s.repo.CreateTemplate(ctx, t)
}

func (s *notificationService) UpdateTemplate(ctx context.Context, t domain.NotificationTemplate) error {
	return s.repo.UpdateTemplate(ctx, t)
}

func (s *notificationService) DeleteTemplate(ctx context.Context, id, tenantId int64) error {
	return s.repo.DeleteTemplate(ctx, id, tenantId)
}

func (s *notificationService) ListTemplates(ctx context.Context, tenantId int64, channel int32) ([]domain.NotificationTemplate, error) {
	return s.repo.ListTemplates(ctx, tenantId, channel)
}

func (s *notificationService) findTemplate(ctx context.Context, tenantId int64, code string, channel int32) (domain.NotificationTemplate, error) {
	tmpl, err := s.repo.FindTemplateByCode(ctx, tenantId, code, channel)
	if err == nil && tmpl.Status == 1 {
		return tmpl, nil
	}
	if tenantId != 0 {
		tmpl, err = s.repo.FindTemplateByCode(ctx, 0, code, channel)
		if err == nil && tmpl.Status == 1 {
			return tmpl, nil
		}
	}
	return domain.NotificationTemplate{}, ErrTemplateNotFound
}

func renderTemplate(tmplStr string, params map[string]string) (string, error) {
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return tmplStr, err
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, params); err != nil {
		return tmplStr, err
	}
	return buf.String(), nil
}
