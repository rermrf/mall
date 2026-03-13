package tenantx

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GinMiddleware 从 header X-Tenant-ID 提取 tenant_id 注入到 context
// 实际生产中 tenant_id 通常从 JWT Claims 解析
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tidStr := c.GetHeader("X-Tenant-ID")
		if tidStr == "" {
			tidStr = "0"
		}
		tid, err := strconv.ParseInt(tidStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"msg": "invalid tenant id"})
			return
		}
		ctx := WithTenantID(c.Request.Context(), tid)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
