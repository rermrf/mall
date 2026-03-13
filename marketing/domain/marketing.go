package domain

import "time"

// ==================== 优惠券 ====================

type Coupon struct {
	ID            int64
	TenantID      int64
	Name          string
	Type          int32 // 1-满减 2-折扣 3-无门槛
	Threshold     int64 // 使用门槛（分），0=无门槛
	DiscountValue int64 // 满减=金额分，折扣=折扣比*100
	TotalCount    int32
	ReceivedCount int32
	UsedCount     int32
	PerLimit      int32 // 每人限领
	StartTime     time.Time
	EndTime       time.Time
	ScopeType     int32  // 1-全店 2-指定分类 3-指定商品
	ScopeIDs      string // JSON
	Status        int32  // 1-未开始 2-进行中 3-已结束 4-已停用
	Ctime         time.Time
}

type UserCoupon struct {
	ID          int64
	UserID      int64
	CouponID    int64
	TenantID    int64
	Status      int32 // 1-未使用 2-已使用 3-已过期
	OrderID     int64
	ReceiveTime time.Time
	UseTime     time.Time
	Coupon      Coupon // 嵌套
}

// ==================== 秒杀 ====================

type SeckillActivity struct {
	ID        int64
	TenantID  int64
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Status    int32 // 1-未开始 2-进行中 3-已结束
	Items     []SeckillItem
}

type SeckillItem struct {
	ID           int64
	ActivityID   int64
	TenantID     int64
	SkuID        int64
	SeckillPrice int64 // 分
	SeckillStock int32
	PerLimit     int32
}

type SeckillOrder struct {
	ID       int64
	UserID   int64
	ItemID   int64
	TenantID int64
	OrderNo  string
	Status   int32 // 1-排队中 2-已创建订单 3-失败
}

// ==================== 满减 ====================

type PromotionRule struct {
	ID            int64
	TenantID      int64
	Name          string
	Type          int32 // 1-满减 2-满折
	Threshold     int64
	DiscountValue int64
	StartTime     time.Time
	EndTime       time.Time
	Status        int32
}

// ==================== 优惠计算结果 ====================

type DiscountResult struct {
	CouponDiscount    int64 // 优惠券优惠金额（分）
	PromotionDiscount int64 // 满减优惠金额（分）
	TotalDiscount     int64 // 总优惠（分）
	PayAmount         int64 // 应付（分）
}
