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
	RefundStatusRefunding RefundStatus = 1 // 退款中
	RefundStatusRefunded  RefundStatus = 2 // 已退款
	RefundStatusFailed    RefundStatus = 3 // 退款失败
)
