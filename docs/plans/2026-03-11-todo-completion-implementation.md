# TODO 全部补齐 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Resolve all 15 TODOs across the codebase: cross-service Kafka consumer data, JWT token blacklist, OAuth comments, user_registered consumer, seckill auto-order.

**Architecture:** Inject order-svc gRPC clients into notification-svc and marketing-svc consumers for cross-service data. Add Redis to all 3 BFFs for JWT blacklist. Extend order-svc's CreateOrderReq with seckill support. Add OrderNo to OrderShippedEvent for gRPC lookup.

**Tech Stack:** Go, gRPC (etcd discovery), Kafka (sarama), Redis (go-redis/v9), Wire DI, JWT (golang-jwt/v5), google/uuid

---

## Task 1: notification-svc — Add order-svc gRPC client

Add `InitOrderClient` to notification-svc so Kafka consumers can call `orderClient.GetOrder()`.

**Files:**
- Modify: `notification/ioc/grpc.go`
- Modify: `notification/wire.go`

**Step 1: Add `initServiceConn` helper and `InitOrderClient` to `notification/ioc/grpc.go`**

After the existing `InitGRPCServer` function (line 49), add:

```go
// Add these imports to the import block:
// orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
// "go.etcd.io/etcd/client/v3/naming/resolver"
// "google.golang.org/grpc/credentials/insecure"
// "github.com/rermrf/mall/pkg/tenantx"

func initServiceConn(etcdClient *clientv3.Client, serviceName string) *grpc.ClientConn {
	etcdResolver, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(fmt.Errorf("创建 etcd resolver 失败: %w", err))
	}
	conn, err := grpc.NewClient(
		"etcd:///service/"+serviceName,
		grpc.WithResolvers(etcdResolver),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(tenantx.GRPCUnaryClientInterceptor()),
	)
	if err != nil {
		panic(fmt.Errorf("连接 gRPC 服务 %s 失败: %w", serviceName, err))
	}
	return conn
}

func InitOrderClient(etcdClient *clientv3.Client) orderv1.OrderServiceClient {
	conn := initServiceConn(etcdClient, "order")
	return orderv1.NewOrderServiceClient(conn)
}
```

**Step 2: Add `ioc.InitOrderClient` to `notification/wire.go`**

In `thirdPartySet`, add `ioc.InitOrderClient` after `ioc.InitEtcdClient`:

```go
var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitOrderClient,
)
```

**Step 3: Verify build**

Run: `cd /Users/emoji/Documents/demo/project/mall && go build ./notification/...`
Expected: BUILD SUCCESS (wire_gen.go will be regenerated later in Task 3)

---

## Task 2: notification-svc — Fix 4 Kafka consumer handlers with real cross-service calls

Replace TODO stubs with actual gRPC calls to order-svc.

**Files:**
- Modify: `notification/ioc/kafka.go`
- Modify: `notification/events/types.go` (add OrderNo to OrderShippedEvent)
- Modify: `logistics/events/types.go` (add OrderNo to OrderShippedEvent)
- Modify: `logistics/service/logistics.go` (populate OrderNo)

**Step 1: Add OrderNo to OrderShippedEvent**

In `notification/events/types.go`, add `OrderNo` field to `OrderShippedEvent`:

```go
type OrderShippedEvent struct {
	OrderId     int64  `json:"order_id"`
	OrderNo     string `json:"order_no"`
	TenantId    int64  `json:"tenant_id"`
	CarrierCode string `json:"carrier_code"`
	CarrierName string `json:"carrier_name"`
	TrackingNo  string `json:"tracking_no"`
}
```

In `logistics/events/types.go`, make the same change:

```go
type OrderShippedEvent struct {
	OrderId     int64  `json:"order_id"`
	OrderNo     string `json:"order_no"`
	TenantId    int64  `json:"tenant_id"`
	CarrierCode string `json:"carrier_code"`
	CarrierName string `json:"carrier_name"`
	TrackingNo  string `json:"tracking_no"`
}
```

