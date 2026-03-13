package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/rermrf/mall/marketing/domain"
	"github.com/rermrf/mall/marketing/repository/cache"
	"github.com/rermrf/mall/marketing/repository/dao"
)

var ErrCouponStockNotEnough = fmt.Errorf("优惠券库存不足")

type MarketingRepository interface {
	// 优惠券
	CreateCoupon(ctx context.Context, c domain.Coupon) (domain.Coupon, error)
	UpdateCoupon(ctx context.Context, c domain.Coupon) error
	ListCoupons(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.Coupon, int64, error)
	FindCouponById(ctx context.Context, id int64) (domain.Coupon, error)
	// 领券
	ReceiveCoupon(ctx context.Context, userId, couponId, tenantId int64) error
	ListUserCoupons(ctx context.Context, userId, tenantId int64, status int32) ([]domain.UserCoupon, error)
	UseCoupon(ctx context.Context, userCouponId, orderId int64) error
	ReleaseCoupon(ctx context.Context, userCouponId int64) error
	FindUserCouponById(ctx context.Context, id int64) (domain.UserCoupon, error)
	CountUserCoupon(ctx context.Context, userId, couponId int64) (int64, error)
	// 秒杀
	CreateSeckillActivity(ctx context.Context, a domain.SeckillActivity) (domain.SeckillActivity, error)
	UpdateSeckillActivity(ctx context.Context, a domain.SeckillActivity) error
	ListSeckillActivities(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SeckillActivity, int64, error)
	FindSeckillActivityById(ctx context.Context, id int64) (domain.SeckillActivity, error)
	FindSeckillItemById(ctx context.Context, id int64) (domain.SeckillItem, error)
	Seckill(ctx context.Context, itemId, userId int64, perLimit int32) (int64, error)
	CreateSeckillOrder(ctx context.Context, o domain.SeckillOrder) (domain.SeckillOrder, error)
	// 满减
	CreatePromotionRule(ctx context.Context, r domain.PromotionRule) (domain.PromotionRule, error)
	UpdatePromotionRule(ctx context.Context, r domain.PromotionRule) error
	ListPromotionRules(ctx context.Context, tenantId int64, status int32) ([]domain.PromotionRule, error)
	ListActivePromotionRules(ctx context.Context, tenantId int64) ([]domain.PromotionRule, error)
}

type marketingRepository struct {
	couponDAO    dao.CouponDAO
	ucDAO        dao.UserCouponDAO
	seckillDAO   dao.SeckillDAO
	promotionDAO dao.PromotionDAO
	cache        cache.MarketingCache
}

func NewMarketingRepository(
	couponDAO dao.CouponDAO,
	ucDAO dao.UserCouponDAO,
	seckillDAO dao.SeckillDAO,
	promotionDAO dao.PromotionDAO,
	c cache.MarketingCache,
) MarketingRepository {
	return &marketingRepository{
		couponDAO:    couponDAO,
		ucDAO:        ucDAO,
		seckillDAO:   seckillDAO,
		promotionDAO: promotionDAO,
		cache:        c,
	}
}

// ==================== 优惠券 ====================

func (r *marketingRepository) CreateCoupon(ctx context.Context, c domain.Coupon) (domain.Coupon, error) {
	dc, err := r.couponDAO.Insert(ctx, r.couponToDAO(c))
	if err != nil {
		return domain.Coupon{}, err
	}
	// 初始化 Redis 库存
	_ = r.cache.SetCouponStock(ctx, dc.ID, dc.TotalCount-dc.ReceivedCount)
	return r.couponToDomain(dc), nil
}

func (r *marketingRepository) UpdateCoupon(ctx context.Context, c domain.Coupon) error {
	return r.couponDAO.Update(ctx, r.couponToDAO(c))
}

