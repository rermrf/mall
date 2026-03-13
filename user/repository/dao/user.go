package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ==================== GORM Models ====================

type User struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	TenantId int64  `gorm:"uniqueIndex:uk_tenant_phone;uniqueIndex:uk_tenant_email;not null"`
	Phone    string `gorm:"uniqueIndex:uk_tenant_phone;type:varchar(20);not null"`
	Email    string `gorm:"uniqueIndex:uk_tenant_email;type:varchar(128)"`
	Password string `gorm:"type:varchar(256)"`
	Nickname string `gorm:"type:varchar(64)"`
	Avatar   string `gorm:"type:varchar(512)"`
	Status   uint8  `gorm:"default:1;not null"` // 1-正常 2-冻结 3-注销
	Ctime    int64  `gorm:"not null"`
	Utime    int64  `gorm:"not null"`
}

type UserRole struct {
	ID       int64 `gorm:"primaryKey;autoIncrement"`
	UserId   int64 `gorm:"uniqueIndex:uk_user_tenant_role;not null"`
	TenantId int64 `gorm:"uniqueIndex:uk_user_tenant_role;not null"`
	RoleId   int64 `gorm:"uniqueIndex:uk_user_tenant_role;not null"`
	Ctime    int64 `gorm:"not null"`
	Utime    int64 `gorm:"not null"`
}

type Role struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	TenantId    int64  `gorm:"uniqueIndex:uk_tenant_code;not null"`
	Name        string `gorm:"type:varchar(64);not null"`
	Code        string `gorm:"uniqueIndex:uk_tenant_code;type:varchar(64);not null"`
	Description string `gorm:"type:varchar(256)"`
	Ctime       int64  `gorm:"not null"`
	Utime       int64  `gorm:"not null"`
}

type RolePermission struct {
	ID           int64 `gorm:"primaryKey;autoIncrement"`
	RoleId       int64 `gorm:"index:idx_role;not null"`
	PermissionId int64 `gorm:"not null"`
	Ctime        int64 `gorm:"not null"`
}

type Permission struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	Code     string `gorm:"uniqueIndex:uk_code;type:varchar(128);not null"`
	Name     string `gorm:"type:varchar(64);not null"`
	Type     uint8  `gorm:"not null"` // 1-菜单 2-按钮 3-API
	Resource string `gorm:"type:varchar(256)"`
	Ctime    int64  `gorm:"not null"`
	Utime    int64  `gorm:"not null"`
}

type UserAddress struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	UserId    int64  `gorm:"index:idx_user;not null"`
	Name      string `gorm:"type:varchar(64);not null"`
	Phone     string `gorm:"type:varchar(20);not null"`
	Province  string `gorm:"type:varchar(32)"`
	City      string `gorm:"type:varchar(32)"`
	District  string `gorm:"type:varchar(32)"`
	Detail    string `gorm:"type:varchar(256)"`
	IsDefault bool   `gorm:"default:false"`
	Ctime     int64  `gorm:"not null"`
	Utime     int64  `gorm:"not null"`
}

type OAuthAccount struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	UserId      int64  `gorm:"not null"`
	TenantId    int64  `gorm:"uniqueIndex:uk_tenant_provider_uid;not null"`
	Provider    string `gorm:"uniqueIndex:uk_tenant_provider_uid;type:varchar(32);not null"`
	ProviderUid string `gorm:"uniqueIndex:uk_tenant_provider_uid;type:varchar(128);not null"`
	AccessToken string `gorm:"type:varchar(512)"`
	Ctime       int64  `gorm:"not null"`
	Utime       int64  `gorm:"not null"`
}

// ==================== DAO Interfaces ====================

type UserDAO interface {
	Insert(ctx context.Context, u User) (User, error)
	FindByTenantAndPhone(ctx context.Context, tenantId int64, phone string) (User, error)
	FindByTenantAndEmail(ctx context.Context, tenantId int64, email string) (User, error)
	FindById(ctx context.Context, id int64) (User, error)
	UpdateNonZeroFields(ctx context.Context, u User) error
	UpdateStatus(ctx context.Context, id int64, status uint8) error
	ListByTenant(ctx context.Context, tenantId int64, offset, limit int, status uint8, keyword string) ([]User, int64, error)
	FindOrCreateByOAuth(ctx context.Context, oauth OAuthAccount, u User) (User, error)
}

type RoleDAO interface {
	InsertRole(ctx context.Context, r Role) (Role, error)
	UpdateRole(ctx context.Context, r Role) error
	ListByTenant(ctx context.Context, tenantId int64) ([]Role, error)
	InsertUserRole(ctx context.Context, ur UserRole) error
	GetPermissionsByUserAndTenant(ctx context.Context, userId, tenantId int64) ([]Permission, error)
	InsertRolePermission(ctx context.Context, rp RolePermission) error
}

type AddressDAO interface {
	Insert(ctx context.Context, a UserAddress) (UserAddress, error)
	ListByUser(ctx context.Context, userId int64) ([]UserAddress, error)
	Update(ctx context.Context, a UserAddress) error
	Delete(ctx context.Context, id, userId int64) error
}

// ==================== UserDAO Impl ====================

type GORMUserDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) UserDAO {
	return &GORMUserDAO{db: db}
}

func (dao *GORMUserDAO) Insert(ctx context.Context, u User) (User, error) {
	now := time.Now().UnixMilli()
	u.Ctime = now
	u.Utime = now
	err := dao.db.WithContext(ctx).Create(&u).Error
	return u, err
}

