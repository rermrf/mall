package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

func TenantExtract() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tid, errResult := ginx.MustGetTenantID(ctx)
		if errResult != nil {
			ctx.AbortWithStatusJSON(http.StatusForbidden, *errResult)
			return
		}
		if tid <= 0 {
			ctx.AbortWithStatusJSON(http.StatusForbidden, ginx.Result{
				Code: ginx.CodeForbidden,
				Msg:  "需要商家身份",
			})
			return
		}

		c := tenantx.WithTenantID(ctx.Request.Context(), tid)
		ctx.Request = ctx.Request.WithContext(c)
		ctx.Next()
	}
}