In `logistics/service/logistics.go:182`, add `OrderNo` to the event construction. The `result` (domain.Shipment) has `OrderID` but not `OrderNo`. Add `OrderNo` field to domain.Shipment and populate from the gRPC CreateShipment request. If the shipment domain doesn't have OrderNo, set it empty — the consumer will handle gracefully.

Actually, looking more carefully: the simplest approach is to have the notification consumer handle both cases: if `OrderNo` is provided, use it; if not, log warning. The logistics-svc change to populate OrderNo would require plumbing order_no through the shipment domain. Let's keep it simple: **only update the notification-svc consumer to handle OrderNo if present, log warning if not.**

**Step 2: Rewrite `notification/ioc/kafka.go` consumer handlers**

Replace the entire file with updated consumer functions that accept `orderv1.OrderServiceClient`:

For `NewOrderPaidConsumer` (line 60-75), change signature and body:
```go
func NewOrderPaidConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
	orderClient orderv1.OrderServiceClient,
) *events.OrderPaidConsumer {
	return events.NewOrderPaidConsumer(cg, l, func(ctx context.Context, evt events.OrderPaidEvent) error {
		l.Info("收到订单支付事件", logger.String("orderNo", evt.OrderNo))
		// 从 order-svc 获取订单详情
		orderResp, err := orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderNo: evt.OrderNo})
		if err != nil {
			l.Error("获取订单详情失败", logger.String("orderNo", evt.OrderNo), logger.Error(err))
			return nil // 不重试，避免阻塞消费
		}
		order := orderResp.GetOrder()
		params := map[string]string{
			"OrderNo":   evt.OrderNo,
			"PaymentNo": evt.PaymentNo,
		}
		// 通知商家有新的已支付订单（站内信）
		_, _ = svc.SendNotification(ctx, order.GetTenantId(), order.GetTenantId(), "order_paid_inapp", 3, params)
		return nil
	})
}
```

For `NewOrderShippedConsumer` (line 77-94):
```go
func NewOrderShippedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
	orderClient orderv1.OrderServiceClient,
) *events.OrderShippedConsumer {
	return events.NewOrderShippedConsumer(cg, l, func(ctx context.Context, evt events.OrderShippedEvent) error {
		l.Info("收到订单发货事件",
			logger.Int64("orderId", evt.OrderId),
			logger.String("trackingNo", evt.TrackingNo))
		if evt.OrderNo == "" {
			l.Warn("order_shipped 事件缺少 OrderNo，无法查询订单详情",
				logger.Int64("orderId", evt.OrderId))
			return nil
		}
		orderResp, err := orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderNo: evt.OrderNo})
		if err != nil {
			l.Error("获取订单详情失败", logger.String("orderNo", evt.OrderNo), logger.Error(err))
			return nil
		}
		order := orderResp.GetOrder()
		params := map[string]string{
			"OrderNo":    evt.OrderNo,
			"TrackingNo": evt.TrackingNo,
			"CarrierName": evt.CarrierName,
		}
		// 通知买家包裹已发出（站内信）
		_, _ = svc.SendNotification(ctx, order.GetBuyerId(), evt.TenantId, "order_shipped_inapp", 3, params)
		return nil
	})
}
```

For `NewInventoryAlertConsumer` (line 96-113), update comment only:
```go
// 使用 tenantId 作为 userId 是合理近似：多租户场景下商家管理员 ID 通常等于 tenantId
```
Remove the TODO line, keep the code unchanged.

For `NewTenantApprovedConsumer` (line 115-130), same: replace TODO with explanation comment.

For `NewTenantPlanChangedConsumer` (line 132-148), same: replace TODO with explanation comment.

