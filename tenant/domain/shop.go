package domain

type Shop struct {
	ID           int64
	TenantID     int64
	Name         string
	Logo         string
	Description  string
	Status       ShopStatus
	Rating       string
	Subdomain    string
	CustomDomain string
	Ctime        int64
	Utime        int64
}

type ShopStatus uint8

const (
	ShopStatusOpen   ShopStatus = 1
	ShopStatusRest   ShopStatus = 2
	ShopStatusClosed ShopStatus = 3
)

type QuotaUsage struct {
	QuotaType string
	Used      int32
	MaxLimit  int32
}
