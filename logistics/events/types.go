package events

const (
	TopicOrderShipped = "order_shipped"
)

type OrderShippedEvent struct {
	OrderId     int64  `json:"order_id"`
	OrderNo     string `json:"order_no"`
	TenantId    int64  `json:"tenant_id"`
	CarrierCode string `json:"carrier_code"`
	CarrierName string `json:"carrier_name"`
	TrackingNo  string `json:"tracking_no"`
}
