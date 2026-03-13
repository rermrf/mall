package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&Coupon{},
		&UserCoupon{},
		&SeckillActivity{},
		&SeckillItem{},
		&SeckillOrder{},
		&PromotionRule{},
	)
}