func (r *marketingRepository) ListCoupons(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.Coupon, int64, error) {
	offset := int((page - 1) * pageSize)
	coupons, total, err := r.couponDAO.List(ctx, tenantId, status, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	result := make([]domain.Coupon, 0, len(coupons))
	for _, c := range coupons {
		result = append(result, r.couponToDomain(c))
	}
	return result, total, nil
}

func (r *marketingRepository) FindCouponById(ctx context.Context, id int64) (domain.Coupon, error) {
	c, err := r.couponDAO.FindById(ctx, id)
	if err != nil {
		return domain.Coupon{}, err
	}
	return r.couponToDomain(c), nil
}

func (r *marketingRepository) ReceiveCoupon(ctx context.Context, userId, couponId, tenantId int64) error {
	// Redis 原子扣减库存
	remaining, err := r.cache.DecrCouponStock(ctx, couponId)
	if err != nil {
		return err
	}
	if remaining < 0 {
		// 回补库存
		_ = r.cache.IncrCouponStock(ctx, couponId)
		return ErrCouponStockNotEnough
	}
	// MySQL 写入领取记录
	_, err = r.ucDAO.Insert(ctx, dao.UserCoupon{
		UserID:   userId,
		CouponID: couponId,
		TenantID: tenantId,
		Status:   1,
	})
	if err != nil {
		// 回补 Redis 库存
		_ = r.cache.IncrCouponStock(ctx, couponId)
		return err
	}
	// 更新 MySQL 已领取计数
	_ = r.couponDAO.IncrReceivedCount(ctx, couponId)
	return nil
}

func (r *marketingRepository) ListUserCoupons(ctx context.Context, userId, tenantId int64, status int32) ([]domain.UserCoupon, error) {
	ucs, err := r.ucDAO.ListByUser(ctx, userId, tenantId, status)
	if err != nil {
		return nil, err
	}
	result := make([]domain.UserCoupon, 0, len(ucs))
	for _, uc := range ucs {
		duc := r.userCouponToDomain(uc)
		// 查询关联的优惠券信息
		coupon, err := r.couponDAO.FindById(ctx, uc.CouponID)
		if err == nil {
			duc.Coupon = r.couponToDomain(coupon)
		}
		result = append(result, duc)
	}
	return result, nil
}

func (r *marketingRepository) UseCoupon(ctx context.Context, userCouponId, orderId int64) error {
	return r.ucDAO.UpdateStatus(ctx, userCouponId, 2, orderId)
}

func (r *marketingRepository) ReleaseCoupon(ctx context.Context, userCouponId int64) error {
	// 查出 user_coupon 获取 coupon_id
	uc, err := r.ucDAO.FindById(ctx, userCouponId)
	if err != nil {
		return err
	}
	// 更新状态为未使用
	err = r.ucDAO.UpdateStatus(ctx, userCouponId, 1, 0)
	if err != nil {
		return err
	}
	// Redis 回补库存
	_ = r.cache.IncrCouponStock(ctx, uc.CouponID)
	// MySQL 扣减已使用计数
	_ = r.couponDAO.DecrUsedCount(ctx, uc.CouponID)
	return nil
}

func (r *marketingRepository) FindUserCouponById(ctx context.Context, id int64) (domain.UserCoupon, error) {
	uc, err := r.ucDAO.FindById(ctx, id)
	if err != nil {
		return domain.UserCoupon{}, err
	}
	return r.userCouponToDomain(uc), nil
}

func (r *marketingRepository) CountUserCoupon(ctx context.Context, userId, couponId int64) (int64, error) {
	return r.ucDAO.CountByUserAndCoupon(ctx, userId, couponId)
}

// ==================== 秒杀 ====================

func (r *marketingRepository) CreateSeckillActivity(ctx context.Context, a domain.SeckillActivity) (domain.SeckillActivity, error) {
	items := make([]dao.SeckillItem, 0, len(a.Items))
	for _, item := range a.Items {
		items = append(items, r.seckillItemToDAO(item))
	}
	da, err := r.seckillDAO.InsertActivity(ctx, r.seckillActivityToDAO(a), items)
	if err != nil {
		return domain.SeckillActivity{}, err
	}
	// 初始化 Redis 秒杀库存
	for _, item := range items {
		_ = r.cache.SetSeckillStock(ctx, item.ID, item.SeckillStock)
	}
	return r.seckillActivityToDomain(da, items), nil
}

func (r *marketingRepository) UpdateSeckillActivity(ctx context.Context, a domain.SeckillActivity) error {
	items := make([]dao.SeckillItem, 0, len(a.Items))
	for _, item := range a.Items {
		items = append(items, r.seckillItemToDAO(item))
	}
	return r.seckillDAO.UpdateActivity(ctx, r.seckillActivityToDAO(a), items)
}

func (r *marketingRepository) ListSeckillActivities(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SeckillActivity, int64, error) {
	offset := int((page - 1) * pageSize)
	activities, total, err := r.seckillDAO.ListActivities(ctx, tenantId, status, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	result := make([]domain.SeckillActivity, 0, len(activities))
	for _, a := range activities {
		result = append(result, r.seckillActivityToDomain(a, nil))
	}
	return result, total, nil
}

func (r *marketingRepository) FindSeckillActivityById(ctx context.Context, id int64) (domain.SeckillActivity, error) {
	a, items, err := r.seckillDAO.FindActivityById(ctx, id)
	if err != nil {
		return domain.SeckillActivity{}, err
	}
	return r.seckillActivityToDomain(a, items), nil
}

func (r *marketingRepository) FindSeckillItemById(ctx context.Context, id int64) (domain.SeckillItem, error) {
	item, err := r.seckillDAO.FindItemById(ctx, id)
	if err != nil {
		return domain.SeckillItem{}, err
	}
	return r.seckillItemToDomain(item), nil
}

func (r *marketingRepository) Seckill(ctx context.Context, itemId, userId int64, perLimit int32) (int64, error) {
	return r.cache.Seckill(ctx, itemId, userId, perLimit)
}

func (r *marketingRepository) CreateSeckillOrder(ctx context.Context, o domain.SeckillOrder) (domain.SeckillOrder, error) {
	do, err := r.seckillDAO.InsertSeckillOrder(ctx, dao.SeckillOrder{
		UserID:   o.UserID,
		ItemID:   o.ItemID,
		TenantID: o.TenantID,
		OrderNo:  o.OrderNo,
		Status:   o.Status,
	})
	if err != nil {
		return domain.SeckillOrder{}, err
	}
	o.ID = do.ID
	return o, nil
}

// ==================== 满减 ====================

func (r *marketingRepository) CreatePromotionRule(ctx context.Context, rule domain.PromotionRule) (domain.PromotionRule, error) {
	dr, err := r.promotionDAO.Insert(ctx, r.promotionToDAO(rule))
	if err != nil {
		return domain.PromotionRule{}, err
	}
	return r.promotionToDomain(dr), nil
}

func (r *marketingRepository) UpdatePromotionRule(ctx context.Context, rule domain.PromotionRule) error {
	return r.promotionDAO.Update(ctx, r.promotionToDAO(rule))
}

func (r *marketingRepository) ListPromotionRules(ctx context.Context, tenantId int64, status int32) ([]domain.PromotionRule, error) {
	rules, err := r.promotionDAO.List(ctx, tenantId, status)
	if err != nil {
		return nil, err
	}
	result := make([]domain.PromotionRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, r.promotionToDomain(rule))
	}
	return result, nil
}

func (r *marketingRepository) ListActivePromotionRules(ctx context.Context, tenantId int64) ([]domain.PromotionRule, error) {
	rules, err := r.promotionDAO.ListActive(ctx, tenantId, time.Now().UnixMilli())
	if err != nil {
		return nil, err
	}
	result := make([]domain.PromotionRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, r.promotionToDomain(rule))
	}
	return result, nil
}

