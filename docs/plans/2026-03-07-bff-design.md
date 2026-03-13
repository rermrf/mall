# BFF 层设计文档（user/tenant 相关）

## 日期

2026-03-07

## 概述

SaaS 多租户商城的 3 个 BFF（Backend for Frontend）网关层实现。当前只有 user-svc 和 tenant-svc 就绪，本次仅实现与这两个服务相关的端点，后续服务就绪后增量补充 handler。

3 个 BFF 各自独立部署，不共享代码，沿用项目已有的目录结构。

## 架构决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| BFF 范围 | 全部 3 个 | 基础设施一次到位，后续只加 handler |
| JWT 策略 | BFF 自管 | 登录时调 user-svc 验证身份，BFF 签发/验证 token |
| Token 方案 | 双 token | access_token 30min + refresh_token 7d |
| 代码共享 | 各自独立 | 3 个 BFF 各自实现 JWT/middleware/client，互不依赖 |
| 目录结构 | 沿用已有 | client/handler/jwt/middleware/errs/ioc/config |

## 通用模式

### BFF 内部结构

```
{bff}/
├── client/              # gRPC 客户端工厂
│   ├── user.go
│   └── tenant.go
├── handler/
│   ├── jwt/
│   │   └── handler.go   # JWTHandler: Login/Logout/Refresh
│   ├── middleware/
│   │   ├── login_jwt.go  # LoginJWT 中间件
│   │   └── ...           # 各 BFF 特有中间件
│   ├── user.go           # UserHandler
│   └── tenant.go         # TenantHandler
├── errs/
│   └── code.go           # 业务错误码
├── ioc/
│   ├── grpc.go           # gRPC 连接（etcd 发现）
│   ├── gin.go            # Gin 引擎 + 路由注册
│   ├── logger.go
│   └── redis.go          # JWT 黑名单（可选）
├── config/
│   └── dev.yaml
├── app.go
├── wire.go
└── main.go
```

### JWT 双 Token

- access_token：30min TTL，`Authorization: Bearer <token>` 请求头
- refresh_token：7d TTL，`X-Refresh-Token` 请求头
- 登录响应通过 `X-Jwt-Token` 和 `X-Refresh-Token` 响应头返回
- Claims：`{uid, tenant_id, user_agent, exp, iat}`

### 统一响应

复用 `pkg/ginx.Result`：
```json
{"code": 0, "msg": "ok", "data": {...}}
```

## admin-bff（端口 8280）

### 职责

平台管理员操作：商家审核/冻结、套餐 CRUD、用户管理、平台 RBAC。

### 中间件链

```
CORS → LoginJWT → AdminOnly(tenant_id==0)
```

- `AdminOnly`：校验 JWT claims 中 tenant_id == 0，仅平台管理员可访问

### HTTP 端点（16 个）

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| POST | /api/v1/login | JWTHandler | 管理员登录 |
| POST | /api/v1/logout | JWTHandler | 登出 |
| POST | /api/v1/refresh-token | JWTHandler | 刷新 token |
| GET | /api/v1/users | UserHandler | 用户列表（分页+搜索） |
| POST | /api/v1/users/:id/status | UserHandler | 冻结/解冻用户 |
| GET | /api/v1/roles | UserHandler | 角色列表 |
| POST | /api/v1/roles | UserHandler | 创建角色 |
| PUT | /api/v1/roles/:id | UserHandler | 更新角色 |
| POST | /api/v1/tenants | TenantHandler | 商家入驻 |
| GET | /api/v1/tenants | TenantHandler | 商家列表 |
| GET | /api/v1/tenants/:id | TenantHandler | 商家详情 |
| POST | /api/v1/tenants/:id/approve | TenantHandler | 审核通过/拒绝 |
| POST | /api/v1/tenants/:id/freeze | TenantHandler | 冻结/解冻 |
| GET | /api/v1/plans | TenantHandler | 套餐列表 |
| POST | /api/v1/plans | TenantHandler | 创建套餐 |
| PUT | /api/v1/plans/:id | TenantHandler | 更新套餐 |

### gRPC 依赖

user-svc、tenant-svc

## merchant-bff（端口 8180）

### 职责

商家员工操作：店铺管理、配额查看、员工/角色管理。

### 中间件链

```
CORS → LoginJWT → TenantExtract(JWT claims → tenant_id)
```

