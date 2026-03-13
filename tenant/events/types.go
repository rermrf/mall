package events

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
