package domain

import "time"

type Tenant struct {
	ID              int64
	Name            string
	ContactName     string
	ContactPhone    string
	BusinessLicense string
	Status          TenantStatus
	PlanID          int64
	PlanExpireTime  int64
	Ctime           time.Time
	Utime           time.Time
}

type TenantStatus uint8

const (
	TenantStatusPending  TenantStatus = 1
	TenantStatusNormal   TenantStatus = 2
	TenantStatusFrozen   TenantStatus = 3
	TenantStatusCanceled TenantStatus = 4
)
