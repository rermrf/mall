package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type OrderModel struct {
	ID              int64  `gorm:"primaryKey;autoIncrement"`
	TenantId        int64  `gorm:"index:idx_tenant_status"`
	OrderNo         string `gorm:"type:varchar(64);uniqueIndex:uk_order_no"`
	BuyerId         int64  `gorm:"uniqueIndex:uk_buyer_hash"`
	BuyerHash       string `gorm:"type:varchar(64);uniqueIndex:uk_buyer_hash"`
	Status          int32  `gorm:"index:idx_tenant_status"`
	TotalAmount     int64
	DiscountAmount  int64
	FreightAmount   int64
	PayAmount       int64
	RefundedAmount  int64
	CouponId        int64
	PaymentNo       string `gorm:"type:varchar(64)"`
	ReceiverName    string `gorm:"type:varchar(64)"`
	ReceiverPhone   string `gorm:"type:varchar(32)"`
	ReceiverAddress string `gorm:"type:varchar(512)"`
	Remark          string `gorm:"type:varchar(256)"`
	PaidAt          int64
	ShippedAt       int64
	ReceivedAt      int64
	ClosedAt        int64
	Ctime           int64
	Utime           int64
}

func (OrderModel) TableName() string { return "orders" }

type OrderItemModel struct {
	ID               int64  `gorm:"primaryKey;autoIncrement"`
	OrderId          int64  `gorm:"index:idx_order_id"`
	TenantId         int64
	ProductId        int64
	SkuId            int64
	ProductName      string `gorm:"type:varchar(256)"`
	SkuSpec          string `gorm:"type:varchar(512)"`
	Image            string `gorm:"type:varchar(512)"`
	Price            int64
	Quantity         int32
	Subtotal         int64
	RefundedQuantity int32
	Ctime            int64
}

func (OrderItemModel) TableName() string { return "order_items" }

type OrderStatusLogModel struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	OrderId      int64  `gorm:"index:idx_order_id"`
	FromStatus   int32
	ToStatus     int32
	OperatorId   int64
	OperatorType int32
	Remark       string `gorm:"type:varchar(256)"`
	Ctime        int64
}

func (OrderStatusLogModel) TableName() string { return "order_status_logs" }

type RefundOrderModel struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	TenantId     int64  `gorm:"index:idx_tenant_status"`
	OrderId      int64  `gorm:"index:idx_order_id"`
	RefundNo     string `gorm:"type:varchar(64);uniqueIndex:uk_refund_no"`
	BuyerId      int64
	Type         int32
	Status       int32 `gorm:"index:idx_tenant_status"`
	RefundAmount int64
	Reason       string `gorm:"type:varchar(512)"`
	RejectReason string `gorm:"type:varchar(512)"`
	Items        string `gorm:"type:text"`
	Ctime        int64
	Utime        int64
}

func (RefundOrderModel) TableName() string { return "refund_orders" }

type OrderDAO interface {
	CreateOrder(ctx context.Context, tx *gorm.DB, order OrderModel, items []OrderItemModel) error
	FindByOrderNo(ctx context.Context, orderNo string) (OrderModel, error)
	FindItemsByOrderId(ctx context.Context, orderId int64) ([]OrderItemModel, error)
	FindByBuyerIdAndHash(ctx context.Context, buyerId int64, hash string) (OrderModel, error)
	ListOrders(ctx context.Context, buyerId, tenantId int64, status int32, offset, limit int) ([]OrderModel, int64, error)
	UpdateStatus(ctx context.Context, orderNo string, oldStatus, newStatus int32, updates map[string]any) error
	UpdatePaymentNo(ctx context.Context, orderNo string, paymentNo string) error
	AddRefundedAmount(ctx context.Context, orderNo string, amount int64) error
	InsertStatusLog(ctx context.Context, tx *gorm.DB, log OrderStatusLogModel) error
	CreateRefund(ctx context.Context, refund RefundOrderModel) error
	FindRefundByNo(ctx context.Context, refundNo string) (RefundOrderModel, error)
	UpdateRefundStatus(ctx context.Context, refundNo string, status int32, updates map[string]any) error
	ListRefunds(ctx context.Context, tenantId, buyerId int64, status int32, offset, limit int) ([]RefundOrderModel, int64, error)
	GetDB() *gorm.DB
}

type GORMOrderDAO struct {
	db *gorm.DB
}

func NewOrderDAO(db *gorm.DB) OrderDAO {
	return &GORMOrderDAO{db: db}
}

func (d *GORMOrderDAO) GetDB() *gorm.DB { return d.db }

