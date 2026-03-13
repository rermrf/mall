package events

const (
	TopicUserRegistered = "user_registered"
	TopicOrderPaid      = "order_paid"
	TopicOrderShipped   = "order_shipped"
	TopicInventoryAlert = "inventory_alert"
	TopicTenantApproved    = "tenant_approved"
	TopicTenantPlanChanged = "tenant_plan_changed"
	TopicOrderCompleted    = "order_completed"
)

type UserRegisteredEvent struct {
	UserId   int64  `json:"user_id"`
	TenantId int64  `json:"tenant_id"`
	Phone    string `json:"phone"`
}

type OrderPaidEvent struct {
	OrderNo   string `json:"order_no"`
	PaymentNo string `json:"payment_no"`
	PaidAt    int64  `json:"paid_at"`
}

type OrderShippedEvent struct {
	OrderId     int64  `json:"order_id"`
	OrderNo     string `json:"order_no"`
	TenantId    int64  `json:"tenant_id"`
	CarrierCode string `json:"carrier_code"`
	CarrierName string `json:"carrier_name"`
	TrackingNo  string `json:"tracking_no"`
}

type InventoryAlertEvent struct {
	TenantID  int64 `json:"tenant_id"`
	SKUID     int64 `json:"sku_id"`
	Available int32 `json:"available"`
	Threshold int32 `json:"threshold"`
}

type TenantApprovedEvent struct {
	TenantId int64  `json:"tenant_id"`
	Name     string `json:"name"`
	PlanId   int64  `json:"plan_id"`
}

type TenantPlanChangedEvent struct {
	TenantId  int64 `json:"tenant_id"`
	OldPlanId int64 `json:"old_plan_id"`
	NewPlanId int64 `json:"new_plan_id"`
}

type OrderCompletedEvent struct {
	OrderNo  string              `json:"order_no"`
	TenantID int64               `json:"tenant_id"`
	Items    []CompletedItemInfo `json:"items"`
}

type CompletedItemInfo struct {
	ProductID int64 `json:"product_id"`
	Quantity  int32 `json:"quantity"`
}
