# Payment Service 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 payment-svc 支付微服务，提供 7 个 gRPC RPC，支持 mock/wechat/alipay 渠道抽象，HandleNotify 布隆过滤器幂等，支付成功后发送 order_paid Kafka 事件。

**Architecture:** 遵循 order-svc 分层模式（domain → dao → cache → repository → service → grpc → ioc → wire）。Channel 接口抽象支付渠道，MockChannel 完整实现，WechatChannel/AlipayChannel 为桩代码。payment-svc 不消费任何 Kafka 事件，仅生产 order_paid。

**Tech Stack:** Go, gRPC, GORM/MySQL, Redis, Kafka (sarama), Wire DI, etcd, Snowflake ID, emo/idempotent

---

## 参考文件

| 文件 | 用途 |
|------|------|
| `docs/plans/2026-03-09-payment-svc-design.md` | 设计文档 |
| `api/proto/payment/v1/payment.proto` | Proto 定义（7 RPC, PaymentOrder, RefundRecord） |
| `api/proto/gen/payment/v1/payment_grpc.pb.go` | gRPC 生成代码 |
| `order/domain/order.go` | Domain 模式参考 |
| `order/repository/dao/order.go` | DAO 模式参考 |
| `order/repository/cache/order.go` | Cache 模式参考 |
| `order/repository/order.go` | Repository 模式参考 |
| `order/service/order.go` | Service 模式参考 |
| `order/grpc/order.go` | gRPC Handler 模式参考 |
| `order/ioc/*.go` | IoC 模式参考 |
| `order/wire.go` | Wire DI 模式参考 |
| `order/config/dev.yaml` | 配置模式参考 |

---

## Task 1: Domain 层

**Files:**
- Create: `payment/domain/payment.go`

```go
package domain

import "time"

// PaymentOrder 支付单聚合根
type PaymentOrder struct {
	ID             int64
	TenantID       int64
	PaymentNo      string
	OrderID        int64
	OrderNo        string
	Channel        string // mock / wechat / alipay
	Amount         int64  // 分
	Status         PaymentStatus
	ChannelTradeNo string
	PayTime        int64 // 毫秒时间戳
	ExpireTime     int64 // 毫秒时间戳
	NotifyUrl      string
	Ctime          time.Time
	Utime          time.Time
}

type PaymentStatus int32

const (
	PaymentStatusPending   PaymentStatus = 1 // 待支付
	PaymentStatusPaying    PaymentStatus = 2 // 支付中
	PaymentStatusPaid      PaymentStatus = 3 // 已支付
	PaymentStatusClosed    PaymentStatus = 4 // 已关闭
	PaymentStatusRefunding PaymentStatus = 5 // 退款中
	PaymentStatusRefunded  PaymentStatus = 6 // 已退款
)

// RefundRecord 退款记录
type RefundRecord struct {
	ID              int64
	TenantID        int64
	PaymentNo       string
	RefundNo        string
	Channel         string
	Amount          int64 // 分
	Status          RefundStatus
	ChannelRefundNo string
	Ctime           time.Time
	Utime           time.Time
}

type RefundStatus int32

const (
	RefundStatusRefunding  RefundStatus = 1 // 退款中
	RefundStatusRefunded   RefundStatus = 2 // 已退款
	RefundStatusFailed     RefundStatus = 3 // 退款失败
)
```

---

## Task 2: DAO 层

**Files:**
- Create: `payment/repository/dao/payment.go`
- Create: `payment/repository/dao/init.go`

### 2.1 payment/repository/dao/payment.go

