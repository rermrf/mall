# BFF Layer Implementation Plan (user/tenant)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement 3 BFF HTTP gateways (admin-bff, merchant-bff, consumer-bff) with user/tenant service endpoints, JWT auth, and per-BFF middleware.

**Architecture:** Each BFF is an independent Gin HTTP server that calls user-svc and tenant-svc via gRPC. JWT tokens are managed entirely within each BFF. Uses `pkg/ginx` for response wrapping and `pkg/tenantx` for context propagation.

**Tech Stack:** Gin, gRPC client, golang-jwt/jwt/v5, Wire DI, Viper config, etcd service discovery

---

## Task 1: Add JWT dependency

**Files:**
- Modify: `go.mod`

**Step 1:** Add JWT library

```bash
go get github.com/golang-jwt/jwt/v5
```

**Step 2:** Verify

```bash
go mod tidy
```

---

## Task 2: admin-bff infrastructure

Build the full admin-bff skeleton: error codes, gRPC clients, JWT handler, middleware, IoC, Wire, config, main. This establishes the pattern all other BFFs follow.

**Files:**
- Create: `admin-bff/errs/code.go`
- Create: `admin-bff/client/user.go`
- Create: `admin-bff/client/tenant.go`
- Create: `admin-bff/handler/jwt/handler.go`
- Create: `admin-bff/handler/middleware/login_jwt.go`
- Create: `admin-bff/handler/middleware/admin_only.go`
- Create: `admin-bff/ioc/grpc.go`
- Create: `admin-bff/ioc/gin.go`
- Create: `admin-bff/ioc/logger.go`
- Create: `admin-bff/config/dev.yaml`
- Create: `admin-bff/app.go`
- Create: `admin-bff/wire.go`
- Create: `admin-bff/main.go`

### errs/code.go

```go
package errs

// 业务错误码
const (
	CodeOK          = 0
	CodeInvalidParam = 4
	CodeServerError  = 5
	CodeUnauthorized = 401001
	CodeForbidden    = 403001
	CodeUserNotFound = 404001
)
```

### client/user.go

gRPC 客户端工厂，使用 etcd 服务发现连接 user-svc。

```go
package client

import (
	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"google.golang.org/grpc"
)

func NewUserClient(conn *grpc.ClientConn) userv1.UserServiceClient {
	return userv1.NewUserServiceClient(conn)
}
```

### client/tenant.go

```go
package client

import (
	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"google.golang.org/grpc"
)

func NewTenantClient(conn *grpc.ClientConn) tenantv1.TenantServiceClient {
	return tenantv1.NewTenantServiceClient(conn)
}
```

### handler/jwt/handler.go

JWTHandler 负责 JWT 签发/验证和 Login/Logout/Refresh 三个端点。

关键设计：
- Claims 结构：`{Uid int64, TenantId int64, UserAgent string, RegisteredClaims}`
- access_token: 30min, signed with `accessSecret`
- refresh_token: 7d, signed with `refreshSecret`
- 登录：调 user-svc `Login` RPC → 签发双 token → 通过响应头返回
- 刷新：验证 refresh_token → 签发新双 token
- 登出：暂不实现 Redis 黑名单，返回成功即可（TODO）

admin-bff 登录时 tenant_id 固定为 0（平台管理员）。

```go
package jwt

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rermrf/emo/logger"
	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type Claims struct {
	Uid       int64  `json:"uid"`
	TenantId  int64  `json:"tenant_id"`
	UserAgent string `json:"user_agent"`
	jwt.RegisteredClaims
}

type JWTHandler struct {
	userClient    userv1.UserServiceClient
	accessSecret  []byte
	refreshSecret []byte
	l             logger.Logger
}

func NewJWTHandler(userClient userv1.UserServiceClient, l logger.Logger) *JWTHandler {
	return &JWTHandler{
		userClient:    userClient,
		accessSecret:  []byte("admin-bff-access-secret-key-32b!"),
		refreshSecret: []byte("admin-bff-refresh-secret-key-32!"),
		l:             l,
	}
}

// GenerateTokens 生成 access_token 和 refresh_token
func (h *JWTHandler) GenerateTokens(uid, tenantId int64, userAgent string) (string, string, error) {
	now := time.Now()
	accessClaims := Claims{
		Uid:       uid,
		TenantId:  tenantId,
		UserAgent: userAgent,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(h.accessSecret)
	if err != nil {
		return "", "", err
	}

	refreshClaims := Claims{
		Uid:       uid,
		TenantId:  tenantId,
		UserAgent: userAgent,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(h.refreshSecret)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// ParseAccessToken 解析 access_token
func (h *JWTHandler) ParseAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		return h.accessSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// SetTokenHeaders 将双 token 写入响应头
func (h *JWTHandler) SetTokenHeaders(ctx *gin.Context, accessToken, refreshToken string) {
	ctx.Header("X-Jwt-Token", accessToken)
	ctx.Header("X-Refresh-Token", refreshToken)
}
```

Login/Logout/Refresh 三个路由方法需要在同一个文件中，作为 `JWTHandler` 的方法。admin-bff 的 Login 调用 `user-svc Login` RPC 并传 tenant_id=0：

