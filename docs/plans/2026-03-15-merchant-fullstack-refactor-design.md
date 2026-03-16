# 商家端全栈重构设计

> 日期：2026-03-15
> 范围：merchant-bff + merchant-frontend + pkg/ 共享包 + 必要时微服务层
> 方案：一次性全量重构

---

## 问题清单

### 后端（merchant-bff）

| # | 严重性 | 问题 | 位置 |
|---|--------|------|------|
| B1 | 严重 | 40+ 处 unsafe 类型断言 `tenantId.(int64)`，context 值缺失时直接 panic | 几乎所有 handler |
| B2 | 严重 | 租户隔离不完整 — 只检查 tenant_id 存在，不验证用户归属关系 | middleware/ |
| B3 | 高 | 错误处理不一致 — 混用 `ginx.WrapBody` 和原生 `ctx.JSON` | handler 层 |
| B4 | 高 | Magic status numbers（如 `Status: 3` 已发货）无常量定义 | order/logistics handler |
| B5 | 高 | 部分 handler 缺少入参校验 | order/payment handler |
| B6 | 中 | 每个 handler 重复提取 uid/tenant_id，无复用 | 所有 handler |
| B7 | 中 | gRPC 调用无超时控制、无熔断 | ioc/grpc.go |
| B8 | 中 | JWT 密钥写在 dev.yaml 里 | config/dev.yaml |
| B9 | 低 | errs/code.go 与 pkg/ginx/errcode.go 存在重复错误码 | errs/ |
| B10 | 低 | client/ 空目录、handler 中未使用变量 | 多处 |

### 前端（merchant-frontend）

| # | 严重性 | 问题 | 位置 |
|---|--------|------|------|
| F1 | 严重 | CouponForm 编辑时加载 100 条再 find | pages/marketing/CouponForm.tsx |
| F2 | 严重 | StockList N+1 查询 — 先取所有商品再逐一取库存 | pages/inventory/StockList.tsx |
| F3 | 严重 | 大量 `.catch(() => {})` 静默吞掉错误 | 多处（Dashboard 等） |
| F4 | 高 | 硬编码 statusMap/typeMap magic numbers 散落各文件 | order/marketing 页面 |
| F5 | 高 | ProTable `request` 内联 API 调用，无法复用/测试 | 所有列表页 |
| F6 | 高 | Token 存 localStorage，存在 XSS 风险 | api/client.ts |
| F7 | 中 | 表单 state 用 `Record<string, unknown>`，类型不安全 | 多个表单页 |
| F8 | 中 | 只有根级 ErrorBoundary，子组件崩溃白屏整个应用 | components/ |
| F9 | 中 | 价格 分/元 转换散落各处，无统一工具函数 | 多处 |

---

## 设计方案

### Part 1：后端安全与健壮性加固

#### 1.1 类型断言安全化

在 `pkg/ginx` 新增安全上下文提取函数：

```go
// pkg/ginx/context.go
func GetUID(ctx *gin.Context) (int64, error) {
    val, exists := ctx.Get("uid")
    if !exists {
        return 0, fmt.Errorf("uid not found in context")
    }
    uid, ok := val.(int64)
    if !ok {
        return 0, fmt.Errorf("uid type assertion failed")
    }
    return uid, nil
}

func GetTenantID(ctx *gin.Context) (int64, error) { /* 同上 */ }
```

所有 handler 替换裸断言为 `ginx.GetUID(ctx)` / `ginx.GetTenantID(ctx)`，错误时返回 401。

#### 1.2 租户隔离增强

在 TenantExtract 中间件中：
- 对比 JWT claims 中的 TenantId 与请求上下文中的 tenant_id 是否一致
- 不一致返回 403 Forbidden

#### 1.3 gRPC 超时与重试

在 `pkg/grpcx` 封装 `DialWithDefaults`：
- 默认 5s 调用超时
- 最多 3 次重试（可配置）
- `WaitForReady(false)` 避免无限等待
- 所有 `ioc/grpc.go` 中的 Dial 改用此函数

#### 1.4 统一错误处理模式

全部 handler 统一使用 `ginx.WrapBody` / `ginx.WrapBodyAndToken`，消除原生 `ctx.JSON` 的混用。

### Part 2：后端代码质量改进

#### 2.1 常量定义

新建 `merchant-bff/domain/consts.go`：