```go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type PaymentOrderModel struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	TenantId       int64  `gorm:"index:idx_tenant_status"`
	PaymentNo      string `gorm:"type:varchar(64);uniqueIndex:uk_payment_no"`
	OrderId        int64
	OrderNo        string `gorm:"type:varchar(64);index:idx_order_no"`
	Channel        string `gorm:"type:varchar(32)"`
	Amount         int64
	Status         int32 `gorm:"index:idx_tenant_status"`
	ChannelTradeNo string `gorm:"type:varchar(128)"`
	PayTime        int64
	ExpireTime     int64
	NotifyUrl      string `gorm:"type:varchar(512)"`
	Ctime          int64
	Utime          int64
}

func (PaymentOrderModel) TableName() string { return "payment_orders" }

type RefundRecordModel struct {
	ID              int64  `gorm:"primaryKey;autoIncrement"`
	TenantId        int64
	PaymentNo       string `gorm:"type:varchar(64);index:idx_payment_no"`
	RefundNo        string `gorm:"type:varchar(64);uniqueIndex:uk_refund_no"`
	Channel         string `gorm:"type:varchar(32)"`
	Amount          int64
	Status          int32
	ChannelRefundNo string `gorm:"type:varchar(128)"`
	Ctime           int64
	Utime           int64
}

func (RefundRecordModel) TableName() string { return "refund_records" }

type PaymentDAO interface {
	CreatePayment(ctx context.Context, payment PaymentOrderModel) (PaymentOrderModel, error)
	FindByPaymentNo(ctx context.Context, paymentNo string) (PaymentOrderModel, error)
	FindByOrderNo(ctx context.Context, orderNo string) (PaymentOrderModel, error)
	UpdateStatus(ctx context.Context, paymentNo string, oldStatus, newStatus int32, updates map[string]any) error
	ListPayments(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]PaymentOrderModel, int64, error)
	CreateRefund(ctx context.Context, refund RefundRecordModel) error
	FindRefundByNo(ctx context.Context, refundNo string) (RefundRecordModel, error)
	UpdateRefundStatus(ctx context.Context, refundNo string, status int32, updates map[string]any) error
	GetDB() *gorm.DB
}

type GORMPaymentDAO struct {
	db *gorm.DB
}

func NewPaymentDAO(db *gorm.DB) PaymentDAO {
	return &GORMPaymentDAO{db: db}
}

func (d *GORMPaymentDAO) GetDB() *gorm.DB { return d.db }

func (d *GORMPaymentDAO) CreatePayment(ctx context.Context, payment PaymentOrderModel) (PaymentOrderModel, error) {
	now := time.Now().UnixMilli()
	payment.Ctime = now
	payment.Utime = now
	err := d.db.WithContext(ctx).Create(&payment).Error
	return payment, err
}

func (d *GORMPaymentDAO) FindByPaymentNo(ctx context.Context, paymentNo string) (PaymentOrderModel, error) {
	var payment PaymentOrderModel
	err := d.db.WithContext(ctx).Where("payment_no = ?", paymentNo).First(&payment).Error
	return payment, err
}

func (d *GORMPaymentDAO) FindByOrderNo(ctx context.Context, orderNo string) (PaymentOrderModel, error) {
	var payment PaymentOrderModel
	err := d.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&payment).Error
	return payment, err
}

func (d *GORMPaymentDAO) UpdateStatus(ctx context.Context, paymentNo string, oldStatus, newStatus int32, updates map[string]any) error {
	updates["status"] = newStatus
	updates["utime"] = time.Now().UnixMilli()
	result := d.db.WithContext(ctx).Model(&PaymentOrderModel{}).
		Where("payment_no = ? AND status = ?", paymentNo, oldStatus).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *GORMPaymentDAO) ListPayments(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]PaymentOrderModel, int64, error) {
	var payments []PaymentOrderModel
	var total int64
	query := d.db.WithContext(ctx).Model(&PaymentOrderModel{})
	if tenantId > 0 {
		query = query.Where("tenant_id = ?", tenantId)
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&payments).Error
	return payments, total, err
}

func (d *GORMPaymentDAO) CreateRefund(ctx context.Context, refund RefundRecordModel) error {
	now := time.Now().UnixMilli()
	refund.Ctime = now
	refund.Utime = now
	return d.db.WithContext(ctx).Create(&refund).Error
}

func (d *GORMPaymentDAO) FindRefundByNo(ctx context.Context, refundNo string) (RefundRecordModel, error) {
	var refund RefundRecordModel
	err := d.db.WithContext(ctx).Where("refund_no = ?", refundNo).First(&refund).Error
	return refund, err
}

func (d *GORMPaymentDAO) UpdateRefundStatus(ctx context.Context, refundNo string, status int32, updates map[string]any) error {
	updates["status"] = status
	updates["utime"] = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Model(&RefundRecordModel{}).
		Where("refund_no = ?", refundNo).Updates(updates).Error
}
```

### 2.2 payment/repository/dao/init.go

```go
package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&PaymentOrderModel{},
		&RefundRecordModel{},
	)
}
```

---

## Task 3: Cache 层

**Files:**
- Create: `payment/repository/cache/payment.go`

```go
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type PaymentCache interface {
	GetPayment(ctx context.Context, paymentNo string) ([]byte, error)
	SetPayment(ctx context.Context, paymentNo string, data []byte) error
	DeletePayment(ctx context.Context, paymentNo string) error
}

type RedisPaymentCache struct {
	client redis.Cmdable
}

func NewPaymentCache(client redis.Cmdable) PaymentCache {
	return &RedisPaymentCache{client: client}
}

func paymentKey(paymentNo string) string {
	return fmt.Sprintf("payment:info:%s", paymentNo)
}

func (c *RedisPaymentCache) GetPayment(ctx context.Context, paymentNo string) ([]byte, error) {
	return c.client.Get(ctx, paymentKey(paymentNo)).Bytes()
}

func (c *RedisPaymentCache) SetPayment(ctx context.Context, paymentNo string, data []byte) error {
	return c.client.Set(ctx, paymentKey(paymentNo), data, 15*time.Minute).Err()
}

func (c *RedisPaymentCache) DeletePayment(ctx context.Context, paymentNo string) error {
	return c.client.Del(ctx, paymentKey(paymentNo)).Err()
}
```

---

## Task 4: Repository 层

**Files:**
- Create: `payment/repository/payment.go`