```go
// Login 管理员密码登录
func (h *JWTHandler) Login(ctx *gin.Context, req LoginReq) (ginx.Result, error) {
	resp, err := h.userClient.Login(ctx, &userv1.LoginRequest{
		TenantId: 0, // admin: platform tenant
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "登录失败"}, err
	}
	accessToken, refreshToken, err := h.GenerateTokens(
		resp.User.Id, 0, ctx.GetHeader("User-Agent"),
	)
	if err != nil {
		return ginx.Result{Code: 5, Msg: "生成 token 失败"}, err
	}
	h.SetTokenHeaders(ctx, accessToken, refreshToken)
	return ginx.Result{Code: 0, Msg: "登录成功"}, nil
}

// Refresh 刷新 token
func (h *JWTHandler) Refresh(ctx *gin.Context, _ RefreshReq) (ginx.Result, error) {
	tokenStr := ctx.GetHeader("X-Refresh-Token")
	if tokenStr == "" {
		return ginx.Result{Code: 401001, Msg: "缺少 refresh token"}, nil
	}
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		return h.refreshSecret, nil
	})
	if err != nil || !token.Valid {
		return ginx.Result{Code: 401001, Msg: "refresh token 无效"}, nil
	}
	claims := token.Claims.(*Claims)
	accessToken, refreshToken, err := h.GenerateTokens(
		claims.Uid, claims.TenantId, claims.UserAgent,
	)
	if err != nil {
		return ginx.Result{Code: 5, Msg: "生成 token 失败"}, err
	}
	h.SetTokenHeaders(ctx, accessToken, refreshToken)
	return ginx.Result{Code: 0, Msg: "刷新成功"}, nil
}

// Logout 登出
func (h *JWTHandler) Logout(ctx *gin.Context, _ LogoutReq) (ginx.Result, error) {
	// TODO: 加入 Redis 黑名单
	return ginx.Result{Code: 0, Msg: "登出成功"}, nil
}

// Request structs
type LoginReq struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type RefreshReq struct{}
type LogoutReq struct{}
```

### handler/middleware/login_jwt.go

LoginJWT 中间件：从 `Authorization: Bearer <token>` 提取并验证 access_token，将 Claims 注入 gin.Context。

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

func NewLoginJWT(jwtHandler *ijwt.JWTHandler) *LoginJWTBuilder {
	return &LoginJWTBuilder{jwtHandler: jwtHandler}
}

func (b *LoginJWTBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized,
				ginx.Result{Code: 401001, Msg: "未登录"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized,
				ginx.Result{Code: 401001, Msg: "token 格式错误"})
			return
		}
		claims, err := b.jwtHandler.ParseAccessToken(parts[1])
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized,
				ginx.Result{Code: 401001, Msg: "token 无效或已过期"})
			return
		}
		// Inject claims into context
		ctx.Set("claims", claims)
		ctx.Set("uid", claims.Uid)
		ctx.Set("tenant_id", claims.TenantId)
		ctx.Next()
	}
}
```

### handler/middleware/admin_only.go

AdminOnly 中间件：仅允许 tenant_id == 0 的平台管理员通过。

```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rermrf/mall/pkg/ginx"
)

func AdminOnly() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tenantId, exists := ctx.Get("tenant_id")
		if !exists || tenantId.(int64) != 0 {
			ctx.AbortWithStatusJSON(http.StatusForbidden,
				ginx.Result{Code: 403001, Msg: "仅平台管理员可访问"})
			return
		}
		ctx.Next()
	}
}
```

### ioc/grpc.go

gRPC 连接初始化，使用 etcd 服务发现。连接 user-svc 和 tenant-svc。

```go
package ioc

import (
	"fmt"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	resolver "go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitEtcdClient() *clientv3.Client {
	type Config struct {
		Addrs []string `yaml:"addrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("etcd", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 etcd 配置失败: %w", err))
	}
	client, err := clientv3.NewFromURLs(cfg.Addrs)
	if err != nil {
		panic(fmt.Errorf("连接 etcd 失败: %w", err))
	}
	return client
}

func InitUserConn(etcdClient *clientv3.Client) *grpc.ClientConn {
	return initServiceConn(etcdClient, "user")
}

func InitTenantConn(etcdClient *clientv3.Client) *grpc.ClientConn {
	return initServiceConn(etcdClient, "tenant")
}

func initServiceConn(etcdClient *clientv3.Client, serviceName string) *grpc.ClientConn {
	bd, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(fmt.Errorf("创建 etcd resolver 失败: %w", err))
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("etcd:///%s", serviceName),
		grpc.WithResolvers(bd),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Errorf("连接 %s 服务失败: %w", serviceName, err))
	}
	return conn
}
```

需要在 ioc/grpc.go 中声明两个不同的 `*grpc.ClientConn` 给 Wire。由于 Wire 不能区分同一类型的两个 provider，需要用命名类型：

```go
// 为 Wire 区分不同的 gRPC 连接
type UserConn grpc.ClientConn
type TenantConn grpc.ClientConn
```

或者更简洁地，直接让 client 工厂接受 `*clientv3.Client` 并内部创建连接。但考虑已有 `client/user.go` 接受 `*grpc.ClientConn`，最佳方案是 ioc 分别提供两个具名初始化函数，并在 wire 中用 `wire.FieldsOf` 或 `wire.Bind`。

**实际方案**：ioc/grpc.go 中 `InitUserClient` 和 `InitTenantClient` 直接返回 gRPC client 接口（不暴露 conn）：

```go
package ioc

import (
	"fmt"

	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	resolver "go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitEtcdClient() *clientv3.Client {
	type Config struct {
		Addrs []string `yaml:"addrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("etcd", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 etcd 配置失败: %w", err))
	}
	client, err := clientv3.NewFromURLs(cfg.Addrs)
	if err != nil {
		panic(fmt.Errorf("连接 etcd 失败: %w", err))
	}
	return client
}

