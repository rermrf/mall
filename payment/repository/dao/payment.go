package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// statusPaid mirrors domain.PaymentStatusPaid to avoid circular imports.
const statusPaid int32 = 3

const (
	statusPending   int32 = 1
	statusPaying    int32 = 2
	refundSucceeded int32 = 2
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
	TotalAmount     int64 // 原始支付总额（用于部分退款场景）
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
	ListOpenPaymentsByOrderNo(ctx context.Context, orderNo string) ([]PaymentOrderModel, error)
	UpdateStatus(ctx context.Context, paymentNo string, oldStatus, newStatus int32, updates map[string]any) error
	ListPayments(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]PaymentOrderModel, int64, error)
	ListPaymentsByDateAndChannel(ctx context.Context, channel string, startTime, endTime int64) ([]PaymentOrderModel, error)
	CreateRefund(ctx context.Context, refund RefundRecordModel) error
	FindRefundByNo(ctx context.Context, refundNo string) (RefundRecordModel, error)
	UpdateRefundStatus(ctx context.Context, refundNo string, oldStatus, newStatus int32, updates map[string]any) error
	SumRefundedAmountByPaymentNo(ctx context.Context, paymentNo string) (int64, error)
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

func (d *GORMPaymentDAO) ListOpenPaymentsByOrderNo(ctx context.Context, orderNo string) ([]PaymentOrderModel, error) {
	var payments []PaymentOrderModel
	err := d.db.WithContext(ctx).
		Where("order_no = ? AND status IN ?", orderNo, []int32{statusPending, statusPaying}).
		Order("id DESC").
		Find(&payments).Error
	return payments, err
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

func (d *GORMPaymentDAO) UpdateRefundStatus(ctx context.Context, refundNo string, oldStatus, newStatus int32, updates map[string]any) error {
	updates["status"] = newStatus
	updates["utime"] = time.Now().UnixMilli()
	result := d.db.WithContext(ctx).Model(&RefundRecordModel{}).
		Where("refund_no = ? AND status = ?", refundNo, oldStatus).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *GORMPaymentDAO) SumRefundedAmountByPaymentNo(ctx context.Context, paymentNo string) (int64, error) {
	var total int64
	err := d.db.WithContext(ctx).
		Model(&RefundRecordModel{}).
		Where("payment_no = ? AND status = ?", paymentNo, refundSucceeded).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}

func (d *GORMPaymentDAO) ListPaymentsByDateAndChannel(ctx context.Context, channel string, startTime, endTime int64) ([]PaymentOrderModel, error) {
	var payments []PaymentOrderModel
	err := d.db.WithContext(ctx).
		Where("channel = ? AND pay_time >= ? AND pay_time < ? AND status = ?", channel, startTime, endTime, statusPaid).
		Find(&payments).Error
	return payments, err
}
