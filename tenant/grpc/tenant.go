package grpc

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/tenant/domain"
	"github.com/rermrf/mall/tenant/service"
)

type TenantGRPCServer struct {
	tenantv1.UnimplementedTenantServiceServer
	svc service.TenantService
}

func NewTenantGRPCServer(svc service.TenantService) *TenantGRPCServer {
	return &TenantGRPCServer{svc: svc}
}

func (s *TenantGRPCServer) Register(server *grpc.Server) {
	tenantv1.RegisterTenantServiceServer(server, s)
}

// ==================== Tenant ====================

func (s *TenantGRPCServer) CreateTenant(ctx context.Context, req *tenantv1.CreateTenantRequest) (*tenantv1.CreateTenantResponse, error) {
	t := req.GetTenant()
	tenant, err := s.svc.CreateTenant(ctx, domain.Tenant{
		Name:            t.GetName(),
		ContactName:     t.GetContactName(),
		ContactPhone:    t.GetContactPhone(),
		BusinessLicense: t.GetBusinessLicense(),
		PlanID:          t.GetPlanId(),
		PlanExpireTime:  t.GetPlanExpireTime(),
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.CreateTenantResponse{Id: tenant.ID}, nil
}

func (s *TenantGRPCServer) GetTenant(ctx context.Context, req *tenantv1.GetTenantRequest) (*tenantv1.GetTenantResponse, error) {
	tenant, err := s.svc.GetTenant(ctx, req.GetId())
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.GetTenantResponse{Tenant: toTenantDTO(tenant)}, nil
}

func (s *TenantGRPCServer) UpdateTenant(ctx context.Context, req *tenantv1.UpdateTenantRequest) (*tenantv1.UpdateTenantResponse, error) {
	t := req.GetTenant()
	err := s.svc.UpdateTenant(ctx, domain.Tenant{
		ID:              t.GetId(),
		Name:            t.GetName(),
		ContactName:     t.GetContactName(),
		ContactPhone:    t.GetContactPhone(),
		BusinessLicense: t.GetBusinessLicense(),
		Status:          domain.TenantStatus(t.GetStatus()),
		PlanID:          t.GetPlanId(),
		PlanExpireTime:  t.GetPlanExpireTime(),
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.UpdateTenantResponse{}, nil
}

func (s *TenantGRPCServer) ListTenants(ctx context.Context, req *tenantv1.ListTenantsRequest) (*tenantv1.ListTenantsResponse, error) {
	tenants, total, err := s.svc.ListTenants(ctx, req.GetPage(), req.GetPageSize(), req.GetStatus())
	if err != nil {
		return nil, handleErr(err)
	}
	dtos := make([]*tenantv1.Tenant, 0, len(tenants))
	for _, t := range tenants {
		dtos = append(dtos, toTenantDTO(t))
	}
	return &tenantv1.ListTenantsResponse{Tenants: dtos, Total: total}, nil
}

func (s *TenantGRPCServer) ApproveTenant(ctx context.Context, req *tenantv1.ApproveTenantRequest) (*tenantv1.ApproveTenantResponse, error) {
	err := s.svc.ApproveTenant(ctx, req.GetId(), req.GetApproved(), req.GetReason())
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.ApproveTenantResponse{}, nil
}

func (s *TenantGRPCServer) FreezeTenant(ctx context.Context, req *tenantv1.FreezeTenantRequest) (*tenantv1.FreezeTenantResponse, error) {
	err := s.svc.FreezeTenant(ctx, req.GetId(), req.GetFreeze())
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.FreezeTenantResponse{}, nil
}

// ==================== Plan ====================

func (s *TenantGRPCServer) GetPlan(ctx context.Context, req *tenantv1.GetPlanRequest) (*tenantv1.GetPlanResponse, error) {
	plan, err := s.svc.GetPlan(ctx, req.GetId())
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.GetPlanResponse{Plan: toPlanDTO(plan)}, nil
}

func (s *TenantGRPCServer) ListPlans(ctx context.Context, req *tenantv1.ListPlansRequest) (*tenantv1.ListPlansResponse, error) {
	plans, err := s.svc.ListPlans(ctx)
	if err != nil {
		return nil, handleErr(err)
	}
	dtos := make([]*tenantv1.TenantPlan, 0, len(plans))
	for _, p := range plans {
		dtos = append(dtos, toPlanDTO(p))
	}
	return &tenantv1.ListPlansResponse{Plans: dtos}, nil
}

func (s *TenantGRPCServer) CreatePlan(ctx context.Context, req *tenantv1.CreatePlanRequest) (*tenantv1.CreatePlanResponse, error) {
	p := req.GetPlan()
	plan, err := s.svc.CreatePlan(ctx, domain.TenantPlan{
		Name:         p.GetName(),
		Price:        p.GetPrice(),
		DurationDays: p.GetDurationDays(),
		MaxProducts:  p.GetMaxProducts(),
		MaxStaff:     p.GetMaxStaff(),
		Features:     p.GetFeatures(),
		Status:       domain.PlanStatus(p.GetStatus()),
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.CreatePlanResponse{Id: plan.ID}, nil
}

func (s *TenantGRPCServer) UpdatePlan(ctx context.Context, req *tenantv1.UpdatePlanRequest) (*tenantv1.UpdatePlanResponse, error) {
	p := req.GetPlan()
	err := s.svc.UpdatePlan(ctx, domain.TenantPlan{
		ID:           p.GetId(),
		Name:         p.GetName(),
		Price:        p.GetPrice(),
		DurationDays: p.GetDurationDays(),
		MaxProducts:  p.GetMaxProducts(),
		MaxStaff:     p.GetMaxStaff(),
		Features:     p.GetFeatures(),
		Status:       domain.PlanStatus(p.GetStatus()),
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.UpdatePlanResponse{}, nil
}

// ==================== Quota ====================

func (s *TenantGRPCServer) CheckQuota(ctx context.Context, req *tenantv1.CheckQuotaRequest) (*tenantv1.CheckQuotaResponse, error) {
	allowed, usage, err := s.svc.CheckQuota(ctx, req.GetTenantId(), req.GetQuotaType())
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.CheckQuotaResponse{
		Allowed: allowed,
		Usage:   toQuotaDTO(usage),
	}, nil
}

func (s *TenantGRPCServer) IncrQuota(ctx context.Context, req *tenantv1.IncrQuotaRequest) (*tenantv1.IncrQuotaResponse, error) {
	err := s.svc.IncrQuota(ctx, req.GetTenantId(), req.GetQuotaType(), req.GetDelta())
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.IncrQuotaResponse{}, nil
}

func (s *TenantGRPCServer) DecrQuota(ctx context.Context, req *tenantv1.DecrQuotaRequest) (*tenantv1.DecrQuotaResponse, error) {
	err := s.svc.DecrQuota(ctx, req.GetTenantId(), req.GetQuotaType(), req.GetDelta())
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.DecrQuotaResponse{}, nil
}

// ==================== Shop ====================

func (s *TenantGRPCServer) GetShop(ctx context.Context, req *tenantv1.GetShopRequest) (*tenantv1.GetShopResponse, error) {
	shop, err := s.svc.GetShop(ctx, req.GetTenantId())
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.GetShopResponse{Shop: toShopDTO(shop)}, nil
}

func (s *TenantGRPCServer) UpdateShop(ctx context.Context, req *tenantv1.UpdateShopRequest) (*tenantv1.UpdateShopResponse, error) {
	sh := req.GetShop()
	err := s.svc.UpdateShop(ctx, domain.Shop{
		ID:           sh.GetId(),
		TenantID:     sh.GetTenantId(),
		Name:         sh.GetName(),
		Logo:         sh.GetLogo(),
		Description:  sh.GetDescription(),
		Status:       domain.ShopStatus(sh.GetStatus()),
		Rating:       sh.GetRating(),
		Subdomain:    sh.GetSubdomain(),
		CustomDomain: sh.GetCustomDomain(),
	})
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.UpdateShopResponse{}, nil
}

func (s *TenantGRPCServer) GetShopByDomain(ctx context.Context, req *tenantv1.GetShopByDomainRequest) (*tenantv1.GetShopByDomainResponse, error) {
	shop, err := s.svc.GetShopByDomain(ctx, req.GetDomain())
	if err != nil {
		return nil, handleErr(err)
	}
	return &tenantv1.GetShopByDomainResponse{Shop: toShopDTO(shop)}, nil
}

// ==================== DTO 转换 ====================

func toTenantDTO(t domain.Tenant) *tenantv1.Tenant {
	return &tenantv1.Tenant{
		Id:              t.ID,
		Name:            t.Name,
		ContactName:     t.ContactName,
		ContactPhone:    t.ContactPhone,
		BusinessLicense: t.BusinessLicense,
		Status:          int32(t.Status),
		PlanId:          t.PlanID,
		PlanExpireTime:  t.PlanExpireTime,
		Ctime:           timestamppb.New(t.Ctime),
		Utime:           timestamppb.New(t.Utime),
	}
}

func toPlanDTO(p domain.TenantPlan) *tenantv1.TenantPlan {
	return &tenantv1.TenantPlan{
		Id:           p.ID,
		Name:         p.Name,
		Price:        p.Price,
		DurationDays: p.DurationDays,
		MaxProducts:  p.MaxProducts,
		MaxStaff:     p.MaxStaff,
		Features:     p.Features,
		Status:       int32(p.Status),
		Ctime:        timestamppb.New(time.UnixMilli(p.Ctime)),
		Utime:        timestamppb.New(time.UnixMilli(p.Utime)),
	}
}

func toShopDTO(s domain.Shop) *tenantv1.Shop {
	return &tenantv1.Shop{
		Id:           s.ID,
		TenantId:     s.TenantID,
		Name:         s.Name,
		Logo:         s.Logo,
		Description:  s.Description,
		Status:       int32(s.Status),
		Rating:       s.Rating,
		Subdomain:    s.Subdomain,
		CustomDomain: s.CustomDomain,
		Ctime:        timestamppb.New(time.UnixMilli(s.Ctime)),
		Utime:        timestamppb.New(time.UnixMilli(s.Utime)),
	}
}

func toQuotaDTO(q domain.QuotaUsage) *tenantv1.QuotaUsage {
	return &tenantv1.QuotaUsage{
		QuotaType: q.QuotaType,
		Used:      q.Used,
		MaxLimit:  q.MaxLimit,
	}
}

// ==================== 错误处理 ====================

func handleErr(err error) error {
	switch {
	case errors.Is(err, service.ErrTenantNotFound), errors.Is(err, service.ErrShopNotFound), errors.Is(err, service.ErrPlanNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrTenantFrozen):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, service.ErrQuotaExceeded):
		return status.Error(codes.ResourceExhausted, err.Error())
	default:
		return status.Errorf(codes.Internal, "内部错误: %v", err)
	}
}
