package domain

import "time"

type CartItem struct {
	ID        int64
	UserID    int64
	SkuID     int64
	ProductID int64
	TenantID  int64
	Quantity  int32
	Selected  bool
	Ctime     time.Time
	Utime     time.Time
}
