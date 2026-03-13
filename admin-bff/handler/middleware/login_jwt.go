package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	ijwt "github.com/rermrf/mall/admin-bff/handler/jwt"
	"github.com/rermrf/mall/pkg/ginx"
)

type LoginJWTBuilder struct {
	jwtHandler *ijwt.JWTHandler
}

func NewLoginJWTBuilder(jwtHandler *ijwt.JWTHandler) *LoginJWTBuilder {
	return &LoginJWTBuilder{
		jwtHandler: jwtHandler,
	}
}

func (b *LoginJWTBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ginx.Result{
				Code: 401001,
				Msg:  "未登录",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ginx.Result{
				Code: 401001,
				Msg:  "token 格式错误",
			})
			return
		}

		tokenStr := parts[1]
		claims, err := b.jwtHandler.ParseAccessToken(tokenStr)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ginx.Result{
				Code: 401001,
				Msg:  "token 无效或已过期",
			})
			return
		}

		// 检查 token 是否已被登出（黑名单）
		if claims.ID != "" && b.jwtHandler.IsTokenBlacklisted(ctx.Request.Context(), claims.ID) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ginx.Result{
				Code: 401002,
				Msg:  "token 已失效，请重新登录",
			})
			return
		}

		ctx.Set("claims", claims)
		ctx.Set("uid", claims.Uid)
		ctx.Set("tenant_id", claims.TenantId)
		ctx.Next()
	}
}
