package domain

type ProductDocument struct {
	ID           int64
	TenantID     int64
	Name         string
	Subtitle     string
	CategoryID   int64
	CategoryName string
	BrandID      int64
	BrandName    string
	Price        int64
	Sales        int64
	MainImage    string
	Status       int32
	ShopID       int64
	ShopName     string
}

type HotWord struct {
	Word  string
	Count int64
}

type SearchHistory struct {
	Keyword string
	Ctime   int64
}
