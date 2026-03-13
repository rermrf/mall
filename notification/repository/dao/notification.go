package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ==================== GORM 模型 ====================

type NotificationTemplate struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	TenantID int64  `gorm:"index:uk_tenant_code_channel,unique"`
	Code     string `gorm:"type:varchar(64);index:uk_tenant_code_channel,unique"`
	Channel  int32  `gorm:"index:uk_tenant_code_channel,unique"`
	Title    string `gorm:"type:varchar(256)"`
	Content  string `gorm:"type:text"`
	Status   int32  `gorm:"default:1"`
	Ctime    int64
	Utime    int64
}

type Notification struct {
	ID       int64 `gorm:"primaryKey;autoIncrement"`
	UserID   int64 `gorm:"index:idx_user_read"`
	TenantID int64 `gorm:"index:idx_tenant"`
	Channel  int32
	Title    string `gorm:"type:varchar(256)"`
	Content  string `gorm:"type:text"`
	IsRead   bool   `gorm:"index:idx_user_read;default:false"`
	Status   int32  `gorm:"default:1"`
	Ctime    int64
	Utime    int64
}

// ==================== DAO 接口 ====================

type NotificationTemplateDAO interface {
	Insert(ctx context.Context, t NotificationTemplate) (NotificationTemplate, error)
	Update(ctx context.Context, t NotificationTemplate) error
	Delete(ctx context.Context, id, tenantId int64) error
	ListByTenantAndChannel(ctx context.Context, tenantId int64, channel int32) ([]NotificationTemplate, error)
	FindByCode(ctx context.Context, tenantId int64, code string, channel int32) (NotificationTemplate, error)
}

type NotificationDAO interface {
	Insert(ctx context.Context, n Notification) (Notification, error)
	BatchInsert(ctx context.Context, ns []Notification) error
	ListByUser(ctx context.Context, userId int64, channel int32, unreadOnly bool, offset, limit int) ([]Notification, int64, error)
	MarkRead(ctx context.Context, id, userId int64) error
	MarkAllRead(ctx context.Context, userId int64) error
	CountUnread(ctx context.Context, userId int64) (int64, error)
}

// ==================== 实现 ====================

type GORMNotificationTemplateDAO struct {
	db *gorm.DB
}

func NewNotificationTemplateDAO(db *gorm.DB) NotificationTemplateDAO {
	return &GORMNotificationTemplateDAO{db: db}
}

func (d *GORMNotificationTemplateDAO) Insert(ctx context.Context, t NotificationTemplate) (NotificationTemplate, error) {
	now := time.Now().UnixMilli()
	t.Ctime = now
	t.Utime = now
	err := d.db.WithContext(ctx).Create(&t).Error
	return t, err
}

func (d *GORMNotificationTemplateDAO) Update(ctx context.Context, t NotificationTemplate) error {
	t.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Model(&t).Updates(map[string]any{
		"title":   t.Title,
		"content": t.Content,
		"status":  t.Status,
		"utime":   t.Utime,
	}).Error
}

func (d *GORMNotificationTemplateDAO) Delete(ctx context.Context, id, tenantId int64) error {
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantId).Delete(&NotificationTemplate{}).Error
}

func (d *GORMNotificationTemplateDAO) ListByTenantAndChannel(ctx context.Context, tenantId int64, channel int32) ([]NotificationTemplate, error) {
	var templates []NotificationTemplate
	query := d.db.WithContext(ctx).Where("tenant_id = ?", tenantId)
	if channel > 0 {
		query = query.Where("channel = ?", channel)
	}
	err := query.Order("id DESC").Find(&templates).Error
	return templates, err
}

func (d *GORMNotificationTemplateDAO) FindByCode(ctx context.Context, tenantId int64, code string, channel int32) (NotificationTemplate, error) {
	var t NotificationTemplate
	err := d.db.WithContext(ctx).Where("tenant_id = ? AND code = ? AND channel = ?", tenantId, code, channel).First(&t).Error
	return t, err
}

type GORMNotificationDAO struct {
	db *gorm.DB
}

func NewNotificationDAO(db *gorm.DB) NotificationDAO {
	return &GORMNotificationDAO{db: db}
}

func (d *GORMNotificationDAO) Insert(ctx context.Context, n Notification) (Notification, error) {
	now := time.Now().UnixMilli()
	n.Ctime = now
	n.Utime = now
	err := d.db.WithContext(ctx).Create(&n).Error
	return n, err
}

func (d *GORMNotificationDAO) BatchInsert(ctx context.Context, ns []Notification) error {
	now := time.Now().UnixMilli()
	for i := range ns {
		ns[i].Ctime = now
		ns[i].Utime = now
	}
	return d.db.WithContext(ctx).CreateInBatches(ns, 100).Error
}

func (d *GORMNotificationDAO) ListByUser(ctx context.Context, userId int64, channel int32, unreadOnly bool, offset, limit int) ([]Notification, int64, error) {
	var notifications []Notification
	var total int64
	query := d.db.WithContext(ctx).Where("user_id = ?", userId)
	if channel > 0 {
		query = query.Where("channel = ?", channel)
	}
	if unreadOnly {
		query = query.Where("is_read = ?", false)
	}
	err := query.Model(&Notification{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = query.Order("id DESC").Offset(offset).Limit(limit).Find(&notifications).Error
	return notifications, total, err
}

func (d *GORMNotificationDAO) MarkRead(ctx context.Context, id, userId int64) error {
	return d.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ? AND user_id = ?", id, userId).
		Updates(map[string]any{"is_read": true, "utime": time.Now().UnixMilli()}).Error
}

func (d *GORMNotificationDAO) MarkAllRead(ctx context.Context, userId int64) error {
	return d.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userId, false).
		Updates(map[string]any{"is_read": true, "utime": time.Now().UnixMilli()}).Error
}

func (d *GORMNotificationDAO) CountUnread(ctx context.Context, userId int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ? AND is_read = ?", userId, false).Count(&count).Error
	return count, err
}