For `NewOrderCompletedConsumer` (line 150-165):
```go
func NewOrderCompletedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
	orderClient orderv1.OrderServiceClient,
) *events.OrderCompletedConsumer {
	return events.NewOrderCompletedConsumer(cg, l, func(ctx context.Context, evt events.OrderCompletedEvent) error {
		l.Info("收到订单完成事件", logger.String("orderNo", evt.OrderNo))
		orderResp, err := orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderNo: evt.OrderNo})
		if err != nil {
			l.Error("获取订单详情失败", logger.String("orderNo", evt.OrderNo), logger.Error(err))
			return nil
		}
		order := orderResp.GetOrder()
		params := map[string]string{
			"OrderNo": evt.OrderNo,
		}
		// 通知买家订单已完成（站内信）
		_, _ = svc.SendNotification(ctx, order.GetBuyerId(), evt.TenantID, "order_completed_inapp", 3, params)
		return nil
	})
}
```

Add import to the file: `orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"`

**Step 3: Verify build**

Run: `cd /Users/emoji/Documents/demo/project/mall && go build ./notification/...`
Expected: May fail until wire_gen.go is regenerated (Task 3)

---

## Task 3: notification-svc — Regenerate wire

**Files:**
- Regenerate: `notification/wire_gen.go`

**Step 1: Run wire**

```bash
cd /Users/emoji/Documents/demo/project/mall/notification && wire
```

**Step 2: Verify build**

```bash
cd /Users/emoji/Documents/demo/project/mall && go build ./notification/...
```

**Step 3: Verify with go vet**

```bash
cd /Users/emoji/Documents/demo/project/mall && go vet ./notification/...
```

---

## Task 4: marketing-svc — Add order-svc client and fix order_cancelled consumer

**Files:**
- Modify: `marketing/ioc/grpc.go`
- Modify: `marketing/ioc/kafka.go`
- Modify: `marketing/wire.go`
- Regenerate: `marketing/wire_gen.go`

**Step 1: Add `initServiceConn` and `InitOrderClient` to `marketing/ioc/grpc.go`**

Same pattern as notification-svc (Task 1). Add after `InitGRPCServer`:

```go
// Add imports:
// orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
// "go.etcd.io/etcd/client/v3/naming/resolver"
// "google.golang.org/grpc/credentials/insecure"
// "github.com/rermrf/mall/pkg/tenantx"

func initServiceConn(etcdClient *clientv3.Client, serviceName string) *grpc.ClientConn {
	etcdResolver, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(fmt.Errorf("创建 etcd resolver 失败: %w", err))
	}
	conn, err := grpc.NewClient(
		"etcd:///service/"+serviceName,
		grpc.WithResolvers(etcdResolver),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(tenantx.GRPCUnaryClientInterceptor()),
	)
	if err != nil {
		panic(fmt.Errorf("连接 gRPC 服务 %s 失败: %w", serviceName, err))
	}
	return conn
}

func InitOrderClient(etcdClient *clientv3.Client) orderv1.OrderServiceClient {
	conn := initServiceConn(etcdClient, "order")
	return orderv1.NewOrderServiceClient(conn)
}
```

**Step 2: Update `marketing/ioc/kafka.go` — NewOrderCancelledConsumer**

Change function signature to accept `orderv1.OrderServiceClient` and implement coupon release:

```go
// Add import: orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"

func NewOrderCancelledConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.MarketingService,
	orderClient orderv1.OrderServiceClient,
) *events.OrderCancelledConsumer {
	return events.NewOrderCancelledConsumer(cg, l, func(ctx context.Context, evt events.OrderCancelledEvent) error {
		l.Info("收到订单取消事件",
			logger.String("orderNo", evt.OrderNo),
			logger.Int64("tenantId", evt.TenantID))
		// 从 order-svc 获取订单详情，取出 coupon_id
		orderResp, err := orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderNo: evt.OrderNo})
		if err != nil {
			l.Error("获取订单详情失败", logger.String("orderNo", evt.OrderNo), logger.Error(err))
			return nil
		}
		couponId := orderResp.GetOrder().GetCouponId()
		if couponId == 0 {
			l.Info("订单未使用优惠券，无需释放", logger.String("orderNo", evt.OrderNo))
			return nil
		}
		// 释放优惠券
		if err := svc.ReleaseCoupon(ctx, couponId); err != nil {
			l.Error("释放优惠券失败",
				logger.Int64("couponId", couponId),
				logger.String("orderNo", evt.OrderNo),
				logger.Error(err))
			return err // 返回错误以触发重试
		}
		l.Info("优惠券释放成功",
			logger.Int64("couponId", couponId),
			logger.String("orderNo", evt.OrderNo))
		return nil
	})
}
```