func InitUserClient(etcdClient *clientv3.Client) userv1.UserServiceClient {
	conn := initServiceConn(etcdClient, "user")
	return userv1.NewUserServiceClient(conn)
}

func InitTenantClient(etcdClient *clientv3.Client) tenantv1.TenantServiceClient {
	conn := initServiceConn(etcdClient, "tenant")
	return tenantv1.NewTenantServiceClient(conn)
}

func initServiceConn(etcdClient *clientv3.Client, serviceName string) *grpc.ClientConn {
	bd, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(fmt.Errorf("创建 etcd resolver 失败: %w", err))
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("etcd:///%s", serviceName),
		grpc.WithResolvers(bd),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Errorf("连接 %s 服务失败: %w", serviceName, err))
	}
	return conn
}
```

这样 `client/` 目录就不需要了（client 工厂直接在 ioc 中），简化 Wire。但设计文档要求有 `client/` 目录。折中方案：**保留 `client/` 目录作为包装**，或者 **取消 `client/` 目录，直接在 ioc/grpc.go 返回 gRPC client**。后者更简洁。

**最终决策**：不使用 `client/` 目录，ioc/grpc.go 直接返回 gRPC client 接口。Wire 可以直接注入。

### ioc/gin.go

Gin 引擎 + 路由注册。此文件实现 `InitGinServer`，注册所有中间件和路由。

```go
package ioc

import (
	"github.com/gin-gonic/gin"
	"github.com/rermrf/emo/logger"
	ijwt "github.com/rermrf/mall/admin-bff/handler/jwt"
	"github.com/rermrf/mall/admin-bff/handler/middleware"
	"github.com/rermrf/mall/admin-bff/handler"
	"github.com/rermrf/mall/pkg/ginx"
)

func InitGinServer(
	jwtHandler *ijwt.JWTHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	l logger.Logger,
) *gin.Engine {
	engine := gin.Default()
	engine.Use(ginx.DefaultCORS())

	// 公开路由（不需要登录）
	pub := engine.Group("/api/v1")
	pub.POST("/login", ginx.WrapBody(l, jwtHandler.Login))

	// 需要登录 + 管理员权限的路由
	auth := engine.Group("/api/v1")
	auth.Use(middleware.NewLoginJWT(jwtHandler).Build())
	auth.Use(middleware.AdminOnly())
	{
		auth.POST("/logout", ginx.WrapBody(l, jwtHandler.Logout))
		auth.POST("/refresh-token", ginx.WrapBody(l, jwtHandler.Refresh))

		// 用户管理
		auth.GET("/users", ginx.WrapQuery(l, userHandler.ListUsers))
		auth.POST("/users/:id/status", ginx.WrapBody(l, userHandler.UpdateUserStatus))
		auth.GET("/roles", ginx.WrapQuery(l, userHandler.ListRoles))
		auth.POST("/roles", ginx.WrapBody(l, userHandler.CreateRole))
		auth.PUT("/roles/:id", ginx.WrapBody(l, userHandler.UpdateRole))

		// 租户管理
		auth.POST("/tenants", ginx.WrapBody(l, tenantHandler.CreateTenant))
		auth.GET("/tenants", ginx.WrapQuery(l, tenantHandler.ListTenants))
		auth.GET("/tenants/:id", tenantHandler.GetTenant) // 路径参数，不用 WrapQuery
		auth.POST("/tenants/:id/approve", ginx.WrapBody(l, tenantHandler.ApproveTenant))
		auth.POST("/tenants/:id/freeze", ginx.WrapBody(l, tenantHandler.FreezeTenant))
		auth.GET("/plans", ginx.WrapQuery(l, tenantHandler.ListPlans))
		auth.POST("/plans", ginx.WrapBody(l, tenantHandler.CreatePlan))
		auth.PUT("/plans/:id", ginx.WrapBody(l, tenantHandler.UpdatePlan))
	}
	return engine
}
```

注意：带路径参数的端点（如 `GET /tenants/:id`）不能用 `WrapQuery`，需要用普通的 `gin.HandlerFunc`，handler 内部从 `ctx.Param("id")` 读取。

### ioc/logger.go

```go
package ioc

import (
	"github.com/rermrf/emo/logger"
	"go.uber.org/zap"
)

func InitLogger() logger.Logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger.NewZapLogger(l)
}
```

### config/dev.yaml

```yaml
server:
  addr: ":8280"

etcd:
  addrs:
    - "rermrf.icu:2379"
