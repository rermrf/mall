package service

import (
	"context"
	"errors"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/marketing/domain"
	"github.com/rermrf/mall/marketing/events"
	"github.com/rermrf/mall/marketing/repository"
)

var (
	ErrCouponNotFound    = errors.New("优惠券不存在")
	ErrCouponExpired     = errors.New("优惠券已过期")
	ErrCouponLimitExceed = errors.New("领取次数已达上限")
	ErrSeckillNotActive  = errors.New("秒杀活动未开始或已结束")
	ErrSeckillStockOut   = errors.New("秒杀库存不足")
	ErrSeckillDuplicate  = errors.New("已参与过该秒杀")
)

type MarketingService interface {
	// 优惠券
	CreateCoupon(ctx context.Context, c domain.Coupon) (domain.Coupon, error)
	UpdateCoupon(ctx context.Context, c domain.Coupon) error
	ListCoupons(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.Coupon, int64, error)
	ReceiveCoupon(ctx context.Context, userId, couponId int64) error
	ListUserCoupons(ctx context.Context, userId, tenantId int64, status int32) ([]domain.UserCoupon, error)
	UseCoupon(ctx context.Context, userCouponId, orderId int64) error
	ReleaseCoupon(ctx context.Context, userCouponId int64) error
	CalculateDiscount(ctx context.Context, tenantId, userId, couponId, totalAmount int64, categoryIds []int64) (domain.DiscountResult, error)
	// 秒杀
	CreateSeckillActivity(ctx context.Context, a domain.SeckillActivity) (domain.SeckillActivity, error)
	UpdateSeckillActivity(ctx context.Context, a domain.SeckillActivity) error
	ListSeckillActivities(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SeckillActivity, int64, error)
	GetSeckillActivity(ctx context.Context, id int64) (domain.SeckillActivity, error)
	Seckill(ctx context.Context, userId, itemId int64) (bool, string, error)
	// 满减
	CreatePromotionRule(ctx context.Context, r domain.PromotionRule) (domain.PromotionRule, error)
	UpdatePromotionRule(ctx context.Context, r domain.PromotionRule) error
	ListPromotionRules(ctx context.Context, tenantId int64, status int32) ([]domain.PromotionRule, error)
}

type marketingService struct {
	repo     repository.MarketingRepository
	producer events.Producer
	l        logger.Logger
}

func NewMarketingService(repo repository.MarketingRepository, producer events.Producer, l logger.Logger) MarketingService {
	return &marketingService{repo: repo, producer: producer, l: l}
}

// ==================== 优惠券 ====================

func (s *marketingService) CreateCoupon(ctx context.Context, c domain.Coupon) (domain.Coupon, error) {
	return s.repo.CreateCoupon(ctx, c)
}

func (s *marketingService) UpdateCoupon(ctx context.Context, c domain.Coupon) error {
	return s.repo.UpdateCoupon(ctx, c)
}

func (s *marketingService) ListCoupons(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.Coupon, int64, error) {
	return s.repo.ListCoupons(ctx, tenantId, status, page, pageSize)
}

func (s *marketingService) ReceiveCoupon(ctx context.Context, userId, couponId int64) error {
	coupon, err := s.repo.FindCouponById(ctx, couponId)
	if err != nil {
		return ErrCouponNotFound
	}
	// 检查优惠券状态
	if coupon.Status != 2 {
		return ErrCouponExpired
	}
	// 检查领取限制
	if coupon.PerLimit > 0 {
		count, err := s.repo.CountUserCoupon(ctx, userId, couponId)
		if err != nil {
			return err
		}
		if count >= int64(coupon.PerLimit) {
			return ErrCouponLimitExceed
		}
	}
	return s.repo.ReceiveCoupon(ctx, userId, couponId, coupon.TenantID)
}

func (s *marketingService) ListUserCoupons(ctx context.Context, userId, tenantId int64, status int32) ([]domain.UserCoupon, error) {
	return s.repo.ListUserCoupons(ctx, userId, tenantId, status)
}

func (s *marketingService) UseCoupon(ctx context.Context, userCouponId, orderId int64) error {
	return s.repo.UseCoupon(ctx, userCouponId, orderId)
}

func (s *marketingService) ReleaseCoupon(ctx context.Context, userCouponId int64) error {
	return s.repo.ReleaseCoupon(ctx, userCouponId)
}