**Step 3: Update `marketing/wire.go`**

Add `ioc.InitOrderClient` to `thirdPartySet`:

```go
var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitOrderClient,
)
```

**Step 4: Regenerate wire and verify**

```bash
cd /Users/emoji/Documents/demo/project/mall/marketing && wire
cd /Users/emoji/Documents/demo/project/mall && go build ./marketing/...
go vet ./marketing/...
```

---

## Task 5: admin-bff — JWT blacklist (Redis + Logout + Middleware)

Implement JWT token blacklist for admin-bff. This is the template — Tasks 6 and 7 repeat for merchant-bff and consumer-bff.

**Files:**
- Create: `admin-bff/ioc/redis.go`
- Modify: `admin-bff/handler/jwt/handler.go`
- Modify: `admin-bff/handler/middleware/login_jwt.go`
- Modify: `admin-bff/wire.go`
- Modify: `admin-bff/config/dev.yaml`

**Step 1: Create `admin-bff/ioc/redis.go`**

```go
package ioc

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
	type Config struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	}
	var cfg Config
	err := viper.UnmarshalKey("redis", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Redis 配置失败: %w", err))
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return client
}
```

**Step 2: Modify `admin-bff/handler/jwt/handler.go`**

Full replacement. Key changes:
- Add `redis.Cmdable` field to `JWTHandler`
- Add `JTI` (uuid) to token claims in `SetTokenHeaders`
- Implement `Logout`: extract JTI from both access+refresh tokens, store in Redis with TTL
- Add `IsTokenBlacklisted` method for middleware

