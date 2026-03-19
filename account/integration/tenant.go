package integration

import (
	"context"
	"fmt"

	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
)

type TenantIntegration interface {
	GetCommissionRate(ctx context.Context, tenantId int64) (int32, error)
}

type tenantIntegration struct {
	client tenantv1.TenantServiceClient
}

func NewTenantIntegration(client tenantv1.TenantServiceClient) TenantIntegration {
	return &tenantIntegration{client: client}
}

func (t *tenantIntegration) GetCommissionRate(ctx context.Context, tenantId int64) (int32, error) {
	resp, err := t.client.GetTenant(ctx, &tenantv1.GetTenantRequest{Id: tenantId})
	if err != nil {
		return 0, fmt.Errorf("获取租户信息失败: %w", err)
	}
	tenant := resp.GetTenant()
	if tenant == nil {
		return 0, fmt.Errorf("租户不存在: %d", tenantId)
	}
	planId := tenant.GetPlanId()
	if planId <= 0 {
		return 0, nil
	}
	planResp, err := t.client.GetPlan(ctx, &tenantv1.GetPlanRequest{Id: planId})
	if err != nil {
		return 0, fmt.Errorf("获取租户套餐失败: %w", err)
	}
	plan := planResp.GetPlan()
	if plan == nil {
		return 0, nil
	}
	return plan.GetCommissionRate(), nil
}
