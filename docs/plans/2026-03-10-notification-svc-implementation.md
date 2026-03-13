# Notification Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement notification-svc microservice (10 gRPC RPCs) + Kafka consumers (5 topics) + 3 channel providers (Aliyun SMS, SMTP Email, In-App) + consumer-bff notification endpoints (4) + merchant-bff notification endpoints (4).

**Architecture:** DDD layered architecture (domain → dao → cache → repository → service → grpc → events → ioc → wire) consistent with all other services. Kafka consumes 5 event topics from other services, renders templates via Go text/template, dispatches via channel providers. Redis caches unread count. Both BFFs expose notification query/read endpoints.

**Tech Stack:** Go, gRPC, GORM/MySQL, go-redis, Sarama/Kafka, Wire DI, Gin (BFF), etcd service discovery, Viper config, Aliyun SMS SDK, net/smtp

---

## Reference Files

Before implementing, read these files to understand established patterns:

- **Proto (generated):** `api/proto/gen/notification/v1/notification_grpc.pb.go`, `notification.pb.go`
- **Pattern reference (marketing-svc):** `marketing/domain/marketing.go`, `marketing/repository/dao/marketing.go`, `marketing/repository/cache/marketing.go`, `marketing/repository/marketing.go`, `marketing/service/marketing.go`, `marketing/grpc/marketing.go`, `marketing/events/consumer.go`, `marketing/events/types.go`, `marketing/ioc/*.go`, `marketing/wire.go`, `marketing/app.go`, `marketing/main.go`
- **BFF patterns:** `consumer-bff/handler/marketing.go`, `consumer-bff/ioc/grpc.go`, `consumer-bff/ioc/gin.go`, `consumer-bff/wire.go`, `merchant-bff/handler/logistics.go`, `merchant-bff/ioc/grpc.go`, `merchant-bff/ioc/gin.go`, `merchant-bff/wire.go`
- **Event types from other services:**
  - `user/events/types.go` — `UserRegisteredEvent{UserId, TenantId, Phone}`
  - `payment/events/types.go` — `OrderPaidEvent{OrderNo, PaymentNo, PaidAt}`
  - `logistics/events/types.go` — `OrderShippedEvent{OrderId, TenantId, CarrierCode, CarrierName, TrackingNo}`
  - `inventory/events/types.go` — `InventoryAlertEvent{TenantID, SKUID, Available, Threshold}`
  - `tenant/events/types.go` — `TenantApprovedEvent{TenantId, Name, PlanId}`
- **Kafka consumer pattern:** `marketing/events/consumer.go`, `marketing/ioc/kafka.go`, `pkg/saramax/types.go`, `pkg/saramax/consumer_handler.go`

---

## Task 1: Domain Models + DAO + Init

**Files:**
- Create: `notification/domain/notification.go`
- Create: `notification/repository/dao/notification.go`
- Create: `notification/repository/dao/init.go`

**Step 1: Create domain models**

Create `notification/domain/notification.go`:

```go
package domain

import "time"

// ==================== 通知模板 ====================

type NotificationTemplate struct {
	ID       int64
	TenantID int64  // 0=平台模板
	Code     string // 模板编码：welcome_sms, order_paid_merchant 等
	Channel  int32  // 1-短信 2-邮件 3-站内信
	Title    string
	Content  string // 模板内容，支持 Go text/template 占位符
	Status   int32  // 1-启用 2-停用
	Ctime    time.Time
	Utime    time.Time
}

// ==================== 通知记录 ====================

type Notification struct {
	ID       int64
	UserID   int64
	TenantID int64
	Channel  int32 // 1-短信 2-邮件 3-站内信
	Title    string
	Content  string
	IsRead   bool
	Status   int32 // 1-待发送 2-已发送 3-发送失败
	Ctime    time.Time
	Utime    time.Time
}
```

**Step 2: Create DAO models + interfaces + implementations**