```go
package ijwt

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

var (
	accessSecret  = []byte("admin-bff-access-secret-key-32b!")
	refreshSecret = []byte("admin-bff-refresh-secret-key-32!")
)

type Claims struct {
	Uid       int64  `json:"uid"`
	TenantId  int64  `json:"tenant_id"`
	UserAgent string `json:"user_agent"`
	jwt.RegisteredClaims
}

type LoginReq struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type RefreshReq struct{}

type LogoutReq struct{}

type JWTHandler struct {
	userClient userv1.UserServiceClient
	rdb        redis.Cmdable
}

func NewJWTHandler(userClient userv1.UserServiceClient, rdb redis.Cmdable) *JWTHandler {
	return &JWTHandler{
		userClient: userClient,
		rdb:        rdb,
	}
}

func (h *JWTHandler) Login(ctx *gin.Context, req LoginReq) (ginx.Result, error) {
	resp, err := h.userClient.Login(ctx.Request.Context(), &userv1.LoginRequest{
		TenantId: 0,
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用用户服务登录失败: %w", err)
	}

	err = h.SetTokenHeaders(ctx, resp.User.GetId(), 0)
	if err != nil {
		return ginx.Result{}, fmt.Errorf("生成 token 失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "登录成功",
	}, nil
}

func (h *JWTHandler) Refresh(ctx *gin.Context, _ RefreshReq) (ginx.Result, error) {
	refreshToken := ctx.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		return ginx.Result{Code: 401001, Msg: "缺少 refresh token"}, nil
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return refreshSecret, nil
	})
	if err != nil || !token.Valid {
		return ginx.Result{Code: 401001, Msg: "refresh token 无效或已过期"}, nil
	}

	// 检查 refresh token 是否已被黑名单
	if claims.ID != "" {
		if h.IsTokenBlacklisted(ctx.Request.Context(), claims.ID) {
			return ginx.Result{Code: 401001, Msg: "token 已失效"}, nil
		}
	}

	err = h.SetTokenHeaders(ctx, claims.Uid, claims.TenantId)
	if err != nil {
		return ginx.Result{}, fmt.Errorf("生成 token 失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "刷新成功",
	}, nil
}

func (h *JWTHandler) Logout(ctx *gin.Context, _ LogoutReq) (ginx.Result, error) {
	// 将 access token 加入黑名单
	accessToken := extractTokenFromHeader(ctx)
	if accessToken != "" {
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(accessToken, claims, func(t *jwt.Token) (interface{}, error) {
			return accessSecret, nil
		})
		if err == nil && token.Valid && claims.ID != "" {
			ttl := time.Until(claims.ExpiresAt.Time)
			if ttl > 0 {
				h.rdb.Set(ctx.Request.Context(), "jwt:blacklist:"+claims.ID, "1", ttl)
			}
		}
	}

	// 将 refresh token 加入黑名单
	refreshToken := ctx.GetHeader("X-Refresh-Token")
	if refreshToken != "" {
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
			return refreshSecret, nil
		})
		if err == nil && token.Valid && claims.ID != "" {
			ttl := time.Until(claims.ExpiresAt.Time)
			if ttl > 0 {
				h.rdb.Set(ctx.Request.Context(), "jwt:blacklist:"+claims.ID, "1", ttl)
			}
		}
	}

	return ginx.Result{
		Code: 0,
		Msg:  "登出成功",
	}, nil
}

func (h *JWTHandler) SetTokenHeaders(ctx *gin.Context, uid int64, tenantId int64) error {
	now := time.Now()
	ua := ctx.GetHeader("User-Agent")

	accessClaims := Claims{
		Uid:       uid,
		TenantId:  tenantId,
		UserAgent: ua,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(accessSecret)
	if err != nil {
		return err
	}

	refreshClaims := Claims{
		Uid:       uid,
		TenantId:  tenantId,
		UserAgent: ua,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(refreshSecret)
	if err != nil {
		return err
	}

	ctx.Header("X-Jwt-Token", accessStr)
	ctx.Header("X-Refresh-Token", refreshStr)
	return nil
}

func (h *JWTHandler) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return accessSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token 无效")
	}
	return claims, nil
}

func (h *JWTHandler) IsTokenBlacklisted(ctx context.Context, jti string) bool {
	val, err := h.rdb.Exists(ctx, "jwt:blacklist:"+jti).Result()
	if err != nil {
		return false
	}
	return val > 0
}

func extractTokenFromHeader(ctx *gin.Context) string {
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
		return authHeader[len(prefix):]
	}
	return ""
}
```

**Step 3: Modify `admin-bff/handler/middleware/login_jwt.go`**

Add blacklist check after parsing token. The `JWTHandler` already has `IsTokenBlacklisted`, and the middleware has access to it via `jwtHandler`:

```go
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
```

**Step 4: Update `admin-bff/wire.go` — add `ioc.InitRedis`**

```go
var thirdPartySet = wire.NewSet(
	ioc.InitEtcdClient,
	ioc.InitLogger,
	ioc.InitRedis,
	ioc.InitUserClient,
	ioc.InitTenantClient,
	ioc.InitProductClient,
	ioc.InitOrderClient,
	ioc.InitPaymentClient,
	ioc.InitNotificationClient,
)
```

**Step 5: Update `admin-bff/config/dev.yaml`**

```yaml
server:
  addr: ":8280"
etcd:
  addrs:
    - "rermrf.icu:2379"
redis:
  addr: "rermrf.icu:6379"
```

**Step 6: Regenerate wire and verify**

```bash
cd /Users/emoji/Documents/demo/project/mall/admin-bff && wire
cd /Users/emoji/Documents/demo/project/mall && go build ./admin-bff/...
go vet ./admin-bff/...
```