```go
package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/payment/repository/cache"
	"github.com/rermrf/mall/payment/repository/dao"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment domain.PaymentOrder) (domain.PaymentOrder, error)
	FindByPaymentNo(ctx context.Context, paymentNo string) (domain.PaymentOrder, error)
	FindByOrderNo(ctx context.Context, orderNo string) (domain.PaymentOrder, error)
	UpdateStatus(ctx context.Context, paymentNo string, oldStatus, newStatus domain.PaymentStatus, updates map[string]any) error
	ListPayments(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.PaymentOrder, int64, error)
	CreateRefund(ctx context.Context, refund domain.RefundRecord) error
	FindRefundByNo(ctx context.Context, refundNo string) (domain.RefundRecord, error)
	UpdateRefundStatus(ctx context.Context, refundNo string, status domain.RefundStatus, updates map[string]any) error
}

type paymentRepository struct {
	dao   dao.PaymentDAO
	cache cache.PaymentCache
}

func NewPaymentRepository(d dao.PaymentDAO, c cache.PaymentCache) PaymentRepository {
	return &paymentRepository{dao: d, cache: c}
}

func (r *paymentRepository) CreatePayment(ctx context.Context, payment domain.PaymentOrder) (domain.PaymentOrder, error) {
	model := r.toModel(payment)
	model, err := r.dao.CreatePayment(ctx, model)
	if err != nil {
		return domain.PaymentOrder{}, err
	}
	payment.ID = model.ID
	payment.Ctime = time.UnixMilli(model.Ctime)
	payment.Utime = time.UnixMilli(model.Utime)
	return payment, nil
}

func (r *paymentRepository) FindByPaymentNo(ctx context.Context, paymentNo string) (domain.PaymentOrder, error) {
	// 尝试缓存
	data, err := r.cache.GetPayment(ctx, paymentNo)
	if err == nil {
		var payment domain.PaymentOrder
		if json.Unmarshal(data, &payment) == nil {
			return payment, nil
		}
	}
	if err != nil && err != redis.Nil {
		// Redis 错误只记录，不阻塞
	}
	model, err := r.dao.FindByPaymentNo(ctx, paymentNo)
	if err != nil {
		return domain.PaymentOrder{}, err
	}
	payment := r.toDomain(model)
	r.setCache(ctx, paymentNo, payment)
	return payment, nil
}

func (r *paymentRepository) FindByOrderNo(ctx context.Context, orderNo string) (domain.PaymentOrder, error) {
	model, err := r.dao.FindByOrderNo(ctx, orderNo)
	if err != nil {
		return domain.PaymentOrder{}, err
	}
	return r.toDomain(model), nil
}

func (r *paymentRepository) UpdateStatus(ctx context.Context, paymentNo string, oldStatus, newStatus domain.PaymentStatus, updates map[string]any) error {
	err := r.dao.UpdateStatus(ctx, paymentNo, int32(oldStatus), int32(newStatus), updates)
	if err != nil {
		return err
	}
	_ = r.cache.DeletePayment(ctx, paymentNo)
	return nil
}

func (r *paymentRepository) ListPayments(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.PaymentOrder, int64, error) {
	offset := int((page - 1) * pageSize)
	models, total, err := r.dao.ListPayments(ctx, tenantId, status, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	payments := make([]domain.PaymentOrder, 0, len(models))
	for _, m := range models {
		payments = append(payments, r.toDomain(m))
	}
	return payments, total, nil
}

func (r *paymentRepository) CreateRefund(ctx context.Context, refund domain.RefundRecord) error {
	return r.dao.CreateRefund(ctx, dao.RefundRecordModel{
		TenantId:  refund.TenantID,
		PaymentNo: refund.PaymentNo,
		RefundNo:  refund.RefundNo,
		Channel:   refund.Channel,
		Amount:    refund.Amount,
		Status:    int32(refund.Status),
	})
}

func (r *paymentRepository) FindRefundByNo(ctx context.Context, refundNo string) (domain.RefundRecord, error) {
	model, err := r.dao.FindRefundByNo(ctx, refundNo)
	if err != nil {
		return domain.RefundRecord{}, err
	}
	return r.toDomainRefund(model), nil
}

func (r *paymentRepository) UpdateRefundStatus(ctx context.Context, refundNo string, status domain.RefundStatus, updates map[string]any) error {
	return r.dao.UpdateRefundStatus(ctx, refundNo, int32(status), updates)
}

func (r *paymentRepository) setCache(ctx context.Context, paymentNo string, payment domain.PaymentOrder) {
	data, _ := json.Marshal(payment)
	_ = r.cache.SetPayment(ctx, paymentNo, data)
}

func (r *paymentRepository) toModel(p domain.PaymentOrder) dao.PaymentOrderModel {
	return dao.PaymentOrderModel{
		TenantId:       p.TenantID,
		PaymentNo:      p.PaymentNo,
		OrderId:        p.OrderID,
		OrderNo:        p.OrderNo,
		Channel:        p.Channel,
		Amount:         p.Amount,
		Status:         int32(p.Status),
		ChannelTradeNo: p.ChannelTradeNo,
		PayTime:        p.PayTime,
		ExpireTime:     p.ExpireTime,
		NotifyUrl:      p.NotifyUrl,
	}
}

func (r *paymentRepository) toDomain(m dao.PaymentOrderModel) domain.PaymentOrder {
	return domain.PaymentOrder{
		ID:             m.ID,
		TenantID:       m.TenantId,
		PaymentNo:      m.PaymentNo,
		OrderID:        m.OrderId,
		OrderNo:        m.OrderNo,
		Channel:        m.Channel,
		Amount:         m.Amount,
		Status:         domain.PaymentStatus(m.Status),
		ChannelTradeNo: m.ChannelTradeNo,
		PayTime:        m.PayTime,
		ExpireTime:     m.ExpireTime,
		NotifyUrl:      m.NotifyUrl,
		Ctime:          time.UnixMilli(m.Ctime),
		Utime:          time.UnixMilli(m.Utime),
	}
}

func (r *paymentRepository) toDomainRefund(m dao.RefundRecordModel) domain.RefundRecord {
	return domain.RefundRecord{
		ID:              m.ID,
		TenantID:        m.TenantId,
		PaymentNo:       m.PaymentNo,
		RefundNo:        m.RefundNo,
		Channel:         m.Channel,
		Amount:          m.Amount,
		Status:          domain.RefundStatus(m.Status),
		ChannelRefundNo: m.ChannelRefundNo,
		Ctime:           time.UnixMilli(m.Ctime),
		Utime:           time.UnixMilli(m.Utime),
	}
}
```

---

## Task 5: Channel 渠道抽象

