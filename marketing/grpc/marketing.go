package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	marketingv1 "github.com/rermrf/mall/api/proto/gen/marketing/v1"
	"github.com/rermrf/mall/marketing/domain"
	"github.com/rermrf/mall/marketing/service"
)

type MarketingGRPCServer struct {
	marketingv1.UnimplementedMarketingServiceServer
	svc service.MarketingService
}

func NewMarketingGRPCServer(svc service.MarketingService) *MarketingGRPCServer {
	return &MarketingGRPCServer{svc: svc}
}

func (s *MarketingGRPCServer) Register(server *grpc.Server) {
	marketingv1.RegisterMarketingServiceServer(server, s)
}

// ==================== 优惠券 ====================

func (s *MarketingGRPCServer) CreateCoupon(ctx context.Context, req *marketingv1.CreateCouponRequest) (*marketingv1.CreateCouponResponse, error) {
	c := req.GetCoupon()
	coupon, err := s.svc.CreateCoupon(ctx, domain.Coupon{
		TenantID: c.GetTenantId(), Name: c.GetName(), Type: c.GetType(),
		Threshold: c.GetThreshold(), DiscountValue: c.GetDiscountValue(),
		TotalCount: c.GetTotalCount(), PerLimit: c.GetPerLimit(),
		StartTime: time.UnixMilli(c.GetStartTime()), EndTime: time.UnixMilli(c.GetEndTime()),
		ScopeType: c.GetScopeType(), ScopeIDs: c.GetScopeIds(), Status: c.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.CreateCouponResponse{Id: coupon.ID}, nil
}

func (s *MarketingGRPCServer) UpdateCoupon(ctx context.Context, req *marketingv1.UpdateCouponRequest) (*marketingv1.UpdateCouponResponse, error) {
	c := req.GetCoupon()
	err := s.svc.UpdateCoupon(ctx, domain.Coupon{
		ID: c.GetId(), TenantID: c.GetTenantId(), Name: c.GetName(), Type: c.GetType(),
		Threshold: c.GetThreshold(), DiscountValue: c.GetDiscountValue(),
		TotalCount: c.GetTotalCount(), PerLimit: c.GetPerLimit(),
		StartTime: time.UnixMilli(c.GetStartTime()), EndTime: time.UnixMilli(c.GetEndTime()),
		ScopeType: c.GetScopeType(), ScopeIDs: c.GetScopeIds(), Status: c.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.UpdateCouponResponse{}, nil
}

func (s *MarketingGRPCServer) ListCoupons(ctx context.Context, req *marketingv1.ListCouponsRequest) (*marketingv1.ListCouponsResponse, error) {
	coupons, total, err := s.svc.ListCoupons(ctx, req.GetTenantId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	pbCoupons := make([]*marketingv1.Coupon, 0, len(coupons))
	for _, c := range coupons {
		pbCoupons = append(pbCoupons, toCouponDTO(c))
	}
	return &marketingv1.ListCouponsResponse{Coupons: pbCoupons, Total: total}, nil
}

func (s *MarketingGRPCServer) ReceiveCoupon(ctx context.Context, req *marketingv1.ReceiveCouponRequest) (*marketingv1.ReceiveCouponResponse, error) {
	err := s.svc.ReceiveCoupon(ctx, req.GetUserId(), req.GetCouponId())
	if err != nil {
		return nil, err
	}
	return &marketingv1.ReceiveCouponResponse{}, nil
}

func (s *MarketingGRPCServer) ListUserCoupons(ctx context.Context, req *marketingv1.ListUserCouponsRequest) (*marketingv1.ListUserCouponsResponse, error) {
	ucs, err := s.svc.ListUserCoupons(ctx, req.GetUserId(), req.GetTenantId(), req.GetStatus())
	if err != nil {
		return nil, err
	}
	pbUCs := make([]*marketingv1.UserCoupon, 0, len(ucs))
	for _, uc := range ucs {
		pbUCs = append(pbUCs, toUserCouponDTO(uc))
	}
	return &marketingv1.ListUserCouponsResponse{Coupons: pbUCs}, nil
}

func (s *MarketingGRPCServer) UseCoupon(ctx context.Context, req *marketingv1.UseCouponRequest) (*marketingv1.UseCouponResponse, error) {
	err := s.svc.UseCoupon(ctx, req.GetUserCouponId(), req.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &marketingv1.UseCouponResponse{}, nil
}

func (s *MarketingGRPCServer) ReleaseCoupon(ctx context.Context, req *marketingv1.ReleaseCouponRequest) (*marketingv1.ReleaseCouponResponse, error) {
	err := s.svc.ReleaseCoupon(ctx, req.GetUserCouponId())
	if err != nil {
		return nil, err
	}
	return &marketingv1.ReleaseCouponResponse{}, nil
}

func (s *MarketingGRPCServer) CalculateDiscount(ctx context.Context, req *marketingv1.CalculateDiscountRequest) (*marketingv1.CalculateDiscountResponse, error) {
	result, err := s.svc.CalculateDiscount(ctx, req.GetTenantId(), req.GetUserId(), req.GetCouponId(), req.GetTotalAmount(), req.GetCategoryIds())
	if err != nil {
		return nil, err
	}
	return &marketingv1.CalculateDiscountResponse{
		Result: &marketingv1.DiscountResult{
			CouponDiscount:    result.CouponDiscount,
			PromotionDiscount: result.PromotionDiscount,
			TotalDiscount:     result.TotalDiscount,
			PayAmount:         result.PayAmount,
		},
	}, nil
}

// ==================== 秒杀 ====================

func (s *MarketingGRPCServer) CreateSeckillActivity(ctx context.Context, req *marketingv1.CreateSeckillActivityRequest) (*marketingv1.CreateSeckillActivityResponse, error) {
	a := req.GetActivity()
	items := make([]domain.SeckillItem, 0, len(a.GetItems()))
	for _, item := range a.GetItems() {
		items = append(items, domain.SeckillItem{
			SkuID: item.GetSkuId(), SeckillPrice: item.GetSeckillPrice(),
			SeckillStock: item.GetSeckillStock(), PerLimit: item.GetPerLimit(),
		})
	}
	activity, err := s.svc.CreateSeckillActivity(ctx, domain.SeckillActivity{
		TenantID: a.GetTenantId(), Name: a.GetName(),
		StartTime: time.UnixMilli(a.GetStartTime()), EndTime: time.UnixMilli(a.GetEndTime()),
		Status: a.GetStatus(), Items: items,
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.CreateSeckillActivityResponse{Id: activity.ID}, nil
}

func (s *MarketingGRPCServer) UpdateSeckillActivity(ctx context.Context, req *marketingv1.UpdateSeckillActivityRequest) (*marketingv1.UpdateSeckillActivityResponse, error) {
	a := req.GetActivity()
	items := make([]domain.SeckillItem, 0, len(a.GetItems()))
	for _, item := range a.GetItems() {
		items = append(items, domain.SeckillItem{
			SkuID: item.GetSkuId(), SeckillPrice: item.GetSeckillPrice(),
			SeckillStock: item.GetSeckillStock(), PerLimit: item.GetPerLimit(),
		})
	}
	err := s.svc.UpdateSeckillActivity(ctx, domain.SeckillActivity{
		ID: a.GetId(), TenantID: a.GetTenantId(), Name: a.GetName(),
		StartTime: time.UnixMilli(a.GetStartTime()), EndTime: time.UnixMilli(a.GetEndTime()),
		Status: a.GetStatus(), Items: items,
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.UpdateSeckillActivityResponse{}, nil
}

func (s *MarketingGRPCServer) ListSeckillActivities(ctx context.Context, req *marketingv1.ListSeckillActivitiesRequest) (*marketingv1.ListSeckillActivitiesResponse, error) {
	activities, total, err := s.svc.ListSeckillActivities(ctx, req.GetTenantId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	pbActivities := make([]*marketingv1.SeckillActivity, 0, len(activities))
	for _, a := range activities {
		pbActivities = append(pbActivities, toSeckillActivityDTO(a))
	}
	return &marketingv1.ListSeckillActivitiesResponse{Activities: pbActivities, Total: total}, nil
}

func (s *MarketingGRPCServer) GetSeckillActivity(ctx context.Context, req *marketingv1.GetSeckillActivityRequest) (*marketingv1.GetSeckillActivityResponse, error) {
	activity, err := s.svc.GetSeckillActivity(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &marketingv1.GetSeckillActivityResponse{Activity: toSeckillActivityDTO(activity)}, nil
}

func (s *MarketingGRPCServer) Seckill(ctx context.Context, req *marketingv1.SeckillRequest) (*marketingv1.SeckillResponse, error) {
	success, msg, err := s.svc.Seckill(ctx, req.GetUserId(), req.GetItemId())
	if err != nil {
		return nil, err
	}
	return &marketingv1.SeckillResponse{Success: success, Message: msg}, nil
}

// ==================== 满减 ====================

func (s *MarketingGRPCServer) CreatePromotionRule(ctx context.Context, req *marketingv1.CreatePromotionRuleRequest) (*marketingv1.CreatePromotionRuleResponse, error) {
	r := req.GetRule()
	rule, err := s.svc.CreatePromotionRule(ctx, domain.PromotionRule{
		TenantID: r.GetTenantId(), Name: r.GetName(), Type: r.GetType(),
		Threshold: r.GetThreshold(), DiscountValue: r.GetDiscountValue(),
		StartTime: time.UnixMilli(r.GetStartTime()), EndTime: time.UnixMilli(r.GetEndTime()),
		Status: r.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.CreatePromotionRuleResponse{Id: rule.ID}, nil
}

func (s *MarketingGRPCServer) UpdatePromotionRule(ctx context.Context, req *marketingv1.UpdatePromotionRuleRequest) (*marketingv1.UpdatePromotionRuleResponse, error) {
	r := req.GetRule()
	err := s.svc.UpdatePromotionRule(ctx, domain.PromotionRule{
		ID: r.GetId(), TenantID: r.GetTenantId(), Name: r.GetName(), Type: r.GetType(),
		Threshold: r.GetThreshold(), DiscountValue: r.GetDiscountValue(),
		StartTime: time.UnixMilli(r.GetStartTime()), EndTime: time.UnixMilli(r.GetEndTime()),
		Status: r.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &marketingv1.UpdatePromotionRuleResponse{}, nil
}

func (s *MarketingGRPCServer) ListPromotionRules(ctx context.Context, req *marketingv1.ListPromotionRulesRequest) (*marketingv1.ListPromotionRulesResponse, error) {
	rules, err := s.svc.ListPromotionRules(ctx, req.GetTenantId(), req.GetStatus())
	if err != nil {
		return nil, err
	}
	pbRules := make([]*marketingv1.PromotionRule, 0, len(rules))
	for _, r := range rules {
		pbRules = append(pbRules, toPromotionRuleDTO(r))
	}
	return &marketingv1.ListPromotionRulesResponse{Rules: pbRules}, nil
}

// ==================== DTO Converters ====================

func toCouponDTO(c domain.Coupon) *marketingv1.Coupon {
	return &marketingv1.Coupon{
		Id: c.ID, TenantId: c.TenantID, Name: c.Name, Type: c.Type,
		Threshold: c.Threshold, DiscountValue: c.DiscountValue,
		TotalCount: c.TotalCount, ReceivedCount: c.ReceivedCount,
		UsedCount: c.UsedCount, PerLimit: c.PerLimit,
		StartTime: c.StartTime.UnixMilli(), EndTime: c.EndTime.UnixMilli(),
		ScopeType: c.ScopeType, ScopeIds: c.ScopeIDs, Status: c.Status,
		Ctime: timestamppb.New(c.Ctime),
	}
}

func toUserCouponDTO(uc domain.UserCoupon) *marketingv1.UserCoupon {
	return &marketingv1.UserCoupon{
		Id: uc.ID, UserId: uc.UserID, CouponId: uc.CouponID,
		TenantId: uc.TenantID, Status: uc.Status, OrderId: uc.OrderID,
		ReceiveTime: uc.ReceiveTime.UnixMilli(), UseTime: uc.UseTime.UnixMilli(),
		Coupon: toCouponDTO(uc.Coupon),
	}
}

func toSeckillActivityDTO(a domain.SeckillActivity) *marketingv1.SeckillActivity {
	items := make([]*marketingv1.SeckillItem, 0, len(a.Items))
	for _, item := range a.Items {
		items = append(items, &marketingv1.SeckillItem{
			Id: item.ID, ActivityId: item.ActivityID, TenantId: item.TenantID,
			SkuId: item.SkuID, SeckillPrice: item.SeckillPrice,
			SeckillStock: item.SeckillStock, PerLimit: item.PerLimit,
		})
	}
	return &marketingv1.SeckillActivity{
		Id: a.ID, TenantId: a.TenantID, Name: a.Name,
		StartTime: a.StartTime.UnixMilli(), EndTime: a.EndTime.UnixMilli(),
		Status: a.Status, Items: items,
	}
}

func toPromotionRuleDTO(r domain.PromotionRule) *marketingv1.PromotionRule {
	return &marketingv1.PromotionRule{
		Id: r.ID, TenantId: r.TenantID, Name: r.Name, Type: r.Type,
		Threshold: r.Threshold, DiscountValue: r.DiscountValue,
		StartTime: r.StartTime.UnixMilli(), EndTime: r.EndTime.UnixMilli(),
		Status: r.Status,
	}
}