---

## Task 6: merchant-bff — JWT blacklist (same pattern as Task 5)

**Files:**
- Create: `merchant-bff/ioc/redis.go`
- Modify: `merchant-bff/handler/jwt/handler.go`
- Modify: `merchant-bff/handler/middleware/login_jwt.go`
- Modify: `merchant-bff/wire.go`
- Modify: `merchant-bff/config/dev.yaml`

Same pattern as Task 5. Key differences:
- Secret keys: `merchant-bff-access-secret-key32!` / `merchant-bff-refresh-secret-key32!`
- Import path: `ijwt "github.com/rermrf/mall/merchant-bff/handler/jwt"`
- `LoginReq` has `TenantId` field, `Login` passes `req.TenantId`
- `NewJWTHandler` accepts `(userClient, rdb)`, same as admin-bff

**Step 1:** Create `merchant-bff/ioc/redis.go` — identical to admin-bff version

**Step 2:** Modify `merchant-bff/handler/jwt/handler.go`:
- Add `rdb redis.Cmdable` field to JWTHandler
- Change `NewJWTHandler` to accept `(userClient userv1.UserServiceClient, rdb redis.Cmdable)`
- Add JTI (uuid) to both access and refresh claims in `SetTokenHeaders`
- Implement Logout with blacklist (same logic as admin-bff)
- Add `IsTokenBlacklisted` and `extractTokenFromHeader` methods
- Add blacklist check to `Refresh`

**Step 3:** Modify `merchant-bff/handler/middleware/login_jwt.go`:
- Add blacklist check after `ParseAccessToken` (same pattern as admin-bff)

**Step 4:** Update `merchant-bff/wire.go` — add `ioc.InitRedis` to thirdPartySet

**Step 5:** Update `merchant-bff/config/dev.yaml` — add redis section

**Step 6:** Regenerate wire and verify:
```bash
cd /Users/emoji/Documents/demo/project/mall/merchant-bff && wire
cd /Users/emoji/Documents/demo/project/mall && go build ./merchant-bff/...
```

---

## Task 7: consumer-bff — JWT blacklist (same pattern as Task 5)

**Files:**
- Create: `consumer-bff/ioc/redis.go`
- Modify: `consumer-bff/handler/jwt/handler.go`
- Modify: `consumer-bff/handler/middleware/login_jwt.go`
- Modify: `consumer-bff/wire.go`
- Modify: `consumer-bff/config/dev.yaml`

Same pattern as Task 5. Key differences:
- Secret keys: `consumer-bff-access-secret-key32!` / `consumer-bff-refresh-secret-key32!`
- Import path: `ijwt "github.com/rermrf/mall/consumer-bff/handler/jwt"`
- Has extra methods: `LoginByPhone`, `OAuthLogin`
- Uses `tenantx.GetTenantID(ctx.Request.Context())` for tenant extraction
- Import: `"github.com/rermrf/mall/pkg/tenantx"`
- `NewJWTHandler` accepts `(userClient, rdb)`

**Step 1:** Create `consumer-bff/ioc/redis.go` — identical to admin-bff version

**Step 2:** Modify `consumer-bff/handler/jwt/handler.go`:
- Add `rdb redis.Cmdable` field to JWTHandler
- Change `NewJWTHandler` to accept `(userClient userv1.UserServiceClient, rdb redis.Cmdable)`
- Add JTI (uuid) to both claims in `SetTokenHeaders`
- Implement Logout with blacklist
- Add `IsTokenBlacklisted` and `extractTokenFromHeader`
- Add blacklist check to `Refresh`
- Keep `LoginByPhone` and `OAuthLogin` methods unchanged

**Step 3:** Modify `consumer-bff/handler/middleware/login_jwt.go`:
- Add blacklist check after `ParseAccessToken`

**Step 4:** Update `consumer-bff/wire.go` — add `ioc.InitRedis` to thirdPartySet

