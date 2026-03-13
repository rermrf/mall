// Package domain user/domain/user.go
package domain

import "time"

// User 用户领域对象（租户级隔离：同一手机号在不同店铺是不同用户）
type User struct {
	ID       int64
	TenantID int64 // 所属租户，唯一索引：tenant_id + phone
	Phone    string
	Email    string
	Password string
	Nickname string
	Avatar   string
	Status   UserStatus
	Ctime    time.Time
	Utime    time.Time
}

type UserStatus uint8

const (
	UserStatusNormal  UserStatus = 1
	UserStatusFrozen  UserStatus = 2
	UserStatusDeleted UserStatus = 3
)

// UserAddress 收货地址
type UserAddress struct {
	Id        int64
	UserID    int64
	Name      string
	Phone     string
	Province  string
	City      string
	District  string
	Detail    string
	IsDefault bool
}

// OAuthAccount 第三方登录账号（租户级隔离）
type OAuthAccount struct {
	ID          int64
	UserID      int64
	TenantID    int64  // 所属租户
	Provider    string // wechat / google / github
	ProviderUID string // 唯一索引：tenant_id + provider + provider_uid
	AccessToken string
	Ctime       time.Time
	Utime       time.Time
}