func (dao *GORMUserDAO) FindByTenantAndPhone(ctx context.Context, tenantId int64, phone string) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).
		Where("tenant_id = ? AND phone = ?", tenantId, phone).
		First(&u).Error
	return u, err
}

func (dao *GORMUserDAO) FindByTenantAndEmail(ctx context.Context, tenantId int64, email string) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).
		Where("tenant_id = ? AND email = ?", tenantId, email).
		First(&u).Error
	return u, err
}

func (dao *GORMUserDAO) FindById(ctx context.Context, id int64) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	return u, err
}

func (dao *GORMUserDAO) UpdateNonZeroFields(ctx context.Context, u User) error {
	u.Utime = time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Updates(&u).Error
}

func (dao *GORMUserDAO) UpdateStatus(ctx context.Context, id int64, status uint8) error {
	return dao.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status": status,
			"utime":  time.Now().UnixMilli(),
		}).Error
}

func (dao *GORMUserDAO) ListByTenant(ctx context.Context, tenantId int64, offset, limit int, status uint8, keyword string) ([]User, int64, error) {
	db := dao.db.WithContext(ctx).Model(&User{})
	if tenantId > 0 {
		db = db.Where("tenant_id = ?", tenantId)
	}
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("phone LIKE ? OR nickname LIKE ?", like, like)
	}
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	var users []User
	err = db.Offset(offset).Limit(limit).Order("id DESC").Find(&users).Error
	return users, total, err
}

func (dao *GORMUserDAO) FindOrCreateByOAuth(ctx context.Context, oauth OAuthAccount, u User) (User, error) {
	var oa OAuthAccount
	err := dao.db.WithContext(ctx).
		Where("tenant_id = ? AND provider = ? AND provider_uid = ?", oauth.TenantId, oauth.Provider, oauth.ProviderUid).
		First(&oa).Error
	if err == nil {
		// 已存在，返回关联用户
		var user User
		err = dao.db.WithContext(ctx).Where("id = ?", oa.UserId).First(&user).Error
		return user, err
	}
	if err != gorm.ErrRecordNotFound {
		return User{}, err
	}
	// 不存在，事务创建用户+OAuth 账号
	now := time.Now().UnixMilli()
	u.Ctime = now
	u.Utime = now
	err = dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先用 Upsert 防止并发重复
		if err := tx.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&u).Error; err != nil {
			return err
		}
		if u.ID == 0 {
			// 并发创建，取已有的
			return tx.Where("tenant_id = ? AND phone = ?", u.TenantId, u.Phone).First(&u).Error
		}
		oauth.UserId = u.ID
		oauth.Ctime = now
		oauth.Utime = now
		return tx.Create(&oauth).Error
	})
	return u, err
}

// ==================== RoleDAO Impl ====================

type GORMRoleDAO struct {
	db *gorm.DB
}

func NewRoleDAO(db *gorm.DB) RoleDAO {
	return &GORMRoleDAO{db: db}
}

func (dao *GORMRoleDAO) InsertRole(ctx context.Context, r Role) (Role, error) {
	now := time.Now().UnixMilli()
	r.Ctime = now
	r.Utime = now
	err := dao.db.WithContext(ctx).Create(&r).Error
	return r, err
}

func (dao *GORMRoleDAO) UpdateRole(ctx context.Context, r Role) error {
	r.Utime = time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Updates(&r).Error
}

func (dao *GORMRoleDAO) ListByTenant(ctx context.Context, tenantId int64) ([]Role, error) {
	var roles []Role
	err := dao.db.WithContext(ctx).
		Where("tenant_id = ? OR tenant_id = 0", tenantId).
		Order("id ASC").
		Find(&roles).Error
	return roles, err
}

func (dao *GORMRoleDAO) InsertUserRole(ctx context.Context, ur UserRole) error {
	now := time.Now().UnixMilli()
	ur.Ctime = now
	ur.Utime = now
	return dao.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&ur).Error
}

func (dao *GORMRoleDAO) GetPermissionsByUserAndTenant(ctx context.Context, userId, tenantId int64) ([]Permission, error) {
	var perms []Permission
	// user_roles → role_permissions → permissions
	err := dao.db.WithContext(ctx).
		Distinct("permissions.*").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ? AND user_roles.tenant_id = ?", userId, tenantId).
		Find(&perms).Error
	return perms, err
}

func (dao *GORMRoleDAO) InsertRolePermission(ctx context.Context, rp RolePermission) error {
	rp.Ctime = time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Create(&rp).Error
}

// ==================== AddressDAO Impl ====================

type GORMAddressDAO struct {
	db *gorm.DB
}

func NewAddressDAO(db *gorm.DB) AddressDAO {
	return &GORMAddressDAO{db: db}
}

func (dao *GORMAddressDAO) Insert(ctx context.Context, a UserAddress) (UserAddress, error) {
	now := time.Now().UnixMilli()
	a.Ctime = now
	a.Utime = now
	err := dao.db.WithContext(ctx).Create(&a).Error
	return a, err
}

func (dao *GORMAddressDAO) ListByUser(ctx context.Context, userId int64) ([]UserAddress, error) {
	var addrs []UserAddress
	err := dao.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Order("is_default DESC, id DESC").
		Find(&addrs).Error
	return addrs, err
}

func (dao *GORMAddressDAO) Update(ctx context.Context, a UserAddress) error {
	a.Utime = time.Now().UnixMilli()
	return dao.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", a.ID, a.UserId).
		Updates(&a).Error
}

func (dao *GORMAddressDAO) Delete(ctx context.Context, id, userId int64) error {
	return dao.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userId).
		Delete(&UserAddress{}).Error
}