**Files:**
- Create: `payment/service/channel/types.go`
- Create: `payment/service/channel/mock.go`
- Create: `payment/service/channel/wechat.go`
- Create: `payment/service/channel/alipay.go`

### 5.1 payment/service/channel/types.go

```go
package channel

import (
	"context"

	"github.com/rermrf/mall/payment/domain"
)

// Channel 支付渠道抽象接口
type Channel interface {
	// Pay 发起支付，返回渠道交易号和支付链接
	Pay(ctx context.Context, payment domain.PaymentOrder) (channelTradeNo string, payUrl string, err error)
	// QueryPayment 查询支付状态
	QueryPayment(ctx context.Context, paymentNo string) (status int32, channelTradeNo string, err error)
	// Refund 发起退款
	Refund(ctx context.Context, refund domain.RefundRecord) (channelRefundNo string, err error)
	// QueryRefund 查询退款状态
	QueryRefund(ctx context.Context, refundNo string) (status int32, channelRefundNo string, err error)
	// VerifyNotify 验证支付回调，返回支付单号和渠道交易号
	VerifyNotify(ctx context.Context, data map[string]string) (paymentNo string, channelTradeNo string, err error)
}
```

### 5.2 payment/service/channel/mock.go

```go
package channel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/pkg/snowflake"
)

type MockChannel struct {
	node *snowflake.Node
}

func NewMockChannel(node *snowflake.Node) *MockChannel {
	return &MockChannel{node: node}
}

func (c *MockChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	channelTradeNo := fmt.Sprintf("MOCK_%d", c.node.Generate())
	return channelTradeNo, "", nil
}

func (c *MockChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	// mock 渠道直接返回已支付
	return int32(domain.PaymentStatusPaid), fmt.Sprintf("MOCK_QUERY_%s", paymentNo), nil
}

func (c *MockChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	channelRefundNo := fmt.Sprintf("MOCK_REFUND_%d", c.node.Generate())
	return channelRefundNo, nil
}

func (c *MockChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	return int32(domain.RefundStatusRefunded), fmt.Sprintf("MOCK_REFUND_QUERY_%s", refundNo), nil
}

func (c *MockChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	paymentNo, ok := data["payment_no"]
	if !ok {
		return "", "", fmt.Errorf("缺少 payment_no")
	}
	channelTradeNo, ok := data["channel_trade_no"]
	if !ok {
		return "", "", fmt.Errorf("缺少 channel_trade_no")
	}
	return paymentNo, channelTradeNo, nil
}

// BuildMockNotifyBody 构造 mock 回调报文（用于测试）
func BuildMockNotifyBody(paymentNo, channelTradeNo string) string {
	data, _ := json.Marshal(map[string]string{
		"payment_no":       paymentNo,
		"channel_trade_no": channelTradeNo,
	})
	return string(data)
}
```

### 5.3 payment/service/channel/wechat.go

```go
package channel

import (
	"context"
	"fmt"

	"github.com/rermrf/mall/payment/domain"
)

type WechatChannel struct{}

func NewWechatChannel() *WechatChannel {
	return &WechatChannel{}
}

func (c *WechatChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	return "", "", fmt.Errorf("微信支付渠道暂未实现")
}

func (c *WechatChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	return 0, "", fmt.Errorf("微信支付渠道暂未实现")
}

func (c *WechatChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	return "", fmt.Errorf("微信支付渠道暂未实现")
}

func (c *WechatChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	return 0, "", fmt.Errorf("微信支付渠道暂未实现")
}

func (c *WechatChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	return "", "", fmt.Errorf("微信支付渠道暂未实现")
}
```

### 5.4 payment/service/channel/alipay.go

```go
package channel

import (
	"context"
	"fmt"

	"github.com/rermrf/mall/payment/domain"
)

type AlipayChannel struct{}

func NewAlipayChannel() *AlipayChannel {
	return &AlipayChannel{}
}

func (c *AlipayChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	return "", "", fmt.Errorf("支付宝渠道暂未实现")
}

func (c *AlipayChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	return 0, "", fmt.Errorf("支付宝渠道暂未实现")
}

func (c *AlipayChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	return "", fmt.Errorf("支付宝渠道暂未实现")
}

func (c *AlipayChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	return 0, "", fmt.Errorf("支付宝渠道暂未实现")
}

func (c *AlipayChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	return "", "", fmt.Errorf("支付宝渠道暂未实现")
}
```

---

## Task 6: Events 层

**Files:**
- Create: `payment/events/types.go`
- Create: `payment/events/producer.go`

### 6.1 payment/events/types.go

```go
package events

// OrderPaidEvent 支付成功事件（发送到 order_paid topic，order-svc 消费）
type OrderPaidEvent struct {
	OrderNo   string `json:"order_no"`
	PaymentNo string `json:"payment_no"`
	PaidAt    int64  `json:"paid_at"` // 毫秒时间戳
}

const (
	TopicOrderPaid = "order_paid"
)
```

### 6.2 payment/events/producer.go

```go
package events

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

type Producer interface {
	ProduceOrderPaid(ctx context.Context, evt OrderPaidEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceOrderPaid(ctx context.Context, evt OrderPaidEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicOrderPaid,
		Key:   sarama.StringEncoder(evt.OrderNo),
		Value: sarama.ByteEncoder(data),
	})
	return err
}
```

---

## Task 7: Service 层

**Files:**
- Create: `payment/service/payment.go`