```

### app.go

```go
package main

import "github.com/gin-gonic/gin"

type App struct {
	Server *gin.Engine
	Addr   string
}
```

### wire.go

```go
//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/rermrf/mall/admin-bff/handler"
	ijwt "github.com/rermrf/mall/admin-bff/handler/jwt"
	"github.com/rermrf/mall/admin-bff/ioc"
)

var thirdPartySet = wire.NewSet(
	ioc.InitEtcdClient,
	ioc.InitLogger,
	ioc.InitUserClient,
	ioc.InitTenantClient,
)

var handlerSet = wire.NewSet(
	ijwt.NewJWTHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	ioc.InitGinServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, handlerSet, wire.Struct(new(App), "*"))
	return new(App)
}
```

注意 Wire 需要将 `Addr` 注入 App。需要 ioc 提供一个 `InitAddr` 函数或直接在 main 中读取。简化方案：App 只包含 `*gin.Engine`，addr 在 main 中直接读取。

### main.go

```go
package main

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()
	app := InitApp()
	addr := viper.GetString("server.addr")
	fmt.Printf("admin-bff 启动于 %s\n", addr)
	if err := app.Server.Run(addr); err != nil {
		panic(fmt.Errorf("启动服务失败: %w", err))
	}
}

func initViper() {
	configFile := pflag.String("config", "config/dev.yaml", "配置文件路径")
	pflag.Parse()
	viper.SetConfigFile(*configFile)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}
}
```

**验证：** `go build ./admin-bff/...` 编译通过。Wire 生成需要先创建 handler 文件（Task 3）。

---

## Task 3: admin-bff handlers (UserHandler + TenantHandler)

**Files:**
- Create: `admin-bff/handler/user.go`
- Create: `admin-bff/handler/tenant.go`

### handler/user.go

admin-bff 的 UserHandler 负责用户管理相关端点。依赖 `userv1.UserServiceClient`。

端点：
- `GET /users` → ListUsers（分页+搜索）
- `POST /users/:id/status` → UpdateUserStatus（冻结/解冻）
- `GET /roles` → ListRoles
- `POST /roles` → CreateRole
- `PUT /roles/:id` → UpdateRole

每个方法对应一个 handler 函数，签名匹配 `ginx.WrapBody` 或 `ginx.WrapQuery` 的泛型约束。

```go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rermrf/emo/logger"
	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type UserHandler struct {
	userClient userv1.UserServiceClient
	l          logger.Logger
}

func NewUserHandler(userClient userv1.UserServiceClient, l logger.Logger) *UserHandler {
	return &UserHandler{userClient: userClient, l: l}
}

// ==================== Request Types ====================

type ListUsersReq struct {
	Page     int32  `form:"page"`
	PageSize int32  `form:"page_size"`
	Status   int32  `form:"status"`
	Keyword  string `form:"keyword"`
}

type UpdateUserStatusReq struct {
	Status int32 `json:"status" binding:"required"` // 1-正常 2-冻结
}