Create `notification/repository/dao/notification.go`:

```go
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
```

**Step 3: Create init.go**

Create `notification/repository/dao/init.go`:

```go
package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&NotificationTemplate{},
		&Notification{},
	)
}
```

**Step 4: Verify build**

Run: `go build ./notification/domain/... && go build ./notification/repository/dao/...`
Expected: PASS

---

## Task 2: Cache + Repository

**Files:**
- Create: `notification/repository/cache/notification.go`
- Create: `notification/repository/notification.go`

**Step 1: Create cache layer**

Create `notification/repository/cache/notification.go`:

```go
package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type NotificationCache interface {
	GetUnreadCount(ctx context.Context, userId int64) (int64, error)
	SetUnreadCount(ctx context.Context, userId int64, count int64) error
	DeleteUnreadCount(ctx context.Context, userId int64) error
}

type RedisNotificationCache struct {
	client redis.Cmdable
}

func NewNotificationCache(client redis.Cmdable) NotificationCache {
	return &RedisNotificationCache{client: client}
}

func unreadKey(userId int64) string {
	return fmt.Sprintf("notification:unread:%d", userId)
}

func (c *RedisNotificationCache) GetUnreadCount(ctx context.Context, userId int64) (int64, error) {
	val, err := c.client.Get(ctx, unreadKey(userId)).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

func (c *RedisNotificationCache) SetUnreadCount(ctx context.Context, userId int64, count int64) error {
	return c.client.Set(ctx, unreadKey(userId), count, 10*time.Minute).Err()
}

func (c *RedisNotificationCache) DeleteUnreadCount(ctx context.Context, userId int64) error {
	return c.client.Del(ctx, unreadKey(userId)).Err()
}
```

**Step 2: Create repository layer**

Create `notification/repository/notification.go`:

```go
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
```

**Step 3: Verify build**

Run: `go build ./notification/repository/...`
Expected: PASS

---

## Task 3: Channel Providers + Service

**Files:**
- Create: `notification/service/provider/sms.go`
- Create: `notification/service/provider/email.go`
- Create: `notification/service/notification.go`

**Step 1: Create SMS provider**

Create `notification/service/provider/sms.go`:

```go
package provider

import (
	"context"
	"encoding/json"
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
)

type SmsProvider interface {
	Send(ctx context.Context, phone string, signName string, templateCode string, params map[string]string) error
}

type AliyunSmsProvider struct {
	client *dysmsapi.Client
}

func NewAliyunSmsProvider(accessKeyId, accessKeySecret, endpoint string) SmsProvider {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
		Endpoint:        tea.String(endpoint),
	}
	client, err := dysmsapi.NewClient(config)
	if err != nil {
		panic(fmt.Errorf("创建阿里云 SMS 客户端失败: %w", err))
	}
	return &AliyunSmsProvider{client: client}
}

func (p *AliyunSmsProvider) Send(ctx context.Context, phone string, signName string, templateCode string, params map[string]string) error {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("序列化短信参数失败: %w", err)
	}
	req := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(signName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(string(paramsJSON)),
	}
	_, err = p.client.SendSms(req)
	if err != nil {
		return fmt.Errorf("发送短信失败: %w", err)
	}
	return nil
}
```

**Step 2: Create Email provider**

Create `notification/service/provider/email.go`:

```go
package provider

import (
	"context"
	"fmt"
	"net/smtp"
)

type EmailProvider interface {
	Send(ctx context.Context, to string, subject string, body string) error
}

type SmtpEmailProvider struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewSmtpEmailProvider(host string, port int, username, password, from string) EmailProvider {
	return &SmtpEmailProvider{
		host: host, port: port, username: username, password: password, from: from,
	}
}

func (p *SmtpEmailProvider) Send(ctx context.Context, to string, subject string, body string) error {
	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	auth := smtp.PlainAuth("", p.username, p.password, p.host)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		p.from, to, subject, body)
	err := smtp.SendMail(addr, auth, p.from, []string{to}, []byte(msg))
	if err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}
	return nil
}
```

