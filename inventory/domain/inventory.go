package domain

import "time"

type Inventory struct {
	ID             int64
	TenantID       int64
	SKUID          int64
	Total          int32
	Available      int32
	Locked         int32
	Sold           int32
	AlertThreshold int32
	Ctime          time.Time
	Utime          time.Time
}

type DeductRecord struct {
	OrderID  int64
	TenantID int64
	Items    []DeductItem
}

type DeductItem struct {
	SKUID    int64
	Quantity int32
}

type InventoryLog struct {
	ID              int64
	SKUID           int64
	OrderID         int64
	Type            LogType
	Quantity        int32
	BeforeAvailable int32
	AfterAvailable  int32
	TenantID        int64
	Ctime           time.Time
}

type LogType uint8

const (
	LogTypeDeduct   LogType = 1
	LogTypeConfirm  LogType = 2
	LogTypeRollback LogType = 3
	LogTypeManual   LogType = 4
)