```go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rermrf/emo/idempotent"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/payment/events"
	"github.com/rermrf/mall/payment/repository"
	"github.com/rermrf/mall/payment/service/channel"
	"github.com/rermrf/mall/pkg/snowflake"
)

type PaymentService interface {
	CreatePayment(ctx context.Context, tenantId int64, orderId int64, orderNo, ch string, amount int64) (string, string, error)
	GetPayment(ctx context.Context, paymentNo string) (domain.PaymentOrder, error)
	HandleNotify(ctx context.Context, ch string, notifyBody string) (bool, error)
	ClosePayment(ctx context.Context, paymentNo string) error
	Refund(ctx context.Context, paymentNo string, amount int64, reason string) (string, error)
	GetRefund(ctx context.Context, refundNo string) (domain.RefundRecord, error)
	ListPayments(ctx context.Context, tenantId int64, status, page, pageSize int32) ([]domain.PaymentOrder, int64, error)
}

type paymentService struct {
	repo           repository.PaymentRepository
	producer       events.Producer
	idempotencySvc idempotent.IdempotencyService
	node           *snowflake.Node
	channels       map[string]channel.Channel
	l              logger.Logger
}

func NewPaymentService(
	repo repository.PaymentRepository,
	producer events.Producer,
	idempotencySvc idempotent.IdempotencyService,
	node *snowflake.Node,
	mockCh *channel.MockChannel,
	l logger.Logger,
) PaymentService {
	channels := map[string]channel.Channel{
		"mock":   mockCh,
		"wechat": channel.NewWechatChannel(),
		"alipay": channel.NewAlipayChannel(),
	}
	return &paymentService{
		repo:           repo,
		producer:       producer,
		idempotencySvc: idempotencySvc,
		node:           node,
		channels:       channels,
		l:              l,
	}
}

func (s *paymentService) getChannel(ch string) (channel.Channel, error) {
	c, ok := s.channels[ch]
	if !ok {
		return nil, fmt.Errorf("不支持的支付渠道: %s", ch)
	}
	return c, nil
}

func (s *paymentService) CreatePayment(ctx context.Context, tenantId int64, orderId int64, orderNo, ch string, amount int64) (string, string, error) {
	c, err := s.getChannel(ch)
	if err != nil {
		return "", "", err
	}
	paymentNo := fmt.Sprintf("P%d", s.node.Generate())
	payment := domain.PaymentOrder{
		TenantID:   tenantId,
		PaymentNo:  paymentNo,
		OrderID:    orderId,
		OrderNo:    orderNo,
		Channel:    ch,
		Amount:     amount,
		Status:     domain.PaymentStatusPending,
		ExpireTime: time.Now().Add(30 * time.Minute).UnixMilli(),
	}
	payment, err = s.repo.CreatePayment(ctx, payment)
	if err != nil {
		return "", "", fmt.Errorf("创建支付单失败: %w", err)
	}
	// 调用渠道发起支付
	channelTradeNo, payUrl, err := c.Pay(ctx, payment)
	if err != nil {
		return "", "", fmt.Errorf("渠道发起支付失败: %w", err)
	}
	// 更新渠道交易号
	if channelTradeNo != "" {
		_ = s.repo.UpdateStatus(ctx, paymentNo, domain.PaymentStatusPending, domain.PaymentStatusPending, map[string]any{
			"channel_trade_no": channelTradeNo,
		})
	}
	return paymentNo, payUrl, nil
}

func (s *paymentService) GetPayment(ctx context.Context, paymentNo string) (domain.PaymentOrder, error) {
	return s.repo.FindByPaymentNo(ctx, paymentNo)
}

func (s *paymentService) HandleNotify(ctx context.Context, ch string, notifyBody string) (bool, error) {
	c, err := s.getChannel(ch)
	if err != nil {
		return false, err
	}
	// 解析回调数据
	var data map[string]string
	if err := json.Unmarshal([]byte(notifyBody), &data); err != nil {
		return false, fmt.Errorf("解析回调报文失败: %w", err)
	}
	// 验证回调
	paymentNo, channelTradeNo, err := c.VerifyNotify(ctx, data)
	if err != nil {
		return false, fmt.Errorf("验证回调失败: %w", err)
	}
	// 布隆过滤器幂等检查
	bloomKey := fmt.Sprintf("payment:notify:%s", paymentNo)
	exists, err := s.idempotencySvc.Exists(ctx, bloomKey)
	if err != nil {
		s.l.Error("幂等检查失败", logger.Error(err))
	}
	if exists {
		// 可能已处理，查询确认
		payment, dbErr := s.repo.FindByPaymentNo(ctx, paymentNo)
		if dbErr == nil && payment.Status == domain.PaymentStatusPaid {
			return true, nil // 已处理
		}
	}
	// 查询支付单
	payment, err := s.repo.FindByPaymentNo(ctx, paymentNo)
	if err != nil {
		return false, fmt.Errorf("支付单不存在: %w", err)
	}
	if payment.Status == domain.PaymentStatusPaid {
		return true, nil // 幂等
	}
	if payment.Status != domain.PaymentStatusPending && payment.Status != domain.PaymentStatusPaying {
		return false, fmt.Errorf("支付单状态不允许回调: %d", payment.Status)
	}
	// 更新状态为已支付
	now := time.Now().UnixMilli()
	err = s.repo.UpdateStatus(ctx, paymentNo, payment.Status, domain.PaymentStatusPaid, map[string]any{
		"channel_trade_no": channelTradeNo,
		"pay_time":         now,
	})
	if err != nil {
		return false, fmt.Errorf("更新支付状态失败: %w", err)
	}
	// 标记布隆过滤器
	_ = s.idempotencySvc.Mark(ctx, bloomKey)
	// 发送 order_paid 事件
	if produceErr := s.producer.ProduceOrderPaid(ctx, events.OrderPaidEvent{
		OrderNo:   payment.OrderNo,
		PaymentNo: paymentNo,
		PaidAt:    now,
	}); produceErr != nil {
		s.l.Error("发送 order_paid 事件失败", logger.String("paymentNo", paymentNo), logger.Error(produceErr))
	}
	return true, nil
}

func (s *paymentService) ClosePayment(ctx context.Context, paymentNo string) error {
	payment, err := s.repo.FindByPaymentNo(ctx, paymentNo)
	if err != nil {
		return fmt.Errorf("支付单不存在: %w", err)
	}
	if payment.Status != domain.PaymentStatusPending && payment.Status != domain.PaymentStatusPaying {
		return fmt.Errorf("当前状态不允许关闭: %d", payment.Status)
	}
	return s.repo.UpdateStatus(ctx, paymentNo, payment.Status, domain.PaymentStatusClosed, map[string]any{})
}

func (s *paymentService) Refund(ctx context.Context, paymentNo string, amount int64, reason string) (string, error) {
	payment, err := s.repo.FindByPaymentNo(ctx, paymentNo)
	if err != nil {
		return "", fmt.Errorf("支付单不存在: %w", err)
	}
	if payment.Status != domain.PaymentStatusPaid {
		return "", fmt.Errorf("当前状态不允许退款: %d", payment.Status)
	}
	if amount > payment.Amount {
		return "", fmt.Errorf("退款金额超出支付金额")
	}
	c, err := s.getChannel(payment.Channel)
	if err != nil {
		return "", err
	}
	refundNo := fmt.Sprintf("R%d", s.node.Generate())
	refund := domain.RefundRecord{
		TenantID:  payment.TenantID,
		PaymentNo: paymentNo,
		RefundNo:  refundNo,
		Channel:   payment.Channel,
		Amount:    amount,
		Status:    domain.RefundStatusRefunding,
	}
	if err := s.repo.CreateRefund(ctx, refund); err != nil {
		return "", fmt.Errorf("创建退款记录失败: %w", err)
	}
	// 调用渠道退款
	channelRefundNo, err := c.Refund(ctx, refund)
	if err != nil {
		// 更新退款记录为失败
		_ = s.repo.UpdateRefundStatus(ctx, refundNo, domain.RefundStatusFailed, map[string]any{})
		return "", fmt.Errorf("渠道退款失败: %w", err)
	}
	// 更新退款记录
	_ = s.repo.UpdateRefundStatus(ctx, refundNo, domain.RefundStatusRefunded, map[string]any{
		"channel_refund_no": channelRefundNo,
	})
	// 更新支付单状态为退款中
	_ = s.repo.UpdateStatus(ctx, paymentNo, domain.PaymentStatusPaid, domain.PaymentStatusRefunding, map[string]any{})
	return refundNo, nil
}

func (s *paymentService) GetRefund(ctx context.Context, refundNo string) (domain.RefundRecord, error) {
	return s.repo.FindRefundByNo(ctx, refundNo)
}

func (s *paymentService) ListPayments(ctx context.Context, tenantId int64, status, page, pageSize int32) ([]domain.PaymentOrder, int64, error) {
	return s.repo.ListPayments(ctx, tenantId, status, page, pageSize)
}
```

