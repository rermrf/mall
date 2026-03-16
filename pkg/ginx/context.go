package ginx

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// GetUID safely extracts uid from Gin context (set by JWT middleware).
func GetUID(ctx *gin.Context) (int64, error) {
	val, exists := ctx.Get("uid")
	if !exists {
		return 0, fmt.Errorf("uid not found in context")
	}
	uid, ok := val.(int64)
	if !ok {
		return 0, fmt.Errorf("uid type assertion failed: got %T", val)
	}
	return uid, nil
}

// GetTenantID safely extracts tenant_id from Gin context (set by JWT middleware).
func GetTenantID(ctx *gin.Context) (int64, error) {
	val, exists := ctx.Get("tenant_id")
	if !exists {
		return 0, fmt.Errorf("tenant_id not found in context")
	}
	tid, ok := val.(int64)
	if !ok {
		return 0, fmt.Errorf("tenant_id type assertion failed: got %T", val)
	}
	return tid, nil
}

// MustGetUID extracts uid, returning (0, Result) on failure for use in WrapBody handlers.
func MustGetUID(ctx *gin.Context) (int64, *Result) {
	uid, err := GetUID(ctx)
	if err != nil {
		return 0, &Result{Code: CodeUnauthorized, Msg: "未授权"}
	}
	return uid, nil
}

// MustGetTenantID extracts tenant_id, returning (0, Result) on failure for use in WrapBody handlers.
func MustGetTenantID(ctx *gin.Context) (int64, *Result) {
	tid, err := GetTenantID(ctx)
	if err != nil {
		return 0, &Result{Code: CodeForbidden, Msg: "需要商家身份"}
	}
	return tid, nil
}
