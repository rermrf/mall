package domain

import "time"

// Order 订单聚合根
type Order struct {
	ID              int64
	TenantID        int64
	OrderNo         string
	BuyerID         int64
	BuyerHash       string // buyer_id + items hash，防重用
	Status          OrderStatus
	TotalAmount     int64 // 分
	DiscountAmount  int64
	FreightAmount   int64
	PayAmount       int64 // 实付
	RefundedAmount  int64 // 已退款
	CouponID        int64
	PaymentNo       string
	ReceiverName    string
	ReceiverPhone   string
	ReceiverAddress string
	Remark          string
	PaidAt          int64
	ShippedAt       int64
	ReceivedAt      int64
	ClosedAt        int64
	Items           []OrderItem
	Ctime           time.Time
	Utime           time.Time
}

type OrderStatus int32

const (
	OrderStatusPending   OrderStatus = 1
	OrderStatusPaid      OrderStatus = 2
	OrderStatusShipped   OrderStatus = 3
	OrderStatusReceived  OrderStatus = 4
	OrderStatusCompleted OrderStatus = 5
	OrderStatusCancelled OrderStatus = 6
	OrderStatusRefunding OrderStatus = 7
	OrderStatusRefunded  OrderStatus = 8
)

type OrderItem struct {
	ID               int64
	OrderID          int64
	TenantID         int64
	ProductID        int64
	SKUID            int64
	ProductName      string
	SKUSpec          string
	Image            string
	Price            int64 // 分
	Quantity         int32
	Subtotal         int64 // price * quantity
	RefundedQuantity int32
	Ctime            time.Time
}

type OrderStatusLog struct {
	ID           int64
	OrderID      int64
	FromStatus   OrderStatus
	ToStatus     OrderStatus
	OperatorID   int64
	OperatorType int32 // 1=买家 2=商家 3=平台 4=系统
	Remark       string
	Ctime        time.Time
}

type RefundOrder struct {
	ID           int64
	TenantID     int64
	OrderID      int64
	RefundNo     string
	BuyerID      int64
	Type         int32 // 1=仅退款 2=退货退款
	Status       RefundStatus
	RefundAmount int64
	Reason       string
	RejectReason string
	Items        string // JSON: [{"sku_id":1,"quantity":2}]
	Ctime        time.Time
	Utime        time.Time
}

type RefundStatus int32

const (
	RefundStatusPending   RefundStatus = 1 // 待审核
	RefundStatusApproved  RefundStatus = 2 // 审核通过
	RefundStatusRefunding RefundStatus = 3 // 退款中
	RefundStatusRefunded  RefundStatus = 4 // 已退款
	RefundStatusRejected  RefundStatus = 5 // 已拒绝
)