type CreateRoleReq struct {
	TenantId    int64  `json:"tenant_id"`
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

type UpdateRoleReq struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

type ListRolesReq struct {
	TenantId int64 `form:"tenant_id"`
}

// ==================== Handlers ====================

func (h *UserHandler) ListUsers(ctx *gin.Context, req ListUsersReq) (ginx.Result, error) {
	resp, err := h.userClient.ListUsers(ctx, &userv1.ListUsersRequest{
		TenantId: 0, // 平台级，查看所有
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   req.Status,
		Keyword:  req.Keyword,
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "获取用户列表失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok", Data: resp}, nil
}

func (h *UserHandler) UpdateUserStatus(ctx *gin.Context, req UpdateUserStatusReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的用户 ID"}, nil
	}
	_, err = h.userClient.UpdateUserStatus(ctx, &userv1.UpdateUserStatusRequest{
		Id:     id,
		Status: req.Status,
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "更新用户状态失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok"}, nil
}

func (h *UserHandler) ListRoles(ctx *gin.Context, req ListRolesReq) (ginx.Result, error) {
	resp, err := h.userClient.ListRoles(ctx, &userv1.ListRolesRequest{
		TenantId: req.TenantId,
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "获取角色列表失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok", Data: resp}, nil
}

func (h *UserHandler) CreateRole(ctx *gin.Context, req CreateRoleReq) (ginx.Result, error) {
	resp, err := h.userClient.CreateRole(ctx, &userv1.CreateRoleRequest{
		Role: &userv1.Role{
			TenantId:    req.TenantId,
			Name:        req.Name,
			Code:        req.Code,
			Description: req.Description,
		},
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "创建角色失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok", Data: map[string]any{"id": resp.Id}}, nil
}

func (h *UserHandler) UpdateRole(ctx *gin.Context, req UpdateRoleReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的角色 ID"}, nil
	}
	_, err = h.userClient.UpdateRole(ctx, &userv1.UpdateRoleRequest{
		Role: &userv1.Role{
			Id:          id,
			Name:        req.Name,
			Code:        req.Code,
			Description: req.Description,
		},
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "更新角色失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok"}, nil
}
```

### handler/tenant.go

admin-bff 的 TenantHandler 负责商家管理和套餐管理端点。

端点：
- `POST /tenants` → CreateTenant
- `GET /tenants` → ListTenants
- `GET /tenants/:id` → GetTenant（路径参数，普通 HandlerFunc）
- `POST /tenants/:id/approve` → ApproveTenant
- `POST /tenants/:id/freeze` → FreezeTenant
- `GET /plans` → ListPlans
- `POST /plans` → CreatePlan
- `PUT /plans/:id` → UpdatePlan

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rermrf/emo/logger"
	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type TenantHandler struct {
	tenantClient tenantv1.TenantServiceClient
	l            logger.Logger
}

func NewTenantHandler(tenantClient tenantv1.TenantServiceClient, l logger.Logger) *TenantHandler {
	return &TenantHandler{tenantClient: tenantClient, l: l}
}

// ==================== Request Types ====================

type CreateTenantReq struct {
	Name            string `json:"name" binding:"required"`
	ContactName     string `json:"contact_name" binding:"required"`
	ContactPhone    string `json:"contact_phone" binding:"required"`
	BusinessLicense string `json:"business_license"`
	PlanId          int64  `json:"plan_id"`
}

type ListTenantsReq struct {
	Page     int32 `form:"page"`
	PageSize int32 `form:"page_size"`
	Status   int32 `form:"status"`
}

type ApproveTenantReq struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

type FreezeTenantReq struct {
	Freeze bool `json:"freeze"`
}

type CreatePlanReq struct {
	Name         string `json:"name" binding:"required"`
	Price        int64  `json:"price"`
	DurationDays int32  `json:"duration_days"`
	MaxProducts  int32  `json:"max_products"`
	MaxStaff     int32  `json:"max_staff"`
	Features     string `json:"features"`
}

type UpdatePlanReq struct {
	Name         string `json:"name"`
	Price        int64  `json:"price"`
	DurationDays int32  `json:"duration_days"`
	MaxProducts  int32  `json:"max_products"`
	MaxStaff     int32  `json:"max_staff"`
	Features     string `json:"features"`
}

type ListPlansReq struct{}

// ==================== Handlers ====================

func (h *TenantHandler) CreateTenant(ctx *gin.Context, req CreateTenantReq) (ginx.Result, error) {
	resp, err := h.tenantClient.CreateTenant(ctx, &tenantv1.CreateTenantRequest{
		Tenant: &tenantv1.Tenant{
			Name:            req.Name,
			ContactName:     req.ContactName,
			ContactPhone:    req.ContactPhone,
			BusinessLicense: req.BusinessLicense,
			PlanId:          req.PlanId,
		},
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "创建商家失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok", Data: map[string]any{"id": resp.Id}}, nil
}

func (h *TenantHandler) ListTenants(ctx *gin.Context, req ListTenantsReq) (ginx.Result, error) {
	resp, err := h.tenantClient.ListTenants(ctx, &tenantv1.ListTenantsRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   req.Status,
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "获取商家列表失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok", Data: resp}, nil
}

// GetTenant 使用路径参数，直接是 gin.HandlerFunc
func (h *TenantHandler) GetTenant(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ginx.Result{Code: 4, Msg: "无效的商家 ID"})
		return
	}
	resp, err := h.tenantClient.GetTenant(ctx, &tenantv1.GetTenantRequest{Id: id})
	if err != nil {
		h.l.Error("获取商家详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "获取商家详情失败"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "ok", Data: resp.Tenant})
}

func (h *TenantHandler) ApproveTenant(ctx *gin.Context, req ApproveTenantReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的商家 ID"}, nil
	}
	_, err = h.tenantClient.ApproveTenant(ctx, &tenantv1.ApproveTenantRequest{
		Id:       id,
		Approved: req.Approved,
		Reason:   req.Reason,
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "审核失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok"}, nil
}

func (h *TenantHandler) FreezeTenant(ctx *gin.Context, req FreezeTenantReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的商家 ID"}, nil
	}
	_, err = h.tenantClient.FreezeTenant(ctx, &tenantv1.FreezeTenantRequest{
		Id:     id,
		Freeze: req.Freeze,
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "操作失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok"}, nil
}

func (h *TenantHandler) ListPlans(ctx *gin.Context, _ ListPlansReq) (ginx.Result, error) {
	resp, err := h.tenantClient.ListPlans(ctx, &tenantv1.ListPlansRequest{})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "获取套餐列表失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok", Data: resp.Plans}, nil
}

func (h *TenantHandler) CreatePlan(ctx *gin.Context, req CreatePlanReq) (ginx.Result, error) {
	resp, err := h.tenantClient.CreatePlan(ctx, &tenantv1.CreatePlanRequest{
		Plan: &tenantv1.TenantPlan{
			Name:         req.Name,
			Price:        req.Price,
			DurationDays: req.DurationDays,
			MaxProducts:  req.MaxProducts,
			MaxStaff:     req.MaxStaff,
			Features:     req.Features,
		},
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "创建套餐失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok", Data: map[string]any{"id": resp.Id}}, nil
}

func (h *TenantHandler) UpdatePlan(ctx *gin.Context, req UpdatePlanReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的套餐 ID"}, nil
	}
	_, err = h.tenantClient.UpdatePlan(ctx, &tenantv1.UpdatePlanRequest{
		Plan: &tenantv1.TenantPlan{
			Id:           id,
			Name:         req.Name,
			Price:        req.Price,
			DurationDays: req.DurationDays,
			MaxProducts:  req.MaxProducts,
			MaxStaff:     req.MaxStaff,
			Features:     req.Features,
		},
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "更新套餐失败"}, err
	}
	return ginx.Result{Code: 0, Msg: "ok"}, nil
}
```

**验证：** Run Wire + `go build ./admin-bff/...`

---

## Task 4: merchant-bff 完整实现

merchant-bff 与 admin-bff 结构相同，差异：
- 端口 8180
- JWT claims 中 `tenant_id > 0`（商家员工）
- 登录时需传 `tenant_id`
- 中间件：`TenantExtract` 从 JWT claims 提取 tenant_id 注入 context（替代 AdminOnly）
- Handler 端点不同：员工管理、店铺、配额

**Files:**
- Create: `merchant-bff/errs/code.go` — 同 admin-bff
- Create: `merchant-bff/handler/jwt/handler.go` — Login 需接收 tenant_id 参数
- Create: `merchant-bff/handler/middleware/login_jwt.go` — 同 admin-bff，但 import 路径不同
- Create: `merchant-bff/handler/middleware/tenant_extract.go` — 从 JWT claims 提取 tenant_id 注入 `pkg/tenantx` context
- Create: `merchant-bff/handler/user.go` — 员工管理端点
- Create: `merchant-bff/handler/tenant.go` — 店铺/配额端点
- Create: `merchant-bff/ioc/grpc.go` — 同 admin-bff
- Create: `merchant-bff/ioc/gin.go` — merchant-bff 路由
- Create: `merchant-bff/ioc/logger.go` — 同 admin-bff
- Create: `merchant-bff/config/dev.yaml` — 端口 8180
- Create: `merchant-bff/app.go`
- Create: `merchant-bff/wire.go`
- Create: `merchant-bff/main.go`

### 关键差异点

**handler/jwt/handler.go — Login 接收 tenant_id：**

```go
type LoginReq struct {
	TenantId int64  `json:"tenant_id" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *JWTHandler) Login(ctx *gin.Context, req LoginReq) (ginx.Result, error) {
	resp, err := h.userClient.Login(ctx, &userv1.LoginRequest{
		TenantId: req.TenantId,
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		return ginx.Result{Code: 5, Msg: "登录失败"}, err
	}
	accessToken, refreshToken, err := h.GenerateTokens(
		resp.User.Id, req.TenantId, ctx.GetHeader("User-Agent"),
	)
	// ... 同 admin-bff
}
```

**handler/middleware/tenant_extract.go：**

```go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rermrf/mall/pkg/tenantx"
)

func TenantExtract() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tenantId, exists := ctx.Get("tenant_id")
		if !exists {
			ctx.Next()
			return
		}
		tid, ok := tenantId.(int64)
		if !ok || tid <= 0 {
			ctx.AbortWithStatusJSON(403, gin.H{"code": 403001, "msg": "需要商家身份"})
			return
		}
		// Inject into request context for gRPC propagation
		c := tenantx.WithTenantID(ctx.Request.Context(), tid)
		ctx.Request = ctx.Request.WithContext(c)
		ctx.Next()
	}
}
```

**handler/user.go — 商家员工端点：**
- `GET /profile` — FindById（从 JWT claims 取 uid）
- `PUT /profile` — UpdateProfile
- `GET /staff` — ListUsers（当前 tenant）
- `GET /roles` — ListRoles（当前 tenant）
- `POST /roles` — CreateRole
- `PUT /roles/:id` — UpdateRole
- `POST /staff/:id/role` — AssignRole

**handler/tenant.go — 店铺/配额端点：**
- `GET /shop` — GetShop（从 context 取 tenant_id）
- `PUT /shop` — UpdateShop
- `GET /quotas/:type` — CheckQuota

**ioc/gin.go 路由：**

```go
pub := engine.Group("/api/v1")
pub.POST("/login", ginx.WrapBody(l, jwtHandler.Login))

auth := engine.Group("/api/v1")
auth.Use(middleware.NewLoginJWT(jwtHandler).Build())
auth.Use(middleware.TenantExtract())
{
	auth.POST("/logout", ginx.WrapBody(l, jwtHandler.Logout))
	auth.POST("/refresh-token", ginx.WrapBody(l, jwtHandler.Refresh))

	auth.GET("/profile", userHandler.GetProfile)
	auth.PUT("/profile", ginx.WrapBody(l, userHandler.UpdateProfile))
	auth.GET("/staff", ginx.WrapQuery(l, userHandler.ListStaff))
	auth.GET("/roles", ginx.WrapQuery(l, userHandler.ListRoles))
	auth.POST("/roles", ginx.WrapBody(l, userHandler.CreateRole))
	auth.PUT("/roles/:id", ginx.WrapBody(l, userHandler.UpdateRole))
	auth.POST("/staff/:id/role", ginx.WrapBody(l, userHandler.AssignRole))

	auth.GET("/shop", tenantHandler.GetShop)
	auth.PUT("/shop", ginx.WrapBody(l, tenantHandler.UpdateShop))
	auth.GET("/quotas/:type", tenantHandler.CheckQuota)
}
```

**config/dev.yaml：**

```yaml
server:
  addr: ":8180"

etcd:
  addrs:
    - "rermrf.icu:2379"
```

**验证：** Run Wire + `go build ./merchant-bff/...`

---

## Task 5: consumer-bff 完整实现

consumer-bff 差异最大：
- 端口 8080
- TenantResolve 中间件（域名 → tenant_id）
- 有公开路由和需登录路由两组
- 登录时 tenant_id 从 context 获取（TenantResolve 已注入）
- 支持注册、手机登录、OAuth 登录、地址管理

**Files:**
- Create: `consumer-bff/errs/code.go`
- Create: `consumer-bff/handler/jwt/handler.go` — Login 从 context 取 tenant_id
- Create: `consumer-bff/handler/middleware/login_jwt.go`
- Create: `consumer-bff/handler/middleware/tenant_resolve.go` — 域名解析中间件
- Create: `consumer-bff/handler/user.go` — 注册、资料、地址、验证码
- Create: `consumer-bff/handler/tenant.go` — 店铺信息
- Create: `consumer-bff/ioc/grpc.go`
- Create: `consumer-bff/ioc/gin.go`
- Create: `consumer-bff/ioc/logger.go`
- Create: `consumer-bff/config/dev.yaml`
- Create: `consumer-bff/app.go`
- Create: `consumer-bff/wire.go`
- Create: `consumer-bff/main.go`

### 关键差异点

**handler/middleware/tenant_resolve.go — 域名解析中间件：**

```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/pkg/ginx"
	"github.com/rermrf/mall/pkg/tenantx"
)

