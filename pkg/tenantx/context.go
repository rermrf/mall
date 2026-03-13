package tenantx

import "context"

type tenantKey struct{}

// WithTenantID 将 tenant_id 注入到 context
func WithTenantID(ctx context.Context, tenantID int64) context.Context {
	return context.WithValue(ctx, tenantKey{}, tenantID)
}

// GetTenantID 从 context 获取 tenant_id，不存在返回 0
func GetTenantID(ctx context.Context) int64 {
	val, ok := ctx.Value(tenantKey{}).(int64)
	if !ok {
		return 0
	}
	return val
}
