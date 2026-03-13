package events

const (
	TopicProductUpdated       = "product_updated"
	TopicProductStatusChanged = "product_status_changed"
)

type ProductUpdatedEvent struct {
	ProductId int64 `json:"product_id"`
	TenantId  int64 `json:"tenant_id"`
}

type ProductStatusChangedEvent struct {
	ProductId int64 `json:"product_id"`
	TenantId  int64 `json:"tenant_id"`
	OldStatus int32 `json:"old_status"`
	NewStatus int32 `json:"new_status"`
}