type TenantResolveBuilder struct {
	tenantClient tenantv1.TenantServiceClient
}

func NewTenantResolve(tenantClient tenantv1.TenantServiceClient) *TenantResolveBuilder {
	return &TenantResolveBuilder{tenantClient: tenantClient}
}

func (b *TenantResolveBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		host := ctx.Request.Host
		// Remove port if present
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		// dev mode: use X-Tenant-Domain header if Host is localhost
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
		resp, err := b.tenantClient.GetShopByDomain(ctx, &tenantv1.GetShopByDomainRequest{
			Domain: domain,
		})
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusNotFound,
				ginx.Result{Code: 404001, Msg: "店铺不存在"})
			return
		}
		// Inject tenant_id into context
		c := tenantx.WithTenantID(ctx.Request.Context(), resp.Shop.TenantId)
		ctx.Request = ctx.Request.WithContext(c)
		ctx.Set("tenant_id", resp.Shop.TenantId)
		ctx.Set("shop", resp.Shop)
		ctx.Next()
	}
}
```

**handler/jwt/handler.go — Login 从 context 取 tenant_id：**

```go
type LoginReq struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *JWTHandler) Login(ctx *gin.Context, req LoginReq) (ginx.Result, error) {
	tenantId := tenantx.GetTenantID(ctx.Request.Context())
	resp, err := h.userClient.Login(ctx, &userv1.LoginRequest{
		TenantId: tenantId,
		Phone:    req.Phone,
		Password: req.Password,
	})
	// ...sign tokens with tenantId...
}
```

同样需要 `LoginByPhone`（手机登录）和 `OAuthLogin`（第三方登录）端点。

**handler/user.go — consumer 端点：**
- `POST /signup` — Signup
- `POST /sms/send` — SendSmsCode
- `GET /profile` — FindById
- `PUT /profile` — UpdateProfile
- `GET /addresses` — ListAddresses
- `POST /addresses` — CreateAddress
- `PUT /addresses/:id` — UpdateAddress
- `DELETE /addresses/:id` — DeleteAddress

**handler/tenant.go：**
- `GET /shop` — 获取当前店铺信息（从 context 中的 shop 或调用 GetShop）

**ioc/gin.go 路由：**

```go
// TenantResolve 全局中间件（所有请求都需要域名解析）
engine.Use(ginx.DefaultCORS())
engine.Use(middleware.NewTenantResolve(tenantClient).Build())