// ==================== Converters ====================

func (r *marketingRepository) couponToDAO(c domain.Coupon) dao.Coupon {
	return dao.Coupon{
		ID: c.ID, TenantID: c.TenantID, Name: c.Name, Type: c.Type,
		Threshold: c.Threshold, DiscountValue: c.DiscountValue,
		TotalCount: c.TotalCount, ReceivedCount: c.ReceivedCount,
		UsedCount: c.UsedCount, PerLimit: c.PerLimit,
		StartTime: c.StartTime.UnixMilli(), EndTime: c.EndTime.UnixMilli(),
		ScopeType: c.ScopeType, ScopeIDs: c.ScopeIDs, Status: c.Status,
	}
}

func (r *marketingRepository) couponToDomain(c dao.Coupon) domain.Coupon {
	return domain.Coupon{
		ID: c.ID, TenantID: c.TenantID, Name: c.Name, Type: c.Type,
		Threshold: c.Threshold, DiscountValue: c.DiscountValue,
		TotalCount: c.TotalCount, ReceivedCount: c.ReceivedCount,
		UsedCount: c.UsedCount, PerLimit: c.PerLimit,
		StartTime: time.UnixMilli(c.StartTime), EndTime: time.UnixMilli(c.EndTime),
		ScopeType: c.ScopeType, ScopeIDs: c.ScopeIDs, Status: c.Status,
		Ctime: time.UnixMilli(c.Ctime),
	}
}