func (d *GORMOrderDAO) CreateOrder(ctx context.Context, tx *gorm.DB, order OrderModel, items []OrderItemModel) error {
	now := time.Now().UnixMilli()
	order.Ctime = now
	order.Utime = now
	if err := tx.WithContext(ctx).Create(&order).Error; err != nil {
		return err
	}
	for i := range items {
		items[i].OrderId = order.ID
		items[i].Ctime = now
	}
	return tx.WithContext(ctx).Create(&items).Error
}

func (d *GORMOrderDAO) FindByOrderNo(ctx context.Context, orderNo string) (OrderModel, error) {
	var order OrderModel
	err := d.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&order).Error
	return order, err
}

func (d *GORMOrderDAO) FindItemsByOrderId(ctx context.Context, orderId int64) ([]OrderItemModel, error) {
	var items []OrderItemModel
	err := d.db.WithContext(ctx).Where("order_id = ?", orderId).Find(&items).Error
	return items, err
}

func (d *GORMOrderDAO) FindByBuyerIdAndHash(ctx context.Context, buyerId int64, hash string) (OrderModel, error) {
	var order OrderModel
	err := d.db.WithContext(ctx).Where("buyer_id = ? AND buyer_hash = ?", buyerId, hash).First(&order).Error
	return order, err
}

func (d *GORMOrderDAO) ListOrders(ctx context.Context, buyerId, tenantId int64, status int32, offset, limit int) ([]OrderModel, int64, error) {
	var orders []OrderModel
	var total int64
	query := d.db.WithContext(ctx).Model(&OrderModel{})
	if buyerId > 0 {
		query = query.Where("buyer_id = ?", buyerId)
	}
	if tenantId > 0 {
		query = query.Where("tenant_id = ?", tenantId)
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&orders).Error
	return orders, total, err
}

func (d *GORMOrderDAO) UpdateStatus(ctx context.Context, orderNo string, oldStatus, newStatus int32, updates map[string]any) error {
	updates["status"] = newStatus
	updates["utime"] = time.Now().UnixMilli()
	result := d.db.WithContext(ctx).Model(&OrderModel{}).
		Where("order_no = ? AND status = ?", orderNo, oldStatus).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *GORMOrderDAO) UpdatePaymentNo(ctx context.Context, orderNo string, paymentNo string) error {
	return d.db.WithContext(ctx).Model(&OrderModel{}).
		Where("order_no = ?", orderNo).
		Updates(map[string]any{"payment_no": paymentNo, "utime": time.Now().UnixMilli()}).Error
}

func (d *GORMOrderDAO) AddRefundedAmount(ctx context.Context, orderNo string, amount int64) error {
	return d.db.WithContext(ctx).Model(&OrderModel{}).
		Where("order_no = ?", orderNo).
		Updates(map[string]any{
			"refunded_amount": gorm.Expr("refunded_amount + ?", amount),
			"utime":           time.Now().UnixMilli(),
		}).Error
}

func (d *GORMOrderDAO) InsertStatusLog(ctx context.Context, tx *gorm.DB, log OrderStatusLogModel) error {
	log.Ctime = time.Now().UnixMilli()
	return tx.WithContext(ctx).Create(&log).Error
}

func (d *GORMOrderDAO) CreateRefund(ctx context.Context, refund RefundOrderModel) error {
	now := time.Now().UnixMilli()
	refund.Ctime = now
	refund.Utime = now
	return d.db.WithContext(ctx).Create(&refund).Error
}

func (d *GORMOrderDAO) FindRefundByNo(ctx context.Context, refundNo string) (RefundOrderModel, error) {
	var refund RefundOrderModel
	err := d.db.WithContext(ctx).Where("refund_no = ?", refundNo).First(&refund).Error
	return refund, err
}

func (d *GORMOrderDAO) UpdateRefundStatus(ctx context.Context, refundNo string, status int32, updates map[string]any) error {
	updates["status"] = status
	updates["utime"] = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Model(&RefundOrderModel{}).
		Where("refund_no = ?", refundNo).Updates(updates).Error
}

func (d *GORMOrderDAO) ListRefunds(ctx context.Context, tenantId, buyerId int64, status int32, offset, limit int) ([]RefundOrderModel, int64, error) {
	var refunds []RefundOrderModel
	var total int64
	query := d.db.WithContext(ctx).Model(&RefundOrderModel{})
	if tenantId > 0 {
		query = query.Where("tenant_id = ?", tenantId)
	}
	if buyerId > 0 {
		query = query.Where("buyer_id = ?", buyerId)
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&refunds).Error
	return refunds, total, err
}
