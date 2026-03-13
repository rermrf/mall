package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/mall/pkg/ginx"
)

func AdminOnly() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tenantId, exists := ctx.Get("tenant_id")
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusForbidden, ginx.Result{
				Code: 403001,
				Msg:  "无权访问",
			})
			return
		}

		tid, ok := tenantId.(int64)
		if !ok || tid != 0 {
			ctx.AbortWithStatusJSON(http.StatusForbidden, ginx.Result{
				Code: 403001,
				Msg:  "仅平台管理员可访问",
			})
			return
		}

		ctx.Next()
	}
}