---

## Task 8: gRPC Handler

**Files:**
- Create: `payment/grpc/payment.go`

```go
package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/payment/service"
)

type PaymentGRPCServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	svc service.PaymentService
}

func NewPaymentGRPCServer(svc service.PaymentService) *PaymentGRPCServer {
	return &PaymentGRPCServer{svc: svc}
}

func (s *PaymentGRPCServer) Register(server *grpc.Server) {
	paymentv1.RegisterPaymentServiceServer(server, s)
}

func (s *PaymentGRPCServer) CreatePayment(ctx context.Context, req *paymentv1.CreatePaymentRequest) (*paymentv1.CreatePaymentResponse, error) {
	paymentNo, payUrl, err := s.svc.CreatePayment(ctx, req.GetTenantId(), req.GetOrderId(), req.GetOrderNo(), req.GetChannel(), req.GetAmount())
	if err != nil {
		return nil, err
	}
	return &paymentv1.CreatePaymentResponse{PaymentNo: paymentNo, PayUrl: payUrl}, nil
}

func (s *PaymentGRPCServer) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.GetPaymentResponse, error) {
	payment, err := s.svc.GetPayment(ctx, req.GetPaymentNo())
	if err != nil {
		return nil, err
	}
	return &paymentv1.GetPaymentResponse{Payment: s.toPaymentDTO(payment)}, nil
}

func (s *PaymentGRPCServer) HandleNotify(ctx context.Context, req *paymentv1.HandleNotifyRequest) (*paymentv1.HandleNotifyResponse, error) {
	success, err := s.svc.HandleNotify(ctx, req.GetChannel(), req.GetNotifyBody())
	if err != nil {
		return nil, err
	}
	return &paymentv1.HandleNotifyResponse{Success: success}, nil
}

func (s *PaymentGRPCServer) ClosePayment(ctx context.Context, req *paymentv1.ClosePaymentRequest) (*paymentv1.ClosePaymentResponse, error) {
	err := s.svc.ClosePayment(ctx, req.GetPaymentNo())
	if err != nil {
		return nil, err
	}
	return &paymentv1.ClosePaymentResponse{}, nil
}

func (s *PaymentGRPCServer) Refund(ctx context.Context, req *paymentv1.RefundRequest) (*paymentv1.RefundResponse, error) {
	refundNo, err := s.svc.Refund(ctx, req.GetPaymentNo(), req.GetAmount(), req.GetReason())
	if err != nil {
		return nil, err
	}
	return &paymentv1.RefundResponse{RefundNo: refundNo}, nil
}

func (s *PaymentGRPCServer) GetRefund(ctx context.Context, req *paymentv1.GetRefundRequest) (*paymentv1.GetRefundResponse, error) {
	refund, err := s.svc.GetRefund(ctx, req.GetRefundNo())
	if err != nil {
		return nil, err
	}
	return &paymentv1.GetRefundResponse{Refund: s.toRefundDTO(refund)}, nil
}

func (s *PaymentGRPCServer) ListPayments(ctx context.Context, req *paymentv1.ListPaymentsRequest) (*paymentv1.ListPaymentsResponse, error) {
	payments, total, err := s.svc.ListPayments(ctx, req.GetTenantId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*paymentv1.PaymentOrder, 0, len(payments))
	for _, p := range payments {
		dtos = append(dtos, s.toPaymentDTO(p))
	}
	return &paymentv1.ListPaymentsResponse{Payments: dtos, Total: total}, nil
}

func (s *PaymentGRPCServer) toPaymentDTO(p domain.PaymentOrder) *paymentv1.PaymentOrder {
	return &paymentv1.PaymentOrder{
		Id:             p.ID,
		TenantId:       p.TenantID,
		PaymentNo:      p.PaymentNo,
		OrderId:        p.OrderID,
		OrderNo:        p.OrderNo,
		Channel:        p.Channel,
		Amount:         p.Amount,
		Status:         int32(p.Status),
		ChannelTradeNo: p.ChannelTradeNo,
		PayTime:        p.PayTime,
		ExpireTime:     p.ExpireTime,
		NotifyUrl:      p.NotifyUrl,
		Ctime:          timestamppb.New(p.Ctime),
		Utime:          timestamppb.New(p.Utime),
	}
}

func (s *PaymentGRPCServer) toRefundDTO(r domain.RefundRecord) *paymentv1.RefundRecord {
	return &paymentv1.RefundRecord{
		Id:              r.ID,
		TenantId:        r.TenantID,
		PaymentNo:       r.PaymentNo,
		RefundNo:        r.RefundNo,
		Channel:         r.Channel,
		Amount:          r.Amount,
		Status:          int32(r.Status),
		ChannelRefundNo: r.ChannelRefundNo,
		Ctime:           timestamppb.New(r.Ctime),
	}
}
```

