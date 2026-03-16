package repository

import (
	"context"
	"time"

	"github.com/rermrf/mall/notification/domain"
	"github.com/rermrf/mall/notification/repository/cache"
	"github.com/rermrf/mall/notification/repository/dao"
)

type NotificationRepository interface {
	// 模板
	CreateTemplate(ctx context.Context, t domain.NotificationTemplate) (domain.NotificationTemplate, error)
	UpdateTemplate(ctx context.Context, t domain.NotificationTemplate) error
	DeleteTemplate(ctx context.Context, id, tenantId int64) error
	ListTemplates(ctx context.Context, tenantId int64, channel int32) ([]domain.NotificationTemplate, error)
	FindTemplateByCode(ctx context.Context, tenantId int64, code string, channel int32) (domain.NotificationTemplate, error)
	// 通知
	CreateNotification(ctx context.Context, n domain.Notification) (domain.Notification, error)
	BatchCreateNotifications(ctx context.Context, ns []domain.Notification) error
	ListNotifications(ctx context.Context, userId int64, channel int32, unreadOnly bool, page, pageSize int32) ([]domain.Notification, int64, error)
	MarkRead(ctx context.Context, id, userId int64) error
	MarkAllRead(ctx context.Context, userId int64) error
	GetUnreadCount(ctx context.Context, userId int64) (int64, error)
	DeleteNotification(ctx context.Context, id, userId int64) error
}

type notificationRepository struct {
	templateDAO     dao.NotificationTemplateDAO
	notificationDAO dao.NotificationDAO
	cache           cache.NotificationCache
}

func NewNotificationRepository(
	templateDAO dao.NotificationTemplateDAO,
	notificationDAO dao.NotificationDAO,
	c cache.NotificationCache,
) NotificationRepository {
	return &notificationRepository{
		templateDAO:     templateDAO,
		notificationDAO: notificationDAO,
		cache:           c,
	}
}

// ==================== 模板 ====================

func (r *notificationRepository) CreateTemplate(ctx context.Context, t domain.NotificationTemplate) (domain.NotificationTemplate, error) {
	dt, err := r.templateDAO.Insert(ctx, r.templateToDAO(t))
	if err != nil {
		return domain.NotificationTemplate{}, err
	}
	return r.templateToDomain(dt), nil
}

func (r *notificationRepository) UpdateTemplate(ctx context.Context, t domain.NotificationTemplate) error {
	return r.templateDAO.Update(ctx, r.templateToDAO(t))
}

func (r *notificationRepository) DeleteTemplate(ctx context.Context, id, tenantId int64) error {
	return r.templateDAO.Delete(ctx, id, tenantId)
}

func (r *notificationRepository) ListTemplates(ctx context.Context, tenantId int64, channel int32) ([]domain.NotificationTemplate, error) {
	templates, err := r.templateDAO.ListByTenantAndChannel(ctx, tenantId, channel)
	if err != nil {
		return nil, err
	}
	result := make([]domain.NotificationTemplate, 0, len(templates))
	for _, t := range templates {
		result = append(result, r.templateToDomain(t))
	}
	return result, nil
}

func (r *notificationRepository) FindTemplateByCode(ctx context.Context, tenantId int64, code string, channel int32) (domain.NotificationTemplate, error) {
	t, err := r.templateDAO.FindByCode(ctx, tenantId, code, channel)
	if err != nil {
		return domain.NotificationTemplate{}, err
	}
	return r.templateToDomain(t), nil
}

// ==================== 通知 ====================

func (r *notificationRepository) CreateNotification(ctx context.Context, n domain.Notification) (domain.Notification, error) {
	dn, err := r.notificationDAO.Insert(ctx, r.notificationToDAO(n))
	if err != nil {
		return domain.Notification{}, err
	}
	// 清除未读缓存
	_ = r.cache.DeleteUnreadCount(ctx, n.UserID)
	return r.notificationToDomain(dn), nil
}

func (r *notificationRepository) BatchCreateNotifications(ctx context.Context, ns []domain.Notification) error {
	daoNs := make([]dao.Notification, 0, len(ns))
	userIds := make(map[int64]struct{})
	for _, n := range ns {
		daoNs = append(daoNs, r.notificationToDAO(n))
		userIds[n.UserID] = struct{}{}
	}
	err := r.notificationDAO.BatchInsert(ctx, daoNs)
	if err != nil {
		return err
	}
	for uid := range userIds {
		_ = r.cache.DeleteUnreadCount(ctx, uid)
	}
	return nil
}

func (r *notificationRepository) ListNotifications(ctx context.Context, userId int64, channel int32, unreadOnly bool, page, pageSize int32) ([]domain.Notification, int64, error) {
	offset := int((page - 1) * pageSize)
	ns, total, err := r.notificationDAO.ListByUser(ctx, userId, channel, unreadOnly, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	result := make([]domain.Notification, 0, len(ns))
	for _, n := range ns {
		result = append(result, r.notificationToDomain(n))
	}
	return result, total, nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, id, userId int64) error {
	err := r.notificationDAO.MarkRead(ctx, id, userId)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteUnreadCount(ctx, userId)
	return nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userId int64) error {
	err := r.notificationDAO.MarkAllRead(ctx, userId)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteUnreadCount(ctx, userId)
	return nil
}

func (r *notificationRepository) GetUnreadCount(ctx context.Context, userId int64) (int64, error) {
	count, err := r.cache.GetUnreadCount(ctx, userId)
	if err == nil {
		return count, nil
	}
	count, err = r.notificationDAO.CountUnread(ctx, userId)
	if err != nil {
		return 0, err
	}
	_ = r.cache.SetUnreadCount(ctx, userId, count)
	return count, nil
}

func (r *notificationRepository) DeleteNotification(ctx context.Context, id, userId int64) error {
	err := r.notificationDAO.DeleteByIdAndUser(ctx, id, userId)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteUnreadCount(ctx, userId)
	return nil
}

// ==================== 转换 ====================

func (r *notificationRepository) templateToDAO(t domain.NotificationTemplate) dao.NotificationTemplate {
	return dao.NotificationTemplate{
		ID: t.ID, TenantID: t.TenantID, Code: t.Code, Channel: t.Channel,
		Title: t.Title, Content: t.Content, Status: t.Status,
	}
}

func (r *notificationRepository) templateToDomain(t dao.NotificationTemplate) domain.NotificationTemplate {
	return domain.NotificationTemplate{
		ID: t.ID, TenantID: t.TenantID, Code: t.Code, Channel: t.Channel,
		Title: t.Title, Content: t.Content, Status: t.Status,
		Ctime: time.UnixMilli(t.Ctime), Utime: time.UnixMilli(t.Utime),
	}
}

func (r *notificationRepository) notificationToDAO(n domain.Notification) dao.Notification {
	return dao.Notification{
		ID: n.ID, UserID: n.UserID, TenantID: n.TenantID, Channel: n.Channel,
		Title: n.Title, Content: n.Content, IsRead: n.IsRead, Status: n.Status,
	}
}

func (r *notificationRepository) notificationToDomain(n dao.Notification) domain.Notification {
	return domain.Notification{
		ID: n.ID, UserID: n.UserID, TenantID: n.TenantID, Channel: n.Channel,
		Title: n.Title, Content: n.Content, IsRead: n.IsRead, Status: n.Status,
		Ctime: time.UnixMilli(n.Ctime), Utime: time.UnixMilli(n.Utime),
	}
}