**Step 3: Create service layer**

Create `notification/service/notification.go`:

```go
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
```

**Step 4: Verify build**

Run: `go mod tidy && go build ./notification/service/...`
Expected: PASS

---

## Task 4: gRPC Handler

**Files:**
- Create: `notification/grpc/notification.go`

**Step 1: Create gRPC handler**

Create `notification/grpc/notification.go`:

```go
package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	notificationv1 "github.com/rermrf/mall/api/proto/gen/notification/v1"
	"github.com/rermrf/mall/notification/domain"
	"github.com/rermrf/mall/notification/service"
)

type NotificationGRPCServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	svc service.NotificationService
}

func NewNotificationGRPCServer(svc service.NotificationService) *NotificationGRPCServer {
	return &NotificationGRPCServer{svc: svc}
}

func (s *NotificationGRPCServer) Register(server *grpc.Server) {
	notificationv1.RegisterNotificationServiceServer(server, s)
}

func (s *NotificationGRPCServer) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	n, err := s.svc.SendNotification(ctx, req.GetUserId(), req.GetTenantId(), req.GetTemplateCode(), req.GetChannel(), req.GetParams())
	if err != nil {
		return nil, err
	}
	return &notificationv1.SendNotificationResponse{Id: n.ID}, nil
}

func (s *NotificationGRPCServer) BatchSendNotification(ctx context.Context, req *notificationv1.BatchSendNotificationRequest) (*notificationv1.BatchSendNotificationResponse, error) {
	success, fail, err := s.svc.BatchSendNotification(ctx, req.GetUserIds(), req.GetTenantId(), req.GetTemplateCode(), req.GetChannel(), req.GetParams())
	if err != nil {
		return nil, err
	}
	return &notificationv1.BatchSendNotificationResponse{SuccessCount: success, FailCount: fail}, nil
}

func (s *NotificationGRPCServer) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	ns, total, err := s.svc.ListNotifications(ctx, req.GetUserId(), req.GetChannel(), req.GetUnreadOnly(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	pbNs := make([]*notificationv1.Notification, 0, len(ns))
	for _, n := range ns {
		pbNs = append(pbNs, toNotificationDTO(n))
	}
	return &notificationv1.ListNotificationsResponse{Notifications: pbNs, Total: total}, nil
}

func (s *NotificationGRPCServer) MarkRead(ctx context.Context, req *notificationv1.MarkReadRequest) (*notificationv1.MarkReadResponse, error) {
	if err := s.svc.MarkRead(ctx, req.GetId(), req.GetUserId()); err != nil {
		return nil, err
	}
	return &notificationv1.MarkReadResponse{}, nil
}

func (s *NotificationGRPCServer) MarkAllRead(ctx context.Context, req *notificationv1.MarkAllReadRequest) (*notificationv1.MarkAllReadResponse, error) {
	if err := s.svc.MarkAllRead(ctx, req.GetUserId()); err != nil {
		return nil, err
	}
	return &notificationv1.MarkAllReadResponse{}, nil
}

func (s *NotificationGRPCServer) GetUnreadCount(ctx context.Context, req *notificationv1.GetUnreadCountRequest) (*notificationv1.GetUnreadCountResponse, error) {
	count, err := s.svc.GetUnreadCount(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &notificationv1.GetUnreadCountResponse{Count: count}, nil
}

func (s *NotificationGRPCServer) CreateTemplate(ctx context.Context, req *notificationv1.CreateTemplateRequest) (*notificationv1.CreateTemplateResponse, error) {
	t := req.GetTemplate()
	tmpl, err := s.svc.CreateTemplate(ctx, domain.NotificationTemplate{
		TenantID: t.GetTenantId(), Code: t.GetCode(), Channel: t.GetChannel(),
		Title: t.GetTitle(), Content: t.GetContent(), Status: t.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &notificationv1.CreateTemplateResponse{Id: tmpl.ID}, nil
}

func (s *NotificationGRPCServer) UpdateTemplate(ctx context.Context, req *notificationv1.UpdateTemplateRequest) (*notificationv1.UpdateTemplateResponse, error) {
	t := req.GetTemplate()
	if err := s.svc.UpdateTemplate(ctx, domain.NotificationTemplate{
		ID: t.GetId(), Title: t.GetTitle(), Content: t.GetContent(), Status: t.GetStatus(),
	}); err != nil {
		return nil, err
	}
	return &notificationv1.UpdateTemplateResponse{}, nil
}

func (s *NotificationGRPCServer) DeleteTemplate(ctx context.Context, req *notificationv1.DeleteTemplateRequest) (*notificationv1.DeleteTemplateResponse, error) {
	if err := s.svc.DeleteTemplate(ctx, req.GetId(), req.GetTenantId()); err != nil {
		return nil, err
	}
	return &notificationv1.DeleteTemplateResponse{}, nil
}

func (s *NotificationGRPCServer) ListTemplates(ctx context.Context, req *notificationv1.ListTemplatesRequest) (*notificationv1.ListTemplatesResponse, error) {
	templates, err := s.svc.ListTemplates(ctx, req.GetTenantId(), req.GetChannel())
	if err != nil {
		return nil, err
	}
	pbTemplates := make([]*notificationv1.NotificationTemplate, 0, len(templates))
	for _, t := range templates {
		pbTemplates = append(pbTemplates, toTemplateDTO(t))
	}
	return &notificationv1.ListTemplatesResponse{Templates: pbTemplates}, nil
}

func toNotificationDTO(n domain.Notification) *notificationv1.Notification {
	return &notificationv1.Notification{
		Id: n.ID, UserId: n.UserID, TenantId: n.TenantID, Channel: n.Channel,
		Title: n.Title, Content: n.Content, IsRead: n.IsRead, Status: n.Status,
		Ctime: timestamppb.New(n.Ctime),
	}
}

func toTemplateDTO(t domain.NotificationTemplate) *notificationv1.NotificationTemplate {
	return &notificationv1.NotificationTemplate{
		Id: t.ID, TenantId: t.TenantID, Code: t.Code, Channel: t.Channel,
		Title: t.Title, Content: t.Content, Status: t.Status,
		Ctime: timestamppb.New(t.Ctime),
	}
}
```