func (r *marketingRepository) userCouponToDomain(uc dao.UserCoupon) domain.UserCoupon {
	return domain.UserCoupon{
		ID: uc.ID, UserID: uc.UserID, CouponID: uc.CouponID,
		TenantID: uc.TenantID, Status: uc.Status, OrderID: uc.OrderID,
		ReceiveTime: time.UnixMilli(uc.ReceiveTime), UseTime: time.UnixMilli(uc.UseTime),
	}
}

func (r *marketingRepository) seckillActivityToDAO(a domain.SeckillActivity) dao.SeckillActivity {
	return dao.SeckillActivity{
		ID: a.ID, TenantID: a.TenantID, Name: a.Name,
		StartTime: a.StartTime.UnixMilli(), EndTime: a.EndTime.UnixMilli(),
		Status: a.Status,
	}
}

func (r *marketingRepository) seckillActivityToDomain(a dao.SeckillActivity, items []dao.SeckillItem) domain.SeckillActivity {
	domainItems := make([]domain.SeckillItem, 0, len(items))
	for _, item := range items {
		domainItems = append(domainItems, r.seckillItemToDomain(item))
	}
	return domain.SeckillActivity{
		ID: a.ID, TenantID: a.TenantID, Name: a.Name,
		StartTime: time.UnixMilli(a.StartTime), EndTime: time.UnixMilli(a.EndTime),
		Status: a.Status, Items: domainItems,
	}
}

func (r *marketingRepository) seckillItemToDAO(item domain.SeckillItem) dao.SeckillItem {
	return dao.SeckillItem{
		ID: item.ID, ActivityID: item.ActivityID, TenantID: item.TenantID,
		SkuID: item.SkuID, SeckillPrice: item.SeckillPrice,
		SeckillStock: item.SeckillStock, PerLimit: item.PerLimit,
	}
}

func (r *marketingRepository) seckillItemToDomain(item dao.SeckillItem) domain.SeckillItem {
	return domain.SeckillItem{
		ID: item.ID, ActivityID: item.ActivityID, TenantID: item.TenantID,
		SkuID: item.SkuID, SeckillPrice: item.SeckillPrice,
		SeckillStock: item.SeckillStock, PerLimit: item.PerLimit,
	}
}

func (r *marketingRepository) promotionToDAO(rule domain.PromotionRule) dao.PromotionRule {
	return dao.PromotionRule{
		ID: rule.ID, TenantID: rule.TenantID, Name: rule.Name, Type: rule.Type,
		Threshold: rule.Threshold, DiscountValue: rule.DiscountValue,
		StartTime: rule.StartTime.UnixMilli(), EndTime: rule.EndTime.UnixMilli(),
		Status: rule.Status,
	}
}

func (r *marketingRepository) promotionToDomain(rule dao.PromotionRule) domain.PromotionRule {
	return domain.PromotionRule{
		ID: rule.ID, TenantID: rule.TenantID, Name: rule.Name, Type: rule.Type,
		Threshold: rule.Threshold, DiscountValue: rule.DiscountValue,
		StartTime: time.UnixMilli(rule.StartTime), EndTime: time.UnixMilli(rule.EndTime),
		Status: rule.Status,
	}
}
