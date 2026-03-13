package events

const (
	TopicSeckillSuccess = "seckill_success"
	TopicOrderCancelled = "order_cancelled"
)

type SeckillSuccessEvent struct {
	UserId       int64 `json:"user_id"`
	ItemId       int64 `json:"item_id"`
	SkuId        int64 `json:"sku_id"`
	SeckillPrice int64 `json:"seckill_price"`
	TenantId     int64 `json:"tenant_id"`
}

type OrderCancelledEvent struct {
	OrderNo  string `json:"order_no"`
	TenantID int64  `json:"tenant_id"`
	Reason   string `json:"reason"`
}
