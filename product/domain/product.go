package domain

import "time"

type Product struct {
	ID          int64
	TenantID    int64
	CategoryID  int64
	BrandID     int64
	Name        string
	Subtitle    string
	MainImage   string
	Images      string // JSON array
	Description string
	Status      ProductStatus
	Sales       int64
	SKUs        []SKU
	Specs       []ProductSpec
	Ctime       time.Time
	Utime       time.Time
}

type ProductStatus uint8

const (
	ProductStatusDraft       ProductStatus = 1
	ProductStatusPublished   ProductStatus = 2
	ProductStatusUnpublished ProductStatus = 3
)

type SKU struct {
	ID            int64
	TenantID      int64
	ProductID     int64
	SpecValues    string // JSON: {"颜色":"红","尺码":"XL"}
	Price         int64  // 分
	OriginalPrice int64
	CostPrice     int64
	SKUCode       string
	BarCode       string
	Status        SKUStatus
	Ctime         time.Time
	Utime         time.Time
}

type SKUStatus uint8

const (
	SKUStatusActive   SKUStatus = 1
	SKUStatusInactive SKUStatus = 2
)

type ProductSpec struct {
	ID        int64
	ProductID int64
	TenantID  int64
	Name      string // e.g. "颜色"
	Values    string // JSON array: ["红色","蓝色"]
}