- 登录时需指定 `tenant_id`（商家员工属于特定租户）
- `TenantExtract` 从 JWT claims 提取 tenant_id 注入 context

### HTTP 端点（13 个）

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| POST | /api/v1/login | JWTHandler | 商家员工登录（需传 tenant_id） |
| POST | /api/v1/logout | JWTHandler | 登出 |
| POST | /api/v1/refresh-token | JWTHandler | 刷新 token |
| GET | /api/v1/profile | UserHandler | 当前员工信息 |
| PUT | /api/v1/profile | UserHandler | 更新个人资料 |
| GET | /api/v1/staff | UserHandler | 员工列表（当前租户） |
| GET | /api/v1/roles | UserHandler | 角色列表（当前租户） |
| POST | /api/v1/roles | UserHandler | 创建角色 |
| PUT | /api/v1/roles/:id | UserHandler | 更新角色 |
| POST | /api/v1/staff/:id/role | UserHandler | 分配角色 |
| GET | /api/v1/shop | TenantHandler | 店铺信息 |
| PUT | /api/v1/shop | TenantHandler | 更新店铺 |
| GET | /api/v1/quotas/:type | TenantHandler | 配额使用量 |

### gRPC 依赖

user-svc、tenant-svc

## consumer-bff（端口 8080）

### 职责

C 端顾客操作：注册登录、个人资料、地址管理、店铺浏览。

### 中间件链

```
CORS → TenantResolve(域名→tenant_id) → [LoginJWT（需登录的路由）]
```

- `TenantResolve`：从请求 Host 解析域名 → 调用 tenant-svc `GetShopByDomain` → 获取 tenant_id 注入 context。所有请求都经过此中间件。
- `LoginJWT`：仅在需要登录的路由组上使用

### HTTP 端点（14 个）

| 方法 | 路径 | Handler | 公开 | 说明 |
|------|------|---------|------|------|
| POST | /api/v1/signup | UserHandler | 是 | 注册 |
| POST | /api/v1/login | JWTHandler | 是 | 密码登录 |
| POST | /api/v1/sms/send | UserHandler | 是 | 发送验证码 |
| POST | /api/v1/login/phone | JWTHandler | 是 | 手机号登录 |
| POST | /api/v1/login/oauth | JWTHandler | 是 | 第三方登录 |
| POST | /api/v1/logout | JWTHandler | 否 | 登出 |
| POST | /api/v1/refresh-token | JWTHandler | 否 | 刷新 token |
| GET | /api/v1/profile | UserHandler | 否 | 个人资料 |
| PUT | /api/v1/profile | UserHandler | 否 | 更新资料 |
| GET | /api/v1/addresses | UserHandler | 否 | 地址列表 |
| POST | /api/v1/addresses | UserHandler | 否 | 创建地址 |
| PUT | /api/v1/addresses/:id | UserHandler | 否 | 更新地址 |
| DELETE | /api/v1/addresses/:id | UserHandler | 否 | 删除地址 |
| GET | /api/v1/shop | TenantHandler | 是 | 当前店铺信息 |

### gRPC 依赖

user-svc、tenant-svc

## 文件清单

每个 BFF 约 12-15 个文件：

| # | 文件 | 说明 |
|---|------|------|
| 1 | `client/user.go` | user-svc gRPC 客户端 |
| 2 | `client/tenant.go` | tenant-svc gRPC 客户端 |
| 3 | `handler/jwt/handler.go` | JWT 登录/登出/刷新 |
| 4 | `handler/middleware/login_jwt.go` | LoginJWT 中间件 |
| 5 | `handler/middleware/*.go` | 各 BFF 特有中间件 |
| 6 | `handler/user.go` | 用户相关端点 |
| 7 | `handler/tenant.go` | 租户相关端点 |
| 8 | `errs/code.go` | 业务错误码 |
| 9 | `ioc/grpc.go` | gRPC 连接初始化 |
| 10 | `ioc/gin.go` | Gin 引擎 + 路由 |
| 11 | `ioc/logger.go` | Logger 初始化 |
| 12 | `config/dev.yaml` | 开发配置 |
| 13 | `app.go` | App 聚合 |
| 14 | `wire.go` | Wire DI |
| 15 | `main.go` | 入口 |

3 个 BFF 总计约 40-45 个文件。
