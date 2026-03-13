package domain

type Category struct {
	ID       int64
	TenantID int64
	ParentID int64
	Name     string
	Level    int32
	Sort     int32
	Icon     string
	Status   CategoryStatus
	Children []Category
}

type CategoryStatus uint8

const (
	CategoryStatusActive CategoryStatus = 1
	CategoryStatusHidden CategoryStatus = 2
)
