package domain

import "time"

// ==================== FreightTemplate ====================

type FreightTemplate struct {
	ID            int64
	TenantID      int64
	Name          string
	ChargeType    int32 // 1-按重量 2-按件数
	FreeThreshold int64 // 包邮门槛（分），0=不包邮
	Rules         []FreightRule
	Ctime         time.Time
	Utime         time.Time
}

type FreightRule struct {
	ID              int64
	TemplateID      int64
	Regions         string // JSON
	FirstUnit       int32
	FirstPrice      int64
	AdditionalUnit  int32
	AdditionalPrice int64
}

// ==================== Shipment ====================

type Shipment struct {
	ID          int64
	TenantID    int64
	OrderID     int64
	CarrierCode string
	CarrierName string
	TrackingNo  string
	Status      int32 // 1-待发货 2-已发货 3-运输中 4-已签收
	Tracks      []ShipmentTrack
	Ctime       time.Time
	Utime       time.Time
}

type ShipmentTrack struct {
	ID          int64
	ShipmentID  int64
	Description string
	Location    string
	TrackTime   int64
}