---

## Task 9: IoC + Wire + Config + Main

**Files:**
- Create: `payment/ioc/db.go`
- Create: `payment/ioc/redis.go`
- Create: `payment/ioc/kafka.go`
- Create: `payment/ioc/logger.go`
- Create: `payment/ioc/grpc.go`
- Create: `payment/ioc/idempotent.go`
- Create: `payment/ioc/snowflake.go`
- Create: `payment/wire.go`
- Create: `payment/app.go`
- Create: `payment/main.go`
- Create: `payment/config/dev.yaml`

### 9.1 payment/ioc/db.go

```go
package ioc

import (
	"fmt"

	"github.com/rermrf/mall/payment/repository/dao"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	type Config struct {
		DSN string `yaml:"dsn"`
	}
	var cfg Config
	err := viper.UnmarshalKey("db", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取数据库配置失败: %w", err))
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("连接数据库失败: %w", err))
	}
	err = dao.InitTables(db)
	if err != nil {
		panic(fmt.Errorf("数据库表初始化失败: %w", err))
	}
	return db
}
```

### 9.2 payment/ioc/redis.go

```go
package ioc

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
	type Config struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	}
	var cfg Config
	err := viper.UnmarshalKey("redis", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Redis 配置失败: %w", err))
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return client
}
```

### 9.3 payment/ioc/kafka.go

```go
package ioc

import (
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rermrf/mall/payment/events"
	"github.com/spf13/viper"
)

func InitKafka() sarama.Client {
	type Config struct {
		Addrs []string `yaml:"addrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("kafka", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Kafka 配置失败: %w", err))
	}
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	client, err := sarama.NewClient(cfg.Addrs, saramaCfg)
	if err != nil {
		panic(fmt.Errorf("连接 Kafka 失败: %w", err))
	}
	return client
}

func InitSyncProducer(client sarama.Client) sarama.SyncProducer {
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka SyncProducer 失败: %w", err))
	}
	return producer
}

func InitProducer(p sarama.SyncProducer) events.Producer {
	return events.NewSaramaProducer(p)
}
```

### 9.4 payment/ioc/logger.go

```go
package ioc

import (
	"github.com/rermrf/emo/logger"
	"go.uber.org/zap"
)

func InitLogger() logger.Logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger.NewZapLogger(l)
}
```

### 9.5 payment/ioc/grpc.go

```go
package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	pgrpc "github.com/rermrf/mall/payment/grpc"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

func InitEtcdClient() *clientv3.Client {
	var cfg struct {
		Addrs []string `yaml:"addrs"`
	}
	err := viper.UnmarshalKey("etcd", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 etcd 配置失败: %w", err))
	}
	client, err := clientv3.New(clientv3.Config{Endpoints: cfg.Addrs})
	if err != nil {
		panic(fmt.Errorf("连接 etcd 失败: %w", err))
	}
	return client
}