**Step 2: Verify build**

Run: `go build ./notification/grpc/...`
Expected: PASS

---

## Task 5: Kafka Consumers (Events)

**Files:**
- Create: `notification/events/types.go`
- Create: `notification/events/consumer.go`

**Step 1: Create event types**

Create `notification/events/types.go`:

```go
package events

const (
	TopicUserRegistered = "user_registered"
	TopicOrderPaid      = "order_paid"
	TopicOrderShipped   = "order_shipped"
	TopicInventoryAlert = "inventory_alert"
	TopicTenantApproved = "tenant_approved"
)

type UserRegisteredEvent struct {
	UserId   int64  `json:"user_id"`
	TenantId int64  `json:"tenant_id"`
	Phone    string `json:"phone"`
}

type OrderPaidEvent struct {
	OrderNo   string `json:"order_no"`
	PaymentNo string `json:"payment_no"`
	PaidAt    int64  `json:"paid_at"`
}

type OrderShippedEvent struct {
	OrderId     int64  `json:"order_id"`
	TenantId    int64  `json:"tenant_id"`
	CarrierCode string `json:"carrier_code"`
	CarrierName string `json:"carrier_name"`
	TrackingNo  string `json:"tracking_no"`
}

type InventoryAlertEvent struct {
	TenantID  int64 `json:"tenant_id"`
	SKUID     int64 `json:"sku_id"`
	Available int32 `json:"available"`
	Threshold int32 `json:"threshold"`
}

type TenantApprovedEvent struct {
	TenantId int64  `json:"tenant_id"`
	Name     string `json:"name"`
	PlanId   int64  `json:"plan_id"`
}
```

