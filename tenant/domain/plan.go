package domain

type TenantPlan struct {
	ID           int64
	Name         string
	Price        int64
	DurationDays int32
	MaxProducts  int32
	MaxStaff     int32
	Features     string
	Status       PlanStatus
	Ctime        int64
	Utime        int64
}

type PlanStatus uint8

const (
	PlanStatusEnabled  PlanStatus = 1
	PlanStatusDisabled PlanStatus = 2
)
