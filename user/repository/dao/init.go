package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&UserRole{},
		&Role{},
		&RolePermission{},
		&Permission{},
		&UserAddress{},
		&OAuthAccount{},
	)
}
