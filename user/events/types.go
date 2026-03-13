package events

type UserRegisteredEvent struct {
	UserId   int64  `json:"user_id"`
	TenantId int64  `json:"tenant_id"`
	Phone    string `json:"phone"`
}
