package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ==================== Models ====================

type Coupon struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	TenantID      int64  `gorm:"not null;index:idx_tenant_status"`
	Name          string `gorm:"type:varchar(200);not null"`
	Type          int32  `gorm:"not null"` // 1-满减 2-折扣 3-无门槛
	Threshold     int64  `gorm:"not null;default:0"`
	DiscountValue int64  `gorm:"not null"`
	TotalCount    int32  `gorm:"not null"`
	ReceivedCount int32  `gorm:"not null;default:0"`
	UsedCount     int32  `gorm:"not null;default:0"`
	PerLimit      int32  `gorm:"not null;default:1"`
	StartTime     int64  `gorm:"not null"`
	EndTime       int64  `gorm:"not null"`
	ScopeType     int32  `gorm:"not null;default:1"`
	ScopeIDs      string `gorm:"type:varchar(1000)"`
	Status        int32  `gorm:"not null;default:1;index:idx_tenant_status,priority:2"`
	Ctime         int64  `gorm:"not null"`
	Utime         int64  `gorm:"not null"`
}

type UserCoupon struct {
	ID          int64 `gorm:"primaryKey;autoIncrement"`
	UserID      int64 `gorm:"not null;uniqueIndex:uk_user_coupon;index:idx_user_tenant_status"`
	CouponID    int64 `gorm:"not null;uniqueIndex:uk_user_coupon,priority:2"`
	TenantID    int64 `gorm:"not null;index:idx_user_tenant_status,priority:2"`
	Status      int32 `gorm:"not null;default:1;index:idx_user_tenant_status,priority:3"` // 1-未使用 2-已使用 3-已过期
	OrderID     int64 `gorm:"default:0"`
	ReceiveTime int64 `gorm:"not null"`
	UseTime     int64 `gorm:"default:0"`
	Ctime       int64 `gorm:"not null"`
	Utime       int64 `gorm:"not null"`
}

type SeckillActivity struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	TenantID  int64  `gorm:"not null;index:idx_tenant_status"`
	Name      string `gorm:"type:varchar(200);not null"`
	StartTime int64  `gorm:"not null"`
	EndTime   int64  `gorm:"not null"`
	Status    int32  `gorm:"not null;default:1;index:idx_tenant_status,priority:2"`
	Ctime     int64  `gorm:"not null"`
	Utime     int64  `gorm:"not null"`
}

type SeckillItem struct {
	ID           int64 `gorm:"primaryKey;autoIncrement"`
	ActivityID   int64 `gorm:"not null;index:idx_activity"`
	TenantID     int64 `gorm:"not null"`
	SkuID        int64 `gorm:"not null;uniqueIndex:uk_activity_sku"`
	SeckillPrice int64 `gorm:"not null"`
	SeckillStock int32 `gorm:"not null"`
	PerLimit     int32 `gorm:"not null;default:1"`
}

type SeckillOrder struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	UserID   int64  `gorm:"not null;uniqueIndex:uk_user_item"`
	ItemID   int64  `gorm:"not null;uniqueIndex:uk_user_item,priority:2"`
	TenantID int64  `gorm:"not null"`
	OrderNo  string `gorm:"type:varchar(64)"`
	Status   int32  `gorm:"not null;default:1"` // 1-排队中 2-已创建订单 3-失败
	Ctime    int64  `gorm:"not null"`
	Utime    int64  `gorm:"not null"`
}

type PromotionRule struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	TenantID      int64  `gorm:"not null;index:idx_tenant_status"`
	Name          string `gorm:"type:varchar(200);not null"`
	Type          int32  `gorm:"not null"` // 1-满减 2-满折
	Threshold     int64  `gorm:"not null"`
	DiscountValue int64  `gorm:"not null"`
	StartTime     int64  `gorm:"not null"`
	EndTime       int64  `gorm:"not null"`
	Status        int32  `gorm:"not null;default:1;index:idx_tenant_status,priority:2"`
	Ctime         int64  `gorm:"not null"`
	Utime         int64  `gorm:"not null"`
}

// ==================== CouponDAO ====================

type CouponDAO interface {
	Insert(ctx context.Context, c Coupon) (Coupon, error)
	Update(ctx context.Context, c Coupon) error
	List(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]Coupon, int64, error)
	FindById(ctx context.Context, id int64) (Coupon, error)
	IncrReceivedCount(ctx context.Context, id int64) error
	IncrUsedCount(ctx context.Context, id int64) error
	DecrUsedCount(ctx context.Context, id int64) error
}