// 公开路由
pub := engine.Group("/api/v1")
pub.POST("/signup", ginx.WrapBody(l, userHandler.Signup))
pub.POST("/login", ginx.WrapBody(l, jwtHandler.Login))
pub.POST("/sms/send", ginx.WrapBody(l, userHandler.SendSmsCode))
pub.POST("/login/phone", ginx.WrapBody(l, jwtHandler.LoginByPhone))
pub.POST("/login/oauth", ginx.WrapBody(l, jwtHandler.OAuthLogin))
pub.GET("/shop", tenantHandler.GetShop)

// 需要登录的路由
auth := engine.Group("/api/v1")
auth.Use(middleware.NewLoginJWT(jwtHandler).Build())
{
	auth.POST("/logout", ginx.WrapBody(l, jwtHandler.Logout))
	auth.POST("/refresh-token", ginx.WrapBody(l, jwtHandler.Refresh))
	auth.GET("/profile", userHandler.GetProfile)
	auth.PUT("/profile", ginx.WrapBody(l, userHandler.UpdateProfile))
	auth.GET("/addresses", userHandler.ListAddresses)
	auth.POST("/addresses", ginx.WrapBody(l, userHandler.CreateAddress))
	auth.PUT("/addresses/:id", ginx.WrapBody(l, userHandler.UpdateAddress))
	auth.DELETE("/addresses/:id", userHandler.DeleteAddress)
}
```

**config/dev.yaml：**

```yaml
server:
  addr: ":8080"

