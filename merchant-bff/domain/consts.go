package domain

// 订单状态
const (
	OrderStatusCancelled = 0 // 已取消
	OrderStatusPending   = 1 // 待付款
	OrderStatusPaid      = 2 // 待发货
	OrderStatusShipped   = 3 // 已发货
	OrderStatusCompleted = 4 // 已完成
	OrderStatusRefunding = 5 // 退款中
)

// 支付状态
const (
	PaymentStatusPending = 0 // 待支付
	PaymentStatusPaid    = 1 // 已支付
	PaymentStatusRefund  = 2 // 已退款
	PaymentStatusClosed  = 3 // 已关闭
)

// 操作者类型
const (
	OperatorTypeSystem   = 0 // 系统
	OperatorTypeCustomer = 1 // 消费者
	OperatorTypeMerchant = 2 // 商家
)

// 优惠券类型
const (
	CouponTypeFixed    = 1 // 满减
	CouponTypeDiscount = 2 // 折扣
	CouponTypeGift     = 3 // 固定金额
)

// 优惠券适用范围
const (
	CouponScopeAll      = 0 // 全场
	CouponScopeProduct  = 1 // 指定商品
	CouponScopeCategory = 2 // 指定分类
)