type GORMCouponDAO struct {
	db *gorm.DB
}

func NewCouponDAO(db *gorm.DB) CouponDAO {
	return &GORMCouponDAO{db: db}
}

func (d *GORMCouponDAO) Insert(ctx context.Context, c Coupon) (Coupon, error) {
	now := time.Now().UnixMilli()
	c.Ctime = now
	c.Utime = now
	err := d.db.WithContext(ctx).Create(&c).Error
	return c, err
}

func (d *GORMCouponDAO) Update(ctx context.Context, c Coupon) error {
	c.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", c.ID, c.TenantID).Updates(&c).Error
}

func (d *GORMCouponDAO) List(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]Coupon, int64, error) {
	db := d.db.WithContext(ctx).Model(&Coupon{}).Where("tenant_id = ?", tenantId)
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	var coupons []Coupon
	err = db.Order("id DESC").Offset(offset).Limit(limit).Find(&coupons).Error
	return coupons, total, err
}

func (d *GORMCouponDAO) FindById(ctx context.Context, id int64) (Coupon, error) {
	var c Coupon
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	return c, err
}

func (d *GORMCouponDAO) IncrReceivedCount(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Model(&Coupon{}).Where("id = ?", id).
		UpdateColumn("received_count", gorm.Expr("received_count + 1")).Error
}

func (d *GORMCouponDAO) IncrUsedCount(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Model(&Coupon{}).Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

func (d *GORMCouponDAO) DecrUsedCount(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Model(&Coupon{}).Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count - 1")).Error
}

// ==================== UserCouponDAO ====================

type UserCouponDAO interface {
	Insert(ctx context.Context, uc UserCoupon) (UserCoupon, error)
	FindById(ctx context.Context, id int64) (UserCoupon, error)
	ListByUser(ctx context.Context, userId, tenantId int64, status int32) ([]UserCoupon, error)
	UpdateStatus(ctx context.Context, id int64, status int32, orderId int64) error
	CountByUserAndCoupon(ctx context.Context, userId, couponId int64) (int64, error)
}

type GORMUserCouponDAO struct {
	db *gorm.DB
}

func NewUserCouponDAO(db *gorm.DB) UserCouponDAO {
	return &GORMUserCouponDAO{db: db}
}

func (d *GORMUserCouponDAO) Insert(ctx context.Context, uc UserCoupon) (UserCoupon, error) {
	now := time.Now().UnixMilli()
	uc.Ctime = now
	uc.Utime = now
	uc.ReceiveTime = now
	err := d.db.WithContext(ctx).Create(&uc).Error
	return uc, err
}

func (d *GORMUserCouponDAO) FindById(ctx context.Context, id int64) (UserCoupon, error) {
	var uc UserCoupon
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&uc).Error
	return uc, err
}

func (d *GORMUserCouponDAO) ListByUser(ctx context.Context, userId, tenantId int64, status int32) ([]UserCoupon, error) {
	db := d.db.WithContext(ctx).Where("user_id = ?", userId)
	if tenantId > 0 {
		db = db.Where("tenant_id = ?", tenantId)
	}
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var ucs []UserCoupon
	err := db.Order("id DESC").Find(&ucs).Error
	return ucs, err
}

func (d *GORMUserCouponDAO) UpdateStatus(ctx context.Context, id int64, status int32, orderId int64) error {
	updates := map[string]any{
		"status": status,
		"utime":  time.Now().UnixMilli(),
	}
	if orderId > 0 {
		updates["order_id"] = orderId
		updates["use_time"] = time.Now().UnixMilli()
	}
	if status == 1 {
		// 释放时清空 order_id 和 use_time
		updates["order_id"] = 0
		updates["use_time"] = 0
	}
	return d.db.WithContext(ctx).Model(&UserCoupon{}).Where("id = ?", id).Updates(updates).Error
}

func (d *GORMUserCouponDAO) CountByUserAndCoupon(ctx context.Context, userId, couponId int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&UserCoupon{}).
		Where("user_id = ? AND coupon_id = ?", userId, couponId).Count(&count).Error
	return count, err
}

// ==================== SeckillDAO ====================

type SeckillDAO interface {
	InsertActivity(ctx context.Context, a SeckillActivity, items []SeckillItem) (SeckillActivity, error)
	UpdateActivity(ctx context.Context, a SeckillActivity, items []SeckillItem) error
	ListActivities(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]SeckillActivity, int64, error)
	FindActivityById(ctx context.Context, id int64) (SeckillActivity, []SeckillItem, error)
	FindItemById(ctx context.Context, id int64) (SeckillItem, error)
	InsertSeckillOrder(ctx context.Context, o SeckillOrder) (SeckillOrder, error)
	FindSeckillOrderByUserAndItem(ctx context.Context, userId, itemId int64) (SeckillOrder, error)
}