```go
// 订单状态
const (
    OrderStatusPending   = 0 // 待付款
    OrderStatusPaid      = 1 // 已付款
    OrderStatusShipping  = 2 // 待发货
    OrderStatusShipped   = 3 // 已发货
    OrderStatusCompleted = 4 // 已完成
    OrderStatusCancelled = 5 // 已取消
)

// 操作者类型
const (
    OperatorTypeSystem   = 0
    OperatorTypeCustomer = 1
    OperatorTypeMerchant = 2
)

// 优惠券类型
const (
    CouponTypeFixed    = 1 // 满减
    CouponTypeDiscount = 2 // 折扣
    CouponTypeGift     = 3 // 赠品
)
```

Handler 中引用 `domain.OrderStatusShipped` 替代裸数字。

#### 2.2 Handler 公共逻辑提取

结合 1.1 的 `ginx.GetUID/GetTenantID`，消除 40+ 处重复提取逻辑。

#### 2.3 错误码去重

- 删除 `merchant-bff/errs/code.go` 中与 `pkg/ginx/errcode.go` 重复的定义
- BFF 层只保留业务特有错误码

#### 2.4 入参校验补全

- 对所有 handler 的 path param / query param 补充 `binding` tag
- 利用 `validatorx` 包统一校验规则
- 添加业务规则自定义 validator（金额 >= 0、页码 > 0 等）

### Part 3：前端数据获取与性能修复

#### 3.1 CouponForm 加载优化

- 确认后端有 `getCoupon(id)` 接口（或新增）
- 前端改为 `getCoupon(id)` 直接获取单条
- 检查并修复其他表单页类似问题

#### 3.2 StockList N+1 查询消除

- 优先：后端新增聚合接口 `/inventory/list`，一次返回带库存的商品列表
- 备选：前端并行请求 + 分页限制

#### 3.3 静默错误修复

- 创建 `src/utils/error.ts` 封装统一错误处理
- 所有 `.catch(() => {})` 替换为有意义的错误处理
- 列表加载失败 → 显示 Empty + 重试按钮

#### 3.4 常量集中管理

新建 `src/constants/` 目录：
- `order.ts` — 订单状态、退款状态
- `marketing.ts` — 优惠券类型、活动状态
- `common.ts` — 通用映射
- 值与后端 `domain/consts.go` 一一对应

### Part 4：前端代码质量与类型安全

#### 4.1 表单类型安全

- 为每个表单定义 `XxxFormValues` 接口
- 使用 `<ProForm<CouponFormValues>>` 泛型
- 消除所有 `as` 强制类型断言

#### 4.2 ProTable 数据层标准化

- 提取 `request` 逻辑到独立函数/hook
- ProTable 只做展示 + 调用函数

#### 4.3 前端常量与后端对齐

- `src/constants/` 的值与后端 `domain/consts.go` 一一对应
- 注释中标注后端常量名
- 价格 分/元 转换封装为 `formatPrice` / `parsePrice`

#### 4.4 错误边界细化

- 路由级别增加 ErrorBoundary
- 表单提交失败 → 行内错误提示
- API 错误区分「可重试」和「不可重试」

### Part 5：中间件与基础设施

#### 5.1 JWT 密钥外置

- dev 环境保持 yaml
- 生产环境通过环境变量注入（Viper env > yaml 优先级）

#### 5.2 未使用代码清理

- 删除 `client/` 空目录
- 清理 `_ = tenantId` 等无用代码

---

## 改动文件清单

| 层级 | 改动 | 文件数 |
|------|------|--------|
| `pkg/ginx` | 新增 context 安全提取、统一错误码 | 2-3 |
| `pkg/grpcx` | 新增超时/重试封装 | 1 |
| `merchant-bff/handler` | 全部 handler 安全化 + 统一模式 + 常量引用 | 11 |
| `merchant-bff/middleware` | 租户校验增强 | 1 |
| `merchant-bff/domain` | 新增常量定义 | 1（新建） |
| `merchant-bff/errs` | 去重、清理 | 1 |
| `merchant-frontend/src/constants` | 新增常量文件 | 3-4（新建） |
| `merchant-frontend/src/utils` | 错误处理、价格转换工具 | 2（新建） |
| `merchant-frontend/src/pages` | 表单类型、数据获取、错误处理 | 10+ |
| `merchant-frontend/src/api` | 接口调整 | 2-3 |
| `merchant-frontend/src/components` | ErrorBoundary 细化 | 1 |
