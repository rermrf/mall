package service

import (
	"context"
	"errors"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/tenant/domain"
	"github.com/rermrf/mall/tenant/events"
	"github.com/rermrf/mall/tenant/repository"
)

var (
	ErrTenantNotFound = errors.New("租户不存在")
	ErrTenantFrozen   = errors.New("租户已被冻结")
	ErrQuotaExceeded  = errors.New("配额已超限")
	ErrPlanNotFound   = errors.New("套餐不存在")
	ErrShopNotFound   = errors.New("店铺不存在")
)

type TenantService interface {
	// Tenant
	CreateTenant(ctx context.Context, t domain.Tenant) (domain.Tenant, error)
	GetTenant(ctx context.Context, id int64) (domain.Tenant, error)
	UpdateTenant(ctx context.Context, t domain.Tenant) error
	ListTenants(ctx context.Context, page, pageSize int32, status int32) ([]domain.Tenant, int64, error)
	ApproveTenant(ctx context.Context, id int64, approved bool, reason string) error
	FreezeTenant(ctx context.Context, id int64, freeze bool) error

	// Plan
	GetPlan(ctx context.Context, id int64) (domain.TenantPlan, error)
	ListPlans(ctx context.Context) ([]domain.TenantPlan, error)
	CreatePlan(ctx context.Context, p domain.TenantPlan) (domain.TenantPlan, error)
	UpdatePlan(ctx context.Context, p domain.TenantPlan) error

	// Quota
	CheckQuota(ctx context.Context, tenantId int64, quotaType string) (bool, domain.QuotaUsage, error)
	IncrQuota(ctx context.Context, tenantId int64, quotaType string, delta int32) error
	DecrQuota(ctx context.Context, tenantId int64, quotaType string, delta int32) error

	// Shop
	GetShop(ctx context.Context, tenantId int64) (domain.Shop, error)
	UpdateShop(ctx context.Context, s domain.Shop) error
	GetShopByDomain(ctx context.Context, d string) (domain.Shop, error)
}

type tenantService struct {
	repo     repository.TenantRepository
	producer events.Producer
	l        logger.Logger
}

func NewTenantService(repo repository.TenantRepository, producer events.Producer, l logger.Logger) TenantService {
	return &tenantService{
		repo:     repo,
		producer: producer,
		l:        l,
	}
}

// ==================== Tenant ====================

func (s *tenantService) CreateTenant(ctx context.Context, t domain.Tenant) (domain.Tenant, error) {
	t.Status = domain.TenantStatusPending
	tenant, err := s.repo.CreateTenant(ctx, t)
	if err != nil {
		return domain.Tenant{}, err
	}
	// Create an empty shop for the tenant
	_, err = s.repo.CreateShop(ctx, domain.Shop{
		TenantID: tenant.ID,
		Name:     tenant.Name,
		Status:   domain.ShopStatusClosed,
	})
	if err != nil {
		s.l.Error("创建默认店铺失败", logger.Error(err), logger.Int64("tid", tenant.ID))
		// Don't fail the tenant creation
	}
	return tenant, nil
}

func (s *tenantService) GetTenant(ctx context.Context, id int64) (domain.Tenant, error) {
	return s.repo.GetTenant(ctx, id)
}

func (s *tenantService) UpdateTenant(ctx context.Context, t domain.Tenant) error {
	// Detect plan change for event
	old, err := s.repo.GetTenant(ctx, t.ID)
	if err != nil {
		return err
	}
	err = s.repo.UpdateTenant(ctx, t)
	if err != nil {
		return err
	}
	// If plan changed, produce event
	if old.PlanID != t.PlanID && t.PlanID != 0 {
		go func() {
			if er := s.producer.ProduceTenantPlanChanged(context.Background(), events.TenantPlanChangedEvent{
				TenantId:  t.ID,
				OldPlanId: old.PlanID,
				NewPlanId: t.PlanID,
			}); er != nil {
				s.l.Error("发送套餐变更事件失败", logger.Error(er), logger.Int64("tid", t.ID))
			}
		}()
	}
	return nil
}

func (s *tenantService) ListTenants(ctx context.Context, page, pageSize int32, status int32) ([]domain.Tenant, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := int((page - 1) * pageSize)
	limit := int(pageSize)
	return s.repo.ListTenants(ctx, offset, limit, uint8(status))
}