type GORMSeckillDAO struct {
	db *gorm.DB
}

func NewSeckillDAO(db *gorm.DB) SeckillDAO {
	return &GORMSeckillDAO{db: db}
}

func (d *GORMSeckillDAO) InsertActivity(ctx context.Context, a SeckillActivity, items []SeckillItem) (SeckillActivity, error) {
	now := time.Now().UnixMilli()
	a.Ctime = now
	a.Utime = now
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&a).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].ActivityID = a.ID
			items[i].TenantID = a.TenantID
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
	return a, err
}

func (d *GORMSeckillDAO) UpdateActivity(ctx context.Context, a SeckillActivity, items []SeckillItem) error {
	a.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND tenant_id = ?", a.ID, a.TenantID).Updates(&a).Error; err != nil {
			return err
		}
		// 删除旧 items 并重建
		if err := tx.Where("activity_id = ?", a.ID).Delete(&SeckillItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].ActivityID = a.ID
			items[i].TenantID = a.TenantID
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
}

func (d *GORMSeckillDAO) ListActivities(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]SeckillActivity, int64, error) {
	db := d.db.WithContext(ctx).Model(&SeckillActivity{}).Where("tenant_id = ?", tenantId)
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	var activities []SeckillActivity
	err = db.Order("id DESC").Offset(offset).Limit(limit).Find(&activities).Error
	return activities, total, err
}

func (d *GORMSeckillDAO) FindActivityById(ctx context.Context, id int64) (SeckillActivity, []SeckillItem, error) {
	var a SeckillActivity
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&a).Error
	if err != nil {
		return a, nil, err
	}
	var items []SeckillItem
	err = d.db.WithContext(ctx).Where("activity_id = ?", id).Find(&items).Error
	return a, items, err
}

func (d *GORMSeckillDAO) FindItemById(ctx context.Context, id int64) (SeckillItem, error) {
	var item SeckillItem
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return item, err
}

func (d *GORMSeckillDAO) InsertSeckillOrder(ctx context.Context, o SeckillOrder) (SeckillOrder, error) {
	now := time.Now().UnixMilli()
	o.Ctime = now
	o.Utime = now
	err := d.db.WithContext(ctx).Create(&o).Error
	return o, err
}

func (d *GORMSeckillDAO) FindSeckillOrderByUserAndItem(ctx context.Context, userId, itemId int64) (SeckillOrder, error) {
	var o SeckillOrder
	err := d.db.WithContext(ctx).Where("user_id = ? AND item_id = ?", userId, itemId).First(&o).Error
	return o, err
}

// ==================== PromotionDAO ====================

type PromotionDAO interface {
	Insert(ctx context.Context, r PromotionRule) (PromotionRule, error)
	Update(ctx context.Context, r PromotionRule) error
	List(ctx context.Context, tenantId int64, status int32) ([]PromotionRule, error)
	ListActive(ctx context.Context, tenantId int64, now int64) ([]PromotionRule, error)
}

type GORMPromotionDAO struct {
	db *gorm.DB
}

func NewPromotionDAO(db *gorm.DB) PromotionDAO {
	return &GORMPromotionDAO{db: db}
}

func (d *GORMPromotionDAO) Insert(ctx context.Context, r PromotionRule) (PromotionRule, error) {
	now := time.Now().UnixMilli()
	r.Ctime = now
	r.Utime = now
	err := d.db.WithContext(ctx).Create(&r).Error
	return r, err
}

func (d *GORMPromotionDAO) Update(ctx context.Context, r PromotionRule) error {
	r.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", r.ID, r.TenantID).Updates(&r).Error
}

func (d *GORMPromotionDAO) List(ctx context.Context, tenantId int64, status int32) ([]PromotionRule, error) {
	db := d.db.WithContext(ctx).Where("tenant_id = ?", tenantId)
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	var rules []PromotionRule
	err := db.Order("id DESC").Find(&rules).Error
	return rules, err
}

func (d *GORMPromotionDAO) ListActive(ctx context.Context, tenantId int64, now int64) ([]PromotionRule, error) {
	var rules []PromotionRule
	err := d.db.WithContext(ctx).
		Where("tenant_id = ? AND status = 2 AND start_time <= ? AND end_time >= ?", tenantId, now, now).
		Find(&rules).Error
	return rules, err
}