func InitGRPCServer(paymentServer *pgrpc.PaymentGRPCServer, l logger.Logger) *grpcx.Server {
	type Config struct {
		Port      int      `yaml:"port"`
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 gRPC 配置失败: %w", err))
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(tenantx.GRPCUnaryServerInterceptor()))
	paymentServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "payment",
		L:         l,
	}
}
```

### 9.6 payment/ioc/idempotent.go

```go
package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/rermrf/emo/idempotent"
)

func InitIdempotencyService(client redis.Cmdable) idempotent.IdempotencyService {
	return idempotent.NewBloomIdempotencyService(client, "payment:bloom", 1000000, 0.001)
}
```

### 9.7 payment/ioc/snowflake.go

```go
package ioc

import "github.com/rermrf/mall/pkg/snowflake"

func InitSnowflakeNode() *snowflake.Node {
	node, err := snowflake.NewNode(2)
	if err != nil {
		panic(err)
	}
	return node
}
```

### 9.8 payment/app.go

```go
package main

import (
	"github.com/rermrf/mall/pkg/grpcx"
)

type App struct {
	Server *grpcx.Server
}
```

注意：payment-svc 不消费 Kafka 事件，所以 App 没有 Consumers 字段。

### 9.9 payment/wire.go

```go
//go:build wireinject

package main

import (
	"github.com/google/wire"
	pgrpc "github.com/rermrf/mall/payment/grpc"
	"github.com/rermrf/mall/payment/ioc"
	"github.com/rermrf/mall/payment/repository"
	"github.com/rermrf/mall/payment/repository/cache"
	"github.com/rermrf/mall/payment/repository/dao"
	"github.com/rermrf/mall/payment/service"
	"github.com/rermrf/mall/payment/service/channel"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitIdempotencyService,
	ioc.InitSnowflakeNode,
)

var paymentSet = wire.NewSet(
	dao.NewPaymentDAO,
	cache.NewPaymentCache,
	repository.NewPaymentRepository,
	channel.NewMockChannel,
	service.NewPaymentService,
	pgrpc.NewPaymentGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, paymentSet, wire.Struct(new(App), "*"))
	return new(App)
}
```

### 9.10 payment/main.go

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()
	app := InitApp()

	go func() {
		if err := app.Server.Serve(); err != nil {
			fmt.Println("gRPC 服务启动失败:", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("正在关闭服务...")
	app.Server.Close()
}

func initViper() {
	configFile := pflag.String("config", "config/dev.yaml", "配置文件路径")
	pflag.Parse()
	viper.SetConfigFile(*configFile)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}
}
```

### 9.11 payment/config/dev.yaml

```yaml
db:
  dsn: "root:wen123...@tcp(rermrf.icu:3306)/mall_payment?charset=utf8mb4&parseTime=true&loc=Local"

redis:
  addr: "rermrf.icu:6379"
  password: ""
  db: 5

kafka:
  addrs:
    - "rermrf.icu:9094"

grpc:
  port: 8086
  etcdAddrs:
    - "rermrf.icu:2379"

etcd:
  addrs:
    - "rermrf.icu:2379"
```

---

## 验证步骤

```bash
wire ./payment/
go build ./payment/...
go vet ./payment/...
```

---

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `payment/domain/payment.go` | 新建 | PaymentOrder, RefundRecord, 状态常量 |
| 2 | `payment/repository/dao/payment.go` | 新建 | 2 GORM 模型 + PaymentDAO 接口 + 实现 |
| 3 | `payment/repository/dao/init.go` | 新建 | AutoMigrate 2 表 |
| 4 | `payment/repository/cache/payment.go` | 新建 | Redis 缓存（15min TTL） |
| 5 | `payment/repository/payment.go` | 新建 | Repository（Cache-Aside + 转换器） |
| 6 | `payment/service/channel/types.go` | 新建 | Channel 接口定义 |
| 7 | `payment/service/channel/mock.go` | 新建 | MockChannel 完整实现 |
| 8 | `payment/service/channel/wechat.go` | 新建 | WechatChannel 桩代码 |
| 9 | `payment/service/channel/alipay.go` | 新建 | AlipayChannel 桩代码 |
| 10 | `payment/service/payment.go` | 新建 | PaymentService 7 方法 |
| 11 | `payment/events/types.go` | 新建 | OrderPaidEvent + Topic 常量 |
| 12 | `payment/events/producer.go` | 新建 | Kafka SaramaProducer |
| 13 | `payment/grpc/payment.go` | 新建 | 7 RPC Handler + DTO 转换 |
| 14 | `payment/ioc/db.go` | 新建 | MySQL 初始化 |
| 15 | `payment/ioc/redis.go` | 新建 | Redis 初始化 |
| 16 | `payment/ioc/kafka.go` | 新建 | Kafka + SyncProducer + Producer |
| 17 | `payment/ioc/logger.go` | 新建 | Logger 初始化 |
| 18 | `payment/ioc/grpc.go` | 新建 | etcd + gRPC Server（端口 8086, 服务名 payment） |
| 19 | `payment/ioc/idempotent.go` | 新建 | BloomIdempotencyService |
| 20 | `payment/ioc/snowflake.go` | 新建 | Snowflake Node(2) |
| 21 | `payment/app.go` | 新建 | App{Server}（无 Consumers） |
| 22 | `payment/wire.go` | 新建 | thirdPartySet(7) + paymentSet(9) |
| 23 | `payment/main.go` | 新建 | 服务入口（无 Consumer 启动） |
| 24 | `payment/config/dev.yaml` | 新建 | 开发配置 |
| 25 | `payment/wire_gen.go` | 生成 | wire ./payment/ |
