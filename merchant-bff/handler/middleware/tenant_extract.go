package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

func TenantExtract() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tenantId, exists := ctx.Get("tenant_id")
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusForbidden, ginx.Result{
				Code: 403001,
				Msg:  "需要商家身份",
			})
			return
		}

		tid, ok := tenantId.(int64)
		if !ok || tid <= 0 {
			ctx.AbortWithStatusJSON(http.StatusForbidden, ginx.Result{
				Code: 403001,
				Msg:  "需要商家身份",
			})
			return
		}

		c := tenantx.WithTenantID(ctx.Request.Context(), tid)
		ctx.Request = ctx.Request.WithContext(c)
		ctx.Next()
	}
}