func (s *tenantService) ApproveTenant(ctx context.Context, id int64, approved bool, reason string) error {
	if !approved {
		return s.repo.UpdateTenantStatus(ctx, id, domain.TenantStatusCanceled)
	}
	// Approved: update status to Normal
	err := s.repo.UpdateTenantStatus(ctx, id, domain.TenantStatusNormal)
	if err != nil {
		return err
	}
	// Get tenant to find plan
	tenant, err := s.repo.GetTenant(ctx, id)
	if err != nil {
		s.l.Error("审核通过后获取租户失败", logger.Error(err), logger.Int64("tid", id))
		return nil
	}
	// Get plan to init quota
	plan, err := s.repo.GetPlan(ctx, tenant.PlanID)
	if err != nil {
		s.l.Error("审核通过后获取套餐失败", logger.Error(err), logger.Int64("planId", tenant.PlanID))
		return nil
	}
	// Init quota from plan
	if er := s.repo.InitQuota(ctx, id, "product_count", plan.MaxProducts); er != nil {
		s.l.Error("初始化商品配额失败", logger.Error(er), logger.Int64("tid", id))
	}
	if er := s.repo.InitQuota(ctx, id, "staff_count", plan.MaxStaff); er != nil {
		s.l.Error("初始化员工配额失败", logger.Error(er), logger.Int64("tid", id))
	}
	// Async produce tenant_approved event
	go func() {
		if er := s.producer.ProduceTenantApproved(context.Background(), events.TenantApprovedEvent{
			TenantId: id,
			Name:     tenant.Name,
			PlanId:   tenant.PlanID,
		}); er != nil {
			s.l.Error("发送租户审核通过事件失败", logger.Error(er), logger.Int64("tid", id))
		}
	}()
	return nil
}

func (s *tenantService) FreezeTenant(ctx context.Context, id int64, freeze bool) error {
	if freeze {
		return s.repo.UpdateTenantStatus(ctx, id, domain.TenantStatusFrozen)
	}
	return s.repo.UpdateTenantStatus(ctx, id, domain.TenantStatusNormal)
}

// ==================== Plan ====================

func (s *tenantService) GetPlan(ctx context.Context, id int64) (domain.TenantPlan, error) {
	return s.repo.GetPlan(ctx, id)
}

func (s *tenantService) ListPlans(ctx context.Context) ([]domain.TenantPlan, error) {
	return s.repo.ListPlans(ctx)
}

func (s *tenantService) CreatePlan(ctx context.Context, p domain.TenantPlan) (domain.TenantPlan, error) {
	return s.repo.CreatePlan(ctx, p)
}

func (s *tenantService) UpdatePlan(ctx context.Context, p domain.TenantPlan) error {
	return s.repo.UpdatePlan(ctx, p)
}

// ==================== Quota ====================

func (s *tenantService) CheckQuota(ctx context.Context, tenantId int64, quotaType string) (bool, domain.QuotaUsage, error) {
	quota, err := s.repo.CheckQuota(ctx, tenantId, quotaType)
	if err != nil {
		return false, domain.QuotaUsage{}, err
	}
	allowed := quota.Used+1 <= quota.MaxLimit
	return allowed, quota, nil
}

func (s *tenantService) IncrQuota(ctx context.Context, tenantId int64, quotaType string, delta int32) error {
	return s.repo.IncrQuota(ctx, tenantId, quotaType, delta)
}

func (s *tenantService) DecrQuota(ctx context.Context, tenantId int64, quotaType string, delta int32) error {
	return s.repo.DecrQuota(ctx, tenantId, quotaType, delta)
}

// ==================== Shop ====================

func (s *tenantService) GetShop(ctx context.Context, tenantId int64) (domain.Shop, error) {
	return s.repo.GetShop(ctx, tenantId)
}

func (s *tenantService) UpdateShop(ctx context.Context, shop domain.Shop) error {
	return s.repo.UpdateShop(ctx, shop)
}

func (s *tenantService) GetShopByDomain(ctx context.Context, d string) (domain.Shop, error) {
	return s.repo.GetShopByDomain(ctx, d)
}
