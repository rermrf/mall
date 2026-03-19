package events

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
	TopicOrderCompleted = "order_completed"
)
