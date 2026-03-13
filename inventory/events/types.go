package events

// DelayMessage go-delay 延迟消息格式
type DelayMessage struct {
	Biz       string `json:"biz"`
	Key       string `json:"key"`
	Payload   string `json:"payload,omitempty"`
	BizTopic  string `json:"biz_topic"`
	ExecuteAt int64  `json:"execute_at"` // Unix 秒
}

// DeductExpireEvent inventory_deduct_expire 消费事件
type DeductExpireEvent struct {
	Biz      string `json:"biz"`
	Key      string `json:"key"`
	Payload  string `json:"payload,omitempty"`
	BizTopic string `json:"biz_topic"`
}

// InventoryAlertEvent 库存预警事件
type InventoryAlertEvent struct {
	TenantID  int64 `json:"tenant_id"`
	SKUID     int64 `json:"sku_id"`
	Available int32 `json:"available"`
	Threshold int32 `json:"threshold"`
}

const (
	TopicDelayMessage   = "delay_topic"
	TopicDeductExpire   = "inventory_deduct_expire"
	TopicInventoryAlert = "inventory_alert"
)