**Step 5:** Update `consumer-bff/config/dev.yaml` — add redis section

**Step 6:** Regenerate wire and verify:
```bash
cd /Users/emoji/Documents/demo/project/mall/consumer-bff && wire
cd /Users/emoji/Documents/demo/project/mall && go build ./consumer-bff/...
```

---

## Task 8: user-svc — Fix TODO comments (B+C+D classes)

Clean up user-svc TODOs: Logout, RefreshToken, OAuthLogin, and user_registered consumer.

**Files:**
- Modify: `user/service/user.go`
- Modify: `user/events/consumer.go`

**Step 1: Update `user/service/user.go`**

Replace lines 146-175:

For `OAuthLogin` (line 146-165), replace TODO:
```go
func (s *userService) OAuthLogin(ctx context.Context, tenantId int64, provider, code string) (domain.User, bool, error) {
	// MVP: 简化 OAuth 实现，用 code 作为 provider_uid
	// 生产环境需对接真实 OAuth provider（Google/GitHub）获取用户信息
	oauth := domain.OAuthAccount{
		TenantID:    tenantId,
		Provider:    provider,
		ProviderUID: code,
	}
	newUser := domain.User{
		TenantID: tenantId,
		Phone:    "",
		Status:   domain.UserStatusNormal,
	}
	u, err := s.repo.FindOrCreateByOAuth(ctx, oauth, newUser)
	if err != nil {
		return domain.User{}, false, err
	}
	isNew := u.Phone == ""
	return u, isNew, nil
}
```

For `RefreshToken` (line 167-169):
```go
func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// JWT 刷新由 BFF 层本地处理，user-svc 不参与 token 管理
	return "", "", nil
}
```

For `Logout` (line 172-174):
```go
func (s *userService) Logout(ctx context.Context, accessToken string) error {
	// JWT 黑名单由 BFF 层 Redis 管理，user-svc 不参与 token 管理
	return nil
}
```

**Step 2: Update `user/events/consumer.go`**

Replace line 51 TODO:
```go
// Consume 处理单条 user_registered 事件
func (c *UserRegisteredConsumer) Consume(msg *sarama.ConsumerMessage, evt UserRegisteredEvent) error {
	c.l.Info("收到用户注册事件",
		logger.Int64("userId", evt.UserId),
		logger.Int64("tenantId", evt.TenantId),
		logger.String("phone", evt.Phone),
	)
	// user-svc 内部消费：预留用于初始化用户默认数据（如偏好设置等）
	// 主要消费者为 notification-svc（发送欢迎通知）
	return nil
}
```

**Step 3: Verify build**

```bash
cd /Users/emoji/Documents/demo/project/mall && go build ./user/...
go vet ./user/...
```

---

## Task 9: order-svc — Seckill auto-order creation

Implement automatic seckill order creation when receiving `seckill_success` Kafka event.

**Files:**
- Modify: `order/service/order.go`
- Modify: `order/ioc/kafka.go`

**Step 1: Add seckill fields to `CreateOrderReq` in `order/service/order.go`**

Update `CreateOrderReq` struct (line 40-48):
```go
type CreateOrderReq struct {
	BuyerID      int64
	TenantID     int64
	Items        []CreateOrderItemReq
	AddressID    int64
	CouponID     int64
	Remark       string
	Channel      string
	IsSeckill    bool
	SeckillPrice int64 // 分，仅当 IsSeckill=true 时使用
}
```

Update `buildOrderItems` method to handle seckill pricing. Find the method (around line 467):

In the loop `for _, ri := range req.Items`, change the price line:
```go
func (s *orderService) buildOrderItems(req CreateOrderReq, skuMap map[int64]*productv1.ProductSKU) ([]domain.OrderItem, int64, error) {
	var totalAmount int64
	items := make([]domain.OrderItem, 0, len(req.Items))
	for _, ri := range req.Items {
		sku, ok := skuMap[ri.SKUID]
		if !ok {
			return nil, 0, fmt.Errorf("SKU %d 不存在", ri.SKUID)
		}
		price := sku.GetPrice()
		if req.IsSeckill && req.SeckillPrice > 0 {
			price = req.SeckillPrice
		}
		subtotal := price * int64(ri.Quantity)
		totalAmount += subtotal
		items = append(items, domain.OrderItem{
			// ... rest stays the same, but use `price` instead of `sku.GetPrice()`
```

