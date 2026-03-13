package domain

type Brand struct {
	ID       int64
	TenantID int64
	Name     string
	Logo     string
	Status   BrandStatus
}

type BrandStatus uint8

const (
	BrandStatusActive   BrandStatus = 1
	BrandStatusInactive BrandStatus = 2
)
