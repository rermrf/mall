package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

const (
	tenantDomainKeyPrefix = "tenant:domain:"
	tenantCacheTTL        = 10 * time.Minute
	tenantNegativeTTL     = 1 * time.Minute
	tenantNegativeValue   = "0"
)

type TenantResolveBuilder struct {
	tenantClient tenantv1.TenantServiceClient
	redisClient  redis.Cmdable
}

func NewTenantResolve(tenantClient tenantv1.TenantServiceClient, redisClient redis.Cmdable) *TenantResolveBuilder {
	return &TenantResolveBuilder{tenantClient: tenantClient, redisClient: redisClient}
}

func (b *TenantResolveBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		host := ctx.Request.Host
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		domain := host
		if strings.HasPrefix(host, "localhost") || host == "127.0.0.1" {
			if d := ctx.GetHeader("X-Tenant-Domain"); d != "" {
				domain = d
			}
		}
		if domain == "" || domain == "localhost" || domain == "127.0.0.1" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest,
				ginx.Result{Code: 4, Msg: "无法识别店铺域名"})
			return
		}

		// try Redis cache first
		cacheKey := tenantDomainKeyPrefix + domain
		val, err := b.redisClient.Get(ctx.Request.Context(), cacheKey).Result()
		if err == nil {
			tenantId, _ := strconv.ParseInt(val, 10, 64)
			if tenantId <= 0 {
				ctx.AbortWithStatusJSON(http.StatusNotFound,
					ginx.Result{Code: 404001, Msg: "店铺不存在"})
				return
			}
			c := tenantx.WithTenantID(ctx.Request.Context(), tenantId)
			ctx.Request = ctx.Request.WithContext(c)
			ctx.Set("tenant_id", tenantId)
			ctx.Next()
			return
		}

		// cache miss: fall through to gRPC
		resp, err := b.tenantClient.GetShopByDomain(ctx.Request.Context(), &tenantv1.GetShopByDomainRequest{
			Domain: domain,
		})
		if err != nil {
			if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
				_ = b.redisClient.Set(ctx.Request.Context(), cacheKey, tenantNegativeValue, tenantNegativeTTL).Err()
				ctx.AbortWithStatusJSON(http.StatusNotFound,
					ginx.Result{Code: 404001, Msg: "店铺不存在"})
				return
			}
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable,
				ginx.Result{Code: ginx.CodeSystem, Msg: "租户服务暂时不可用"})
			return
		}

		tenantId := resp.Shop.GetTenantId()
		_ = b.redisClient.Set(ctx.Request.Context(), cacheKey, fmt.Sprintf("%d", tenantId), tenantCacheTTL).Err()

		c := tenantx.WithTenantID(ctx.Request.Context(), tenantId)
		ctx.Request = ctx.Request.WithContext(c)
		ctx.Set("tenant_id", tenantId)
		ctx.Set("shop", resp.Shop)
		ctx.Next()
	}
}