func (s *marketingService) CalculateDiscount(ctx context.Context, tenantId, userId, couponId, totalAmount int64, categoryIds []int64) (domain.DiscountResult, error) {
	var couponDiscount int64
	// 计算优惠券优惠
	if couponId > 0 {
		uc, err := s.repo.FindUserCouponById(ctx, couponId)
		if err == nil && uc.Status == 1 {
			coupon, err := s.repo.FindCouponById(ctx, uc.CouponID)
			if err == nil {
				couponDiscount = s.calcCouponDiscount(coupon, totalAmount)
			}
		}
	}
	// 计算满减优惠
	var promotionDiscount int64
	rules, err := s.repo.ListActivePromotionRules(ctx, tenantId)
	if err == nil {
		for _, rule := range rules {
			promotionDiscount += s.calcPromotionDiscount(rule, totalAmount)
		}
	}
	totalDiscount := couponDiscount + promotionDiscount
	payAmount := totalAmount - totalDiscount
	if payAmount < 0 {
		payAmount = 0
	}
	return domain.DiscountResult{
		CouponDiscount:    couponDiscount,
		PromotionDiscount: promotionDiscount,
		TotalDiscount:     totalDiscount,
		PayAmount:         payAmount,
	}, nil
}

func (s *marketingService) calcCouponDiscount(c domain.Coupon, amount int64) int64 {
	if c.Threshold > 0 && amount < c.Threshold {
		return 0
	}
	switch c.Type {
	case 1: // 满减
		return c.DiscountValue
	case 2: // 折扣
		return amount - amount*c.DiscountValue/100
	case 3: // 无门槛
		return c.DiscountValue
	}
	return 0
}

func (s *marketingService) calcPromotionDiscount(r domain.PromotionRule, amount int64) int64 {
	if amount < r.Threshold {
		return 0
	}
	switch r.Type {
	case 1: // 满减
		return r.DiscountValue
	case 2: // 满折
		return amount - amount*r.DiscountValue/100
	}
	return 0
}

// ==================== 秒杀 ====================

func (s *marketingService) CreateSeckillActivity(ctx context.Context, a domain.SeckillActivity) (domain.SeckillActivity, error) {
	return s.repo.CreateSeckillActivity(ctx, a)
}

func (s *marketingService) UpdateSeckillActivity(ctx context.Context, a domain.SeckillActivity) error {
	return s.repo.UpdateSeckillActivity(ctx, a)
}

func (s *marketingService) ListSeckillActivities(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SeckillActivity, int64, error) {
	return s.repo.ListSeckillActivities(ctx, tenantId, status, page, pageSize)
}

func (s *marketingService) GetSeckillActivity(ctx context.Context, id int64) (domain.SeckillActivity, error) {
	return s.repo.FindSeckillActivityById(ctx, id)
}

func (s *marketingService) Seckill(ctx context.Context, userId, itemId int64) (bool, string, error) {
	// 查询秒杀商品
	item, err := s.repo.FindSeckillItemById(ctx, itemId)
	if err != nil {
		return false, "秒杀商品不存在", nil
	}
	// Redis+Lua 原子扣减
	result, err := s.repo.Seckill(ctx, itemId, userId, item.PerLimit)
	if err != nil {
		return false, "系统繁忙", err
	}
	switch result {
	case 1:
		return false, "秒杀库存不足", nil
	case 2:
		return false, "您已参与过该秒杀", nil
	case 3:
		return false, "超出限购数量", nil
	}
	// 写入秒杀订单记录
	_, err = s.repo.CreateSeckillOrder(ctx, domain.SeckillOrder{
		UserID:   userId,
		ItemID:   itemId,
		TenantID: item.TenantID,
		Status:   1, // 排队中
	})
	if err != nil {
		s.l.Error("创建秒杀订单记录失败", logger.Error(err))
	}
	// 发 Kafka 事件
	err = s.producer.ProduceSeckillSuccess(ctx, events.SeckillSuccessEvent{
		UserId:       userId,
		ItemId:       itemId,
		SkuId:        item.SkuID,
		SeckillPrice: item.SeckillPrice,
		TenantId:     item.TenantID,
	})
	if err != nil {
		s.l.Error("发送秒杀成功事件失败", logger.Error(err))
		return false, "系统繁忙", err
	}
	return true, "秒杀成功，订单创建中", nil
}

// ==================== 满减 ====================

func (s *marketingService) CreatePromotionRule(ctx context.Context, r domain.PromotionRule) (domain.PromotionRule, error) {
	return s.repo.CreatePromotionRule(ctx, r)
}

func (s *marketingService) UpdatePromotionRule(ctx context.Context, r domain.PromotionRule) error {
	return s.repo.UpdatePromotionRule(ctx, r)
}

func (s *marketingService) ListPromotionRules(ctx context.Context, tenantId int64, status int32) ([]domain.PromotionRule, error) {
	return s.repo.ListPromotionRules(ctx, tenantId, status)
}