etcd:
  addrs:
    - "rermrf.icu:2379"
```

**验证：** Run Wire + `go build ./consumer-bff/...`

---

## Task 6: 最终验证

**Step 1:** 确保所有三个 BFF 都能编译通过：

```bash
go build ./admin-bff/...
go build ./merchant-bff/...
go build ./consumer-bff/...
```

**Step 2:** 确保 go vet 无问题：

```bash
go vet ./admin-bff/...
go vet ./merchant-bff/...
go vet ./consumer-bff/...
```

---

## 文件清单

| # | 文件路径 | 说明 |
|---|---------|------|
| **admin-bff** | | |
| 1 | `admin-bff/errs/code.go` | 业务错误码 |
| 2 | `admin-bff/handler/jwt/handler.go` | JWT 签发/验证 + Login/Logout/Refresh |
| 3 | `admin-bff/handler/middleware/login_jwt.go` | LoginJWT 中间件 |
| 4 | `admin-bff/handler/middleware/admin_only.go` | AdminOnly 中间件 |
| 5 | `admin-bff/handler/user.go` | 用户管理端点（5 个） |
| 6 | `admin-bff/handler/tenant.go` | 商家/套餐端点（8 个） |
| 7 | `admin-bff/ioc/grpc.go` | etcd + gRPC client 初始化 |
| 8 | `admin-bff/ioc/gin.go` | Gin 引擎 + 路由注册 |
| 9 | `admin-bff/ioc/logger.go` | Logger 初始化 |
| 10 | `admin-bff/config/dev.yaml` | 配置（端口 8280） |
| 11 | `admin-bff/app.go` | App 聚合 |
| 12 | `admin-bff/wire.go` | Wire DI |
| 13 | `admin-bff/main.go` | 入口 |
| **merchant-bff** | | |
| 14 | `merchant-bff/errs/code.go` | 业务错误码 |
| 15 | `merchant-bff/handler/jwt/handler.go` | JWT（Login 带 tenant_id） |
| 16 | `merchant-bff/handler/middleware/login_jwt.go` | LoginJWT 中间件 |
| 17 | `merchant-bff/handler/middleware/tenant_extract.go` | TenantExtract 中间件 |
| 18 | `merchant-bff/handler/user.go` | 员工管理端点（7 个） |
| 19 | `merchant-bff/handler/tenant.go` | 店铺/配额端点（3 个） |
| 20 | `merchant-bff/ioc/grpc.go` | gRPC client 初始化 |
| 21 | `merchant-bff/ioc/gin.go` | Gin 引擎 + 路由 |
| 22 | `merchant-bff/ioc/logger.go` | Logger |
| 23 | `merchant-bff/config/dev.yaml` | 配置（端口 8180） |
| 24 | `merchant-bff/app.go` | App |
| 25 | `merchant-bff/wire.go` | Wire DI |
| 26 | `merchant-bff/main.go` | 入口 |
| **consumer-bff** | | |
| 27 | `consumer-bff/errs/code.go` | 业务错误码 |
| 28 | `consumer-bff/handler/jwt/handler.go` | JWT（Login 从 context 取 tenant_id）+ LoginByPhone + OAuthLogin |
| 29 | `consumer-bff/handler/middleware/login_jwt.go` | LoginJWT 中间件 |
| 30 | `consumer-bff/handler/middleware/tenant_resolve.go` | TenantResolve 域名解析中间件 |
| 31 | `consumer-bff/handler/user.go` | 注册/资料/地址端点（8 个） |
| 32 | `consumer-bff/handler/tenant.go` | 店铺信息端点（1 个） |
| 33 | `consumer-bff/ioc/grpc.go` | gRPC client 初始化 |
| 34 | `consumer-bff/ioc/gin.go` | Gin 引擎 + 路由（公开+登录两组） |
| 35 | `consumer-bff/ioc/logger.go` | Logger |
| 36 | `consumer-bff/config/dev.yaml` | 配置（端口 8080） |
| 37 | `consumer-bff/app.go` | App |
| 38 | `consumer-bff/wire.go` | Wire DI |
| 39 | `consumer-bff/main.go` | 入口 |

共 39 个文件。

## 参考文件

- 设计文档：`docs/plans/2026-03-07-bff-design.md`
- 用户 proto：`api/proto/user/v1/user.proto`
- 租户 proto：`api/proto/tenant/v1/tenant.proto`
- 通用包：`pkg/ginx/`（wrapper + CORS）、`pkg/tenantx/`（context + gRPC 拦截器）
- 模式参考：`user/grpc/user.go`、`user/wire.go`、`user/main.go`