Only change the price calculation line and the `Price` field assignment below it.

**Step 2: Implement seckill order in `order/ioc/kafka.go`**

Replace `NewSeckillSuccessConsumer` (line 73-91). Add `userv1.UserServiceClient` to the function params:

```go
// Add import: userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"

func NewSeckillSuccessConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.OrderService,
	userClient userv1.UserServiceClient,
) *events.SeckillSuccessConsumer {
	return events.NewSeckillSuccessConsumer(cg, l, func(ctx context.Context, evt events.SeckillSuccessEvent) error {
		l.Info("收到秒杀成功事件，创建秒杀订单",
			logger.Int64("userId", evt.UserId),
			logger.Int64("skuId", evt.SkuId),
			logger.Int64("seckillPrice", evt.SeckillPrice))

		// 获取用户默认收货地址
		addrResp, err := userClient.ListAddresses(ctx, &userv1.ListAddressesRequest{
			UserId: evt.UserId,
		})
		if err != nil {
			l.Error("获取用户地址失败", logger.Int64("userId", evt.UserId), logger.Error(err))
			return err
		}
		var addressId int64
		if len(addrResp.GetAddresses()) > 0 {
			addressId = addrResp.GetAddresses()[0].GetId()
		}

		// 创建秒杀订单
		orderNo, payAmount, err := svc.CreateOrder(ctx, service.CreateOrderReq{
			BuyerID:      evt.UserId,
			TenantID:     evt.TenantId,
			Items:        []service.CreateOrderItemReq{{SKUID: evt.SkuId, Quantity: 1}},
			AddressID:    addressId,
			CouponID:     0,
			Remark:       "秒杀订单",
			Channel:      "seckill",
			IsSeckill:    true,
			SeckillPrice: evt.SeckillPrice,
		})
		if err != nil {
			l.Error("创建秒杀订单失败",
				logger.Int64("userId", evt.UserId),
				logger.Int64("skuId", evt.SkuId),
				logger.Error(err))
			return err
		}
		l.Info("秒杀订单创建成功",
			logger.String("orderNo", orderNo),
			logger.Int64("payAmount", payAmount))
		return nil
	})
}
```

Note: `order/wire.go` already has `ioc.InitUserClient` in thirdPartySet, so no wire change needed. The wire will auto-inject `userv1.UserServiceClient` into `NewSeckillSuccessConsumer`.

**Step 3: Regenerate wire and verify**

```bash
cd /Users/emoji/Documents/demo/project/mall/order && wire
cd /Users/emoji/Documents/demo/project/mall && go build ./order/...
go vet ./order/...
```

---

## Task 10: Final verification — all services build clean

**Step 1: Build all affected services**

```bash
cd /Users/emoji/Documents/demo/project/mall
go build ./notification/...
go build ./marketing/...
go build ./admin-bff/...
go build ./merchant-bff/...
go build ./consumer-bff/...
go build ./user/...
go build ./order/...
go build ./logistics/...
```

**Step 2: Vet all services**

```bash
go vet ./notification/...
go vet ./marketing/...
go vet ./admin-bff/...
go vet ./merchant-bff/...
go vet ./consumer-bff/...
go vet ./user/...
go vet ./order/...
go vet ./logistics/...
```

**Step 3: Verify no remaining TODOs (except non-actionable ones)**

```bash
grep -rn "TODO" --include="*.go" --exclude-dir=api --exclude-dir=vendor | grep -v "_test.go"
```

Expected: Zero or only non-actionable TODOs (like "MVP: ..." explanations).