**Step 2: Create consumers**

Create `notification/events/consumer.go`:

```go
package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/saramax"
)

// ==================== UserRegisteredConsumer ====================

type UserRegisteredConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt UserRegisteredEvent) error
}

func NewUserRegisteredConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt UserRegisteredEvent) error) *UserRegisteredConsumer {
	return &UserRegisteredConsumer{client: client, l: l, handler: handler}
}

func (c *UserRegisteredConsumer) Start() error {
	h := saramax.NewHandler[UserRegisteredEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicUserRegistered}, h); err != nil {
				c.l.Error("消费 user_registered 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *UserRegisteredConsumer) Consume(msg *sarama.ConsumerMessage, evt UserRegisteredEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== OrderPaidConsumer ====================

type OrderPaidConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt OrderPaidEvent) error
}

func NewOrderPaidConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt OrderPaidEvent) error) *OrderPaidConsumer {
	return &OrderPaidConsumer{client: client, l: l, handler: handler}
}

func (c *OrderPaidConsumer) Start() error {
	h := saramax.NewHandler[OrderPaidEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicOrderPaid}, h); err != nil {
				c.l.Error("消费 order_paid 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderPaidConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderPaidEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== OrderShippedConsumer ====================

type OrderShippedConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt OrderShippedEvent) error
}

func NewOrderShippedConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt OrderShippedEvent) error) *OrderShippedConsumer {
	return &OrderShippedConsumer{client: client, l: l, handler: handler}
}

func (c *OrderShippedConsumer) Start() error {
	h := saramax.NewHandler[OrderShippedEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicOrderShipped}, h); err != nil {
				c.l.Error("消费 order_shipped 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderShippedConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderShippedEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== InventoryAlertConsumer ====================

type InventoryAlertConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt InventoryAlertEvent) error
}

func NewInventoryAlertConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt InventoryAlertEvent) error) *InventoryAlertConsumer {
	return &InventoryAlertConsumer{client: client, l: l, handler: handler}
}

func (c *InventoryAlertConsumer) Start() error {
	h := saramax.NewHandler[InventoryAlertEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicInventoryAlert}, h); err != nil {
				c.l.Error("消费 inventory_alert 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *InventoryAlertConsumer) Consume(msg *sarama.ConsumerMessage, evt InventoryAlertEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== TenantApprovedConsumer ====================

type TenantApprovedConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt TenantApprovedEvent) error
}

func NewTenantApprovedConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt TenantApprovedEvent) error) *TenantApprovedConsumer {
	return &TenantApprovedConsumer{client: client, l: l, handler: handler}
}

func (c *TenantApprovedConsumer) Start() error {
	h := saramax.NewHandler[TenantApprovedEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicTenantApproved}, h); err != nil {
				c.l.Error("消费 tenant_approved 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *TenantApprovedConsumer) Consume(msg *sarama.ConsumerMessage, evt TenantApprovedEvent) error {
	return c.handler(context.Background(), evt)
}
```

**Step 3: Verify build**

Run: `go build ./notification/events/...`
Expected: PASS

---

## Task 6: IoC + Wire + Config + Main

**Files:**
- Create: `notification/ioc/db.go`, `notification/ioc/redis.go`, `notification/ioc/logger.go`, `notification/ioc/grpc.go`, `notification/ioc/kafka.go`, `notification/ioc/provider.go`
- Create: `notification/wire.go`, `notification/app.go`, `notification/main.go`
- Create: `notification/config/dev.yaml`
- Generate: `notification/wire_gen.go`

**Step 1: Create IoC files**

Create `notification/ioc/db.go` — same as marketing pattern, use `dao.InitTables`, DSN from viper `db.dsn`.

