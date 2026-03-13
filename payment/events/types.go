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
