# Consumer-BFF 域名→租户 Redis 缓存 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Redis caching to TenantResolve middleware to avoid a gRPC call to tenant-service on every HTTP request.

**Architecture:** Redis stores `tenant:domain:{domain} → tenant_id` (string). Cache hit skips gRPC entirely. Cache miss falls through to gRPC, then writes cache. Negative results cached with short TTL to prevent cache penetration.

**Tech Stack:** Go, Redis (github.com/redis/go-redis), Gin middleware

**Design simplification:** The design doc proposed caching a full `domainCacheEntry` struct. However, `ctx.Set("shop", ...)` is only consumed by `GetShop` handler which has its own gRPC fallback (see `consumer-bff/handler/tenant.go:30`). All other handlers only need `tenant_id`. So we cache just the tenant_id integer, keeping implementation minimal.

---

### Task 1: Modify TenantResolveBuilder to accept Redis

**Files:**
- Modify: `consumer-bff/handler/middleware/tenant_resolve.go`

**Step 1: Update struct and constructor**

Replace the current struct and constructor:

```go
package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

const (
	tenantDomainKeyPrefix = "tenant:domain:"
	tenantCacheTTL        = 10 * time.Minute
	tenantNegativeTTL     = 1 * time.Minute
	tenantNegativeValue   = "0" // marks "domain not found"
)

type TenantResolveBuilder struct {
	tenantClient tenantv1.TenantServiceClient
	redisClient  redis.Cmdable
}

func NewTenantResolve(tenantClient tenantv1.TenantServiceClient, redisClient redis.Cmdable) *TenantResolveBuilder {
	return &TenantResolveBuilder{tenantClient: tenantClient, redisClient: redisClient}
}
```

**Step 2: Replace Build() method with cache logic**

Replace the `Build()` method body. The domain extraction logic (lines 22-37) stays the same. After extracting domain, add:

```go
func (b *TenantResolveBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// === existing domain extraction (unchanged) ===
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

		// === NEW: try Redis cache first ===
		cacheKey := tenantDomainKeyPrefix + domain
		val, err := b.redisClient.Get(ctx.Request.Context(), cacheKey).Result()
		if err == nil {
			// cache hit
			tenantId, _ := strconv.ParseInt(val, 10, 64)
			if tenantId <= 0 {
				// negative cache — domain not found
				ctx.AbortWithStatusJSON(http.StatusNotFound,
					ginx.Result{Code: 404001, Msg: "店铺不存在"})
				return
			}
			c := tenantx.WithTenantID(ctx.Request.Context(), tenantId)
			ctx.Request = ctx.Request.WithContext(c)
			ctx.Set("tenant_id", tenantId)
			// note: "shop" not set on cache hit; GetShop handler has gRPC fallback
			ctx.Next()
			return
		}

		// === cache miss: fall through to gRPC ===
		resp, err := b.tenantClient.GetShopByDomain(ctx.Request.Context(), &tenantv1.GetShopByDomainRequest{
			Domain: domain,
		})
		if err != nil {
			// write negative cache to prevent penetration
			_ = b.redisClient.Set(ctx.Request.Context(), cacheKey, tenantNegativeValue, tenantNegativeTTL).Err()
			ctx.AbortWithStatusJSON(http.StatusNotFound,
				ginx.Result{Code: 404001, Msg: "店铺不存在"})
			return
		}

		tenantId := resp.Shop.GetTenantId()
		// write to cache
		_ = b.redisClient.Set(ctx.Request.Context(), cacheKey, fmt.Sprintf("%d", tenantId), tenantCacheTTL).Err()

		c := tenantx.WithTenantID(ctx.Request.Context(), tenantId)
		ctx.Request = ctx.Request.WithContext(c)
		ctx.Set("tenant_id", tenantId)
		ctx.Set("shop", resp.Shop)
		ctx.Next()
	}
}
```

**Step 3: Verify build compiles (will fail — gin.go not yet updated)**

Run: `go build ./consumer-bff/... 2>&1`
Expected: compile error about `NewTenantResolve` argument count in `ioc/gin.go`

---

### Task 2: Update ioc/gin.go to pass Redis

**Files:**
- Modify: `consumer-bff/ioc/gin.go:25-30`

**Step 1: Add redis import and parameter**

Add `redis "github.com/redis/go-redis/v9"` to imports.

Add `redisClient redis.Cmdable,` parameter to `InitGinServer` (after `tenantClient`).

**Step 2: Pass redis to NewTenantResolve**

Change line 30 from:
```go
engine.Use(middleware.NewTenantResolve(tenantClient).Build())
```
to:
```go
engine.Use(middleware.NewTenantResolve(tenantClient, redisClient).Build())
```

**Step 3: Verify build compiles (will fail — wire_gen.go stale)**

Run: `go build ./consumer-bff/... 2>&1`
Expected: compile error about `ioc.InitGinServer` argument count in `wire_gen.go`

---

### Task 3: Update wire_gen.go

**Files:**
- Modify: `consumer-bff/wire_gen.go:43`

**Step 1: Pass cmdable to InitGinServer**

The `cmdable` variable (line 20) already holds `redis.Cmdable` from `ioc.InitRedis()`. Update line 43:

From:
```go
engine := ioc.InitGinServer(jwtHandler, userHandler, tenantHandler, inventoryHandler, orderHandler, paymentHandler, cartHandler, searchHandler, marketingHandler, logisticsHandler, notificationHandler, tenantServiceClient, logger)
```

To:
```go
engine := ioc.InitGinServer(jwtHandler, userHandler, tenantHandler, inventoryHandler, orderHandler, paymentHandler, cartHandler, searchHandler, marketingHandler, logisticsHandler, notificationHandler, tenantServiceClient, cmdable, logger)
```

**Step 2: Build and vet**

Run: `go build ./consumer-bff/... 2>&1`
Expected: clean (no errors)

Run: `go vet ./consumer-bff/... 2>&1`
Expected: clean (no errors)

**Step 3: Commit**

```bash
git add consumer-bff/handler/middleware/tenant_resolve.go consumer-bff/ioc/gin.go consumer-bff/wire_gen.go
git commit -m "feat(consumer-bff): add Redis cache to TenantResolve middleware

Cache domain→tenant_id in Redis (TTL 10min) to avoid gRPC call on every
request. Negative results cached with 1min TTL to prevent penetration."
```
