package gormx

import (
	"context"
	"reflect"

	"github.com/rermrf/mall/pkg/tenantx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type skipTenantKey struct{}

// SkipTenant 标记 context 跳过租户自动过滤（用于跨租户的管理员操作等场景）
func SkipTenant(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipTenantKey{}, true)
}

func shouldSkipTenant(ctx context.Context) bool {
	val, _ := ctx.Value(skipTenantKey{}).(bool)
	return val
}

// RegisterTenantPlugin 注册 GORM 多租户回调，自动为含 tenant_id 列的表追加租户条件
func RegisterTenantPlugin(db *gorm.DB) {
	_ = db.Callback().Query().Before("gorm:query").Register("tenant:query", addTenantWhere)
	_ = db.Callback().Update().Before("gorm:update").Register("tenant:update", addTenantWhere)
	_ = db.Callback().Delete().Before("gorm:delete").Register("tenant:delete", addTenantWhere)
	_ = db.Callback().Row().Before("gorm:row").Register("tenant:row", addTenantWhere)
	_ = db.Callback().Create().Before("gorm:create").Register("tenant:create", fillTenantOnCreate)
}

// addTenantWhere 在 Query/Update/Delete/Row 操作中自动追加 WHERE tenant_id = ?
func addTenantWhere(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}
	if _, ok := db.Statement.Schema.FieldsByDBName["tenant_id"]; !ok {
		return
	}
	ctx := db.Statement.Context
	if shouldSkipTenant(ctx) {
		return
	}
	tid := tenantx.GetTenantID(ctx)
	if tid == 0 {
		return
	}
	db.Statement.AddClause(clause.Where{
		Exprs: []clause.Expression{
			clause.Eq{
				Column: clause.Column{Table: clause.CurrentTable, Name: "tenant_id"},
				Value:  tid,
			},
		},
	})
}

// fillTenantOnCreate 在 Create 操作中自动填充 tenant_id（仅当模型的 tenant_id 为零值时）
func fillTenantOnCreate(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}
	field, ok := db.Statement.Schema.FieldsByDBName["tenant_id"]
	if !ok {
		return
	}
	ctx := db.Statement.Context
	if shouldSkipTenant(ctx) {
		return
	}
	tid := tenantx.GetTenantID(ctx)
	if tid == 0 {
		return
	}
	rv := db.Statement.ReflectValue
	switch rv.Kind() {
	case reflect.Struct:
		setTenantIfZero(ctx, field, rv, tid)
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			setTenantIfZero(ctx, field, rv.Index(i), tid)
		}
	}
}

func setTenantIfZero(ctx context.Context, field *schema.Field, rv reflect.Value, tid int64) {
	val, isZero := field.ValueOf(ctx, rv)
	if isZero || val == nil || val.(int64) == 0 {
		_ = field.Set(ctx, rv, tid)
	}
}