Create `notification/ioc/redis.go` — same as marketing pattern, from viper `redis`.

Create `notification/ioc/logger.go` — same as marketing pattern.

Create `notification/ioc/grpc.go` — InitEtcdClient + InitGRPCServer (register NotificationGRPCServer, name="notification").

Create `notification/ioc/kafka.go`:
- `InitKafka()` — sarama.Client
- `InitConsumerGroup()` — group name "notification-svc"
- 5 consumer constructors: `NewUserRegisteredConsumer`, `NewOrderPaidConsumer`, `NewOrderShippedConsumer`, `NewInventoryAlertConsumer`, `NewTenantApprovedConsumer` — each wires event handler to call `svc.SendNotification` with appropriate template code. Some consumers (order_paid, order_shipped, inventory_alert, tenant_approved) need cross-service lookup for user_id which is left as TODO with log.
- `InitConsumers()` — returns `[]saramax.Consumer`

Create `notification/ioc/provider.go`:
- `InitSmsProvider()` — reads viper `sms` config, returns `provider.NewAliyunSmsProvider(...)`
- `InitEmailProvider()` — reads viper `email` config, returns `provider.NewSmtpEmailProvider(...)`

**Step 2: Create app.go** — App{Server *grpcx.Server, Consumers []saramax.Consumer}

**Step 3: Create wire.go** — thirdPartySet + notificationSet → InitApp()

**Step 4: Create main.go** — initViper + InitApp + start consumers + serve gRPC + graceful shutdown

**Step 5: Create config/dev.yaml** — db(mall_notification), redis(db:10), kafka, grpc(port:8091), etcd, sms(aliyun), email(smtp)

**Step 6: Generate wire_gen.go and verify**

Run: `cd notification && wire && cd .. && go build ./notification/... && go vet ./notification/...`
Expected: PASS

---

## Task 7: Consumer BFF — Notification Handler + Routes

**Files:**
- Create: `consumer-bff/handler/notification.go` — NotificationHandler + 4 methods (ListNotifications, GetUnreadCount, MarkRead, MarkAllRead)
- Modify: `consumer-bff/ioc/grpc.go` — add `InitNotificationClient`
- Modify: `consumer-bff/ioc/gin.go` — add `notificationHandler` param + 4 routes in auth group
- Modify: `consumer-bff/wire.go` — add `ioc.InitNotificationClient` to thirdPartySet, `handler.NewNotificationHandler` to handlerSet
- Regenerate: `consumer-bff/wire_gen.go`

Routes:
```
auth.GET("/notifications", ginx.WrapQuery[handler.ListNotificationsReq](l, notificationHandler.ListNotifications))
auth.GET("/notifications/unread-count", notificationHandler.GetUnreadCount)
auth.PUT("/notifications/:id/read", notificationHandler.MarkRead)
auth.PUT("/notifications/read-all", notificationHandler.MarkAllRead)
```

Verify: `cd consumer-bff && wire && cd .. && go build ./consumer-bff/...`

---

## Task 8: Merchant BFF — Notification Handler + Routes

**Files:**
- Create: `merchant-bff/handler/notification.go` — NotificationHandler + 4 methods (identical to consumer-bff)
- Modify: `merchant-bff/ioc/grpc.go` — add `InitNotificationClient`
- Modify: `merchant-bff/ioc/gin.go` — add `notificationHandler` param + 4 routes in auth group
- Modify: `merchant-bff/wire.go` — add providers
- Regenerate: `merchant-bff/wire_gen.go`

Same routes and handler code as consumer-bff.

Verify: `cd merchant-bff && wire && cd .. && go build ./merchant-bff/...`

---

## Task 9: Final Verification

Run: `go build ./notification/... && go build ./consumer-bff/... && go build ./merchant-bff/... && go vet ./notification/... && go vet ./consumer-bff/... && go vet ./merchant-bff/...`
Expected: ALL PASS
