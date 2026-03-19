package events

type DelayMessage struct {
	Biz       string `json:"biz"`
	Key       string `json:"key"`
	Payload   string `json:"payload,omitempty"`
	BizTopic  string `json:"biz_topic"`
	ExecuteAt int64  `json:"execute_at"`
}

type OrderCloseDelayEvent struct {
	Biz      string `json:"biz"`
	Key      string `json:"key"`
	Payload  string `json:"payload,omitempty"`
	BizTopic string `json:"biz_topic"`
}

type OrderPaidEvent struct {
	OrderNo   string `json:"order_no"`
	PaymentNo string `json:"payment_no"`
	PaidAt    int64  `json:"paid_at"`
}

type OrderCancelledEvent struct {
	OrderNo  string `json:"order_no"`
	TenantID int64  `json:"tenant_id"`
	Reason   string `json:"reason"`
}

type OrderCompletedEvent struct {
	OrderNo   string              `json:"order_no"`
	TenantID  int64               `json:"tenant_id"`
	PaymentNo string              `json:"payment_no"`
	Amount    int64               `json:"amount"`
	Items     []CompletedItemInfo `json:"items"`
}

type CompletedItemInfo struct {
	ProductID int64 `json:"product_id"`
	Quantity  int32 `json:"quantity"`
}

const (
	TopicDelayMessage    = "delay_topic"
	TopicOrderCloseDelay = "order_close_delay"
	TopicOrderPaid       = "order_paid"
	TopicOrderCancelled  = "order_cancelled"
	TopicOrderCompleted  = "order_completed"
	TopicSeckillSuccess  = "seckill_success"
)

type SeckillSuccessEvent struct {
	UserId       int64 `json:"user_id"`
	ItemId       int64 `json:"item_id"`
	SkuId        int64 `json:"sku_id"`
	SeckillPrice int64 `json:"seckill_price"`
	TenantId     int64 `json:"tenant_id"`
}
