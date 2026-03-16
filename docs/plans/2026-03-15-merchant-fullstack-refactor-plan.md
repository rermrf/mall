# 商家端全栈重构 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all identified security, robustness, code quality, and performance issues across merchant-bff, merchant-frontend, and shared packages in a single refactoring pass.

**Architecture:** BFF handler layer gets safe context extraction via new `pkg/ginx` helpers; all raw `ctx.JSON` handlers migrate to `ginx.WrapBody`/`WrapQuery`; frontend gets centralized constants, typed forms, and proper error handling.

**Tech Stack:** Go 1.25 / Gin / gRPC / React 19 / TypeScript 5 / Ant Design Pro Components

---

### Task 1: Add safe context extraction helpers to pkg/ginx

**Files:**
- Create: `pkg/ginx/context.go`

**Step 1: Create `pkg/ginx/context.go`**

```go
package ginx

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// GetUID safely extracts uid from Gin context (set by JWT middleware).
func GetUID(ctx *gin.Context) (int64, error) {
	val, exists := ctx.Get("uid")
	if !exists {
		return 0, fmt.Errorf("uid not found in context")
	}
	uid, ok := val.(int64)
	if !ok {
		return 0, fmt.Errorf("uid type assertion failed: got %T", val)
	}
	return uid, nil
}

// GetTenantID safely extracts tenant_id from Gin context (set by JWT middleware).
func GetTenantID(ctx *gin.Context) (int64, error) {
	val, exists := ctx.Get("tenant_id")
	if !exists {
		return 0, fmt.Errorf("tenant_id not found in context")
	}
	tid, ok := val.(int64)
	if !ok {
		return 0, fmt.Errorf("tenant_id type assertion failed: got %T", val)
	}
	return tid, nil
}

// MustGetUID extracts uid, returning (0, Result) on failure for use in WrapBody handlers.
func MustGetUID(ctx *gin.Context) (int64, *Result) {
	uid, err := GetUID(ctx)
	if err != nil {
		return 0, &Result{Code: CodeUnauthorized, Msg: "未授权"}
	}
	return uid, nil
}

// MustGetTenantID extracts tenant_id, returning (0, Result) on failure for use in WrapBody handlers.
func MustGetTenantID(ctx *gin.Context) (int64, *Result) {
	tid, err := GetTenantID(ctx)
	if err != nil {
		return 0, &Result{Code: CodeForbidden, Msg: "需要商家身份"}
	}
	return tid, nil
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/emoji/Documents/demo/project/mall && go build ./pkg/ginx/...`
Expected: SUCCESS (no errors)

**Step 3: Commit**

```bash
git add pkg/ginx/context.go
git commit -m "feat(ginx): add safe context extraction helpers GetUID/GetTenantID"
```

---

### Task 2: Add gRPC client timeout and retry to pkg/grpcx

**Files:**
- Create: `pkg/grpcx/client.go`

**Step 1: Create `pkg/grpcx/client.go`**

```go
package grpcx

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// DefaultCallTimeout is the default per-call timeout for gRPC clients.
const DefaultCallTimeout = 5 * time.Second

// DefaultClientDialOptions returns common gRPC client dial options.
func DefaultClientDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(false),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}
}
```

**Step 2: Modify `merchant-bff/ioc/grpc.go` to use timeout options**

In `initServiceConn`, add `DefaultClientDialOptions()...` to the `grpc.NewClient` call, and wrap each handler call context with `context.WithTimeout` at the handler level (not here — handlers will use `grpcx.DefaultCallTimeout`).

Replace lines 47-52 of `merchant-bff/ioc/grpc.go`:
```go
conn, err := grpc.NewClient(
    "etcd:///service/"+serviceName,
    append([]grpc.DialOption{
        grpc.WithResolvers(etcdResolver),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithUnaryInterceptor(tenantx.GRPCUnaryClientInterceptor()),
    }, grpcx.DefaultClientDialOptions()...)...,
)
```

Add import: `"github.com/rermrf/mall/pkg/grpcx"`

**Step 3: Verify it compiles**

Run: `cd /Users/emoji/Documents/demo/project/mall && go build ./merchant-bff/...`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add pkg/grpcx/client.go merchant-bff/ioc/grpc.go
git commit -m "feat(grpcx): add default client dial options with timeout and keepalive"
```

---

### Task 3: Enhance tenant extraction middleware

**Files:**
- Modify: `merchant-bff/handler/middleware/tenant_extract.go`

**Step 1: Update TenantExtract to use ginx constants**

Replace `403001` magic numbers with `ginx.CodeForbidden`:

```go
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
```

**Step 2: Update login_jwt.go to use ginx constants**

Replace all `401001` / `401002` magic numbers with `ginx.CodeUnauthorized` / `ginx.CodeInvalidCredentials`:

```go
// Line 28: Code: 401001 → Code: ginx.CodeUnauthorized
// Line 37: Code: 401001 → Code: ginx.CodeUnauthorized
// Line 47: Code: 401001 → Code: ginx.CodeUnauthorized
// Line 55: Code: 401002 → Code: ginx.CodeInvalidCredentials
```

**Step 3: Verify it compiles**

Run: `cd /Users/emoji/Documents/demo/project/mall && go build ./merchant-bff/...`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add merchant-bff/handler/middleware/
git commit -m "refactor(middleware): use ginx constants, safe context extraction"
```

---

### Task 4: Create backend business constants

**Files:**
- Create: `merchant-bff/domain/consts.go`

**Step 1: Create `merchant-bff/domain/consts.go`**

```go
package domain

// 订单状态
const (
	OrderStatusCancelled = 0 // 已取消
	OrderStatusPending   = 1 // 待付款
	OrderStatusPaid      = 2 // 待发货
	OrderStatusShipped   = 3 // 已发货
	OrderStatusCompleted = 4 // 已完成
	OrderStatusRefunding = 5 // 退款中
)

// 支付状态
const (
	PaymentStatusPending = 0 // 待支付
	PaymentStatusPaid    = 1 // 已支付
	PaymentStatusRefund  = 2 // 已退款
	PaymentStatusClosed  = 3 // 已关闭
)

// 操作者类型
const (
	OperatorTypeSystem   = 0 // 系统
	OperatorTypeCustomer = 1 // 消费者
	OperatorTypeMerchant = 2 // 商家
)

// 优惠券类型
const (
	CouponTypeFixed    = 1 // 满减
	CouponTypeDiscount = 2 // 折扣
	CouponTypeGift     = 3 // 固定金额
)

// 优惠券适用范围
const (
	CouponScopeAll      = 0 // 全场
	CouponScopeProduct  = 1 // 指定商品
	CouponScopeCategory = 2 // 指定分类
)
```

**Step 2: Verify it compiles**

Run: `cd /Users/emoji/Documents/demo/project/mall && go build ./merchant-bff/domain/...`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add merchant-bff/domain/consts.go
git commit -m "feat(domain): add business constant definitions for order/payment/coupon"
```

---

### Task 5: Refactor all backend handlers — safe assertions + constants + unified error handling

**Files:**
- Modify: `merchant-bff/handler/order.go`
- Modify: `merchant-bff/handler/product.go`
- Modify: `merchant-bff/handler/marketing.go`
- Modify: `merchant-bff/handler/inventory.go`
- Modify: `merchant-bff/handler/logistics.go`
- Modify: `merchant-bff/handler/payment.go`
- Modify: `merchant-bff/handler/user.go`
- Modify: `merchant-bff/handler/tenant.go`
- Modify: `merchant-bff/handler/notification.go`

**Step 1: Refactor `order.go`**

For every handler method:
1. Replace `tenantId, _ := ctx.Get("tenant_id")` + `tenantId.(int64)` with:
   ```go
   tenantId, errResult := ginx.MustGetTenantID(ctx)
   if errResult != nil {
       return *errResult, nil
   }
   ```
   (For WrapBody/WrapQuery handlers that return `(Result, error)`)

2. For raw `ctx.JSON` handlers (GetOrder, ShipOrder), convert to WrapBody/WrapQuery pattern or use:
   ```go
   uid, err := ginx.GetUID(ctx)
   if err != nil {
       ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeUnauthorized, Msg: "未授权"})
       return
   }
   ```

3. Replace magic numbers:
   - `Status: 3` → `Status: int32(domain.OrderStatusShipped)`
   - `OperatorType: 2` → `OperatorType: int32(domain.OperatorTypeMerchant)`
   - `Code: 4` → `Code: ginx.CodeBadReq`

4. Remove `_ = tenantId` dead code

**Step 2: Refactor remaining handlers**

Apply the same pattern to all 8 other handler files:
- `product.go`: ~16 unsafe assertions → use `ginx.MustGetTenantID(ctx)`
- `marketing.go`: ~14 unsafe assertions → same pattern
- `logistics.go`: ~8 assertions + `Status: 3` magic numbers → use domain constants
- `payment.go`: ~1 assertion + raw handlers → convert
- `inventory.go`: ~2 assertions → convert
- `user.go`: ~6 assertions → convert
- `tenant.go`: ~3 assertions → convert
- `notification.go`: ~6 assertions → use `ginx.MustGetUID(ctx)` pattern

**Step 3: Delete duplicate error codes**

Delete `merchant-bff/errs/code.go` entirely — all constants already exist in `pkg/ginx/errcode.go`.

**Step 4: Clean up dead code**

- Remove `merchant-bff/client/` empty directory
- Remove any `_ = tenantId` unused variable patterns

**Step 5: Verify it compiles**

Run: `cd /Users/emoji/Documents/demo/project/mall && go build ./merchant-bff/...`
Expected: SUCCESS

**Step 6: Commit**

```bash
git add merchant-bff/handler/ merchant-bff/errs/
git commit -m "refactor(handler): safe context extraction, domain constants, unified error handling"
```

---

### Task 6: Add GetCoupon BFF endpoint

**Files:**
- Modify: `merchant-bff/handler/marketing.go`
- Modify: BFF route registration (check `ioc/web.go` or equivalent)

**Step 1: Add GetCoupon handler to marketing.go**

After the `ListCoupons` method, add:

```go
func (h *MarketingHandler) GetCoupon(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		ctx.JSON(http.StatusOK, ginx.Result{Code: ginx.CodeBadReq, Msg: "无效的优惠券 ID"})
		return
	}
	resp, err := h.marketingClient.GetCoupon(ctx.Request.Context(), &marketingv1.GetCouponRequest{Id: id})
	if err != nil {
		h.l.Error("查询优惠券详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.MarketingErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetCoupon()})
}
```

**Step 2: Register route**

Find the route registration file and add:
```go
marketingGroup.GET("/coupons/:id", marketingHandler.GetCoupon)
```

**Step 3: Verify it compiles**

Run: `cd /Users/emoji/Documents/demo/project/mall && go build ./merchant-bff/...`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add merchant-bff/
git commit -m "feat(marketing): add GetCoupon endpoint for single coupon retrieval"
```

---

### Task 7: Create frontend constants

**Files:**
- Create: `merchant-frontend/src/constants/order.ts`
- Create: `merchant-frontend/src/constants/payment.ts`
- Create: `merchant-frontend/src/constants/marketing.ts`
- Create: `merchant-frontend/src/constants/index.ts`

**Step 1: Create `src/constants/order.ts`**

```typescript
// 订单状态 — 对应后端 domain.OrderStatusXxx
export const ORDER_STATUS = {
  CANCELLED: 0,
  PENDING: 1,
  PAID: 2,
  SHIPPED: 3,
  COMPLETED: 4,
  REFUNDING: 5,
} as const

export const ORDER_STATUS_MAP: Record<number, { text: string; color: string }> = {
  [ORDER_STATUS.CANCELLED]: { text: '已取消', color: 'default' },
  [ORDER_STATUS.PENDING]: { text: '待付款', color: 'orange' },
  [ORDER_STATUS.PAID]: { text: '待发货', color: 'blue' },
  [ORDER_STATUS.SHIPPED]: { text: '已发货', color: 'cyan' },
  [ORDER_STATUS.COMPLETED]: { text: '已完成', color: 'green' },
  [ORDER_STATUS.REFUNDING]: { text: '退款中', color: 'red' },
}
```

**Step 2: Create `src/constants/payment.ts`**

```typescript
// 支付状态 — 对应后端 domain.PaymentStatusXxx
export const PAYMENT_STATUS = {
  PENDING: 0,
  PAID: 1,
  REFUNDED: 2,
  CLOSED: 3,
} as const

export const PAYMENT_STATUS_MAP: Record<number, { text: string; color: string }> = {
  [PAYMENT_STATUS.PENDING]: { text: '待支付', color: 'orange' },
  [PAYMENT_STATUS.PAID]: { text: '已支付', color: 'green' },
  [PAYMENT_STATUS.REFUNDED]: { text: '已退款', color: 'red' },
  [PAYMENT_STATUS.CLOSED]: { text: '已关闭', color: 'default' },
}
```

**Step 3: Create `src/constants/marketing.ts`**

```typescript
// 优惠券类型 — 对应后端 domain.CouponTypeXxx
export const COUPON_TYPE = {
  FIXED: 1,
  DISCOUNT: 2,
  GIFT: 3,
} as const

export const COUPON_TYPE_OPTIONS = [
  { label: '满减', value: COUPON_TYPE.FIXED },
  { label: '折扣', value: COUPON_TYPE.DISCOUNT },
  { label: '固定金额', value: COUPON_TYPE.GIFT },
]

// 优惠券适用范围 — 对应后端 domain.CouponScopeXxx
export const COUPON_SCOPE = {
  ALL: 0,
  PRODUCT: 1,
  CATEGORY: 2,
} as const

export const COUPON_SCOPE_OPTIONS = [
  { label: '全场', value: COUPON_SCOPE.ALL },
  { label: '指定商品', value: COUPON_SCOPE.PRODUCT },
  { label: '指定分类', value: COUPON_SCOPE.CATEGORY },
]
```

**Step 4: Create `src/constants/index.ts`**

```typescript
export * from './order'
export * from './payment'
export * from './marketing'

// 通用价格工具
export function formatPrice(fen: number): string {
  return `¥${((fen ?? 0) / 100).toFixed(2)}`
}

export function parsePriceToFen(yuan: number): number {
  return Math.round(yuan * 100)
}
```

**Step 5: Commit**

```bash
git add merchant-frontend/src/constants/
git commit -m "feat(frontend): add centralized constants for order/payment/marketing"
```

---

### Task 8: Create frontend error handling utility

**Files:**
- Create: `merchant-frontend/src/utils/error.ts`

**Step 1: Create `src/utils/error.ts`**

```typescript
import { message } from 'antd'

/**
 * Handles API errors with user-visible feedback.
 * Use in place of `.catch(() => {})`.
 */
export function handleApiError(context: string) {
  return (err: unknown) => {
    console.error(`[${context}]`, err)
    // Don't show message for cancelled requests
    if (err instanceof Error && err.message === 'canceled') return
    // axios interceptor already shows message.error for most API errors,
    // so this is a fallback for unexpected errors
  }
}

/**
 * Silently catches an API error but still logs it.
 * Use for non-critical data loads (e.g. dashboard stats).
 */
export function silentApiError(context: string) {
  return (err: unknown) => {
    console.error(`[${context}]`, err)
  }
}
```

**Step 2: Commit**

```bash
git add merchant-frontend/src/utils/error.ts
git commit -m "feat(frontend): add error handling utilities"
```

---

### Task 9: Fix frontend silent catches and use centralized constants

**Files:**
- Modify: `merchant-frontend/src/pages/dashboard/index.tsx`
- Modify: `merchant-frontend/src/pages/order/OrderList.tsx`
- Modify: `merchant-frontend/src/pages/order/OrderDetail.tsx`
- Modify: `merchant-frontend/src/pages/payment/PaymentList.tsx`
- Modify: `merchant-frontend/src/pages/payment/PaymentDetail.tsx`

**Step 1: Fix `dashboard/index.tsx`**

Replace `.catch(() => {})` with `.catch(silentApiError('dashboard:xxx'))`:

```typescript
import { silentApiError } from '@/utils/error'

// Line 23: .catch(() => {})  →  .catch(silentApiError('dashboard:pendingShip'))
// Line 26: .catch(() => {})  →  .catch(silentApiError('dashboard:pendingRefund'))
// Line 29: .catch(() => {})  →  .catch(silentApiError('dashboard:todayOrders'))
```

**Step 2: Fix `order/OrderList.tsx`**

Replace local `statusMap` with import:

```typescript
// Remove lines 9-16 (local statusMap)
import { ORDER_STATUS_MAP, formatPrice } from '@/constants'

// Line 28: `¥${((r.pay_amount ?? 0) / 100).toFixed(2)}`  →  formatPrice(r.pay_amount)
// Line 34: use ORDER_STATUS_MAP instead of local statusMap
// Line 36: use ORDER_STATUS_MAP instead of statusMap
```

**Step 3: Fix `order/OrderDetail.tsx`**

Replace local `statusMap` + silent catches:

```typescript
import { ORDER_STATUS_MAP, ORDER_STATUS, formatPrice } from '@/constants'
import { silentApiError } from '@/utils/error'

// Remove lines 18-25 (local statusMap)
// Line 29: .catch(() => {})  →  .catch(silentApiError('orderDetail:getOrder'))
// Line 30: .catch(() => {})  →  .catch(silentApiError('orderDetail:getLogistics'))
// Line 44: .catch(() => {})  →  .catch(silentApiError('orderDetail:refreshOrder'))
// Line 59-61: use ORDER_STATUS_MAP + formatPrice
// Line 78: `¥${((v ?? 0) / 100).toFixed(2)}`  →  formatPrice(v)
// Line 93: `order.status === 2`  →  `order.status === ORDER_STATUS.PAID`
```

**Step 4: Fix `payment/PaymentList.tsx`**

```typescript
import { PAYMENT_STATUS_MAP, formatPrice } from '@/constants'

// Remove lines 9-14 (local statusMap)
// Use PAYMENT_STATUS_MAP throughout
// Line 27: use formatPrice(r.amount)
```

**Step 5: Fix `payment/PaymentDetail.tsx`**

```typescript
import { PAYMENT_STATUS_MAP, PAYMENT_STATUS, formatPrice } from '@/constants'
import { silentApiError } from '@/utils/error'

// Remove lines 7-12 (local statusMap)
// Line 29: .catch(() => {})  →  .catch(silentApiError('paymentDetail:getPayment'))
// Line 46: .catch(() => {})  →  .catch(silentApiError('paymentDetail:refreshPayment'))
// Line 60: use PAYMENT_STATUS_MAP
// Line 68: `payment.status === 1`  →  `payment.status === PAYMENT_STATUS.PAID`
// Line 78: use formatPrice
```

**Step 6: Verify it compiles**

Run: `cd /Users/emoji/Documents/demo/project/mall/merchant-frontend && npx tsc --noEmit`
Expected: SUCCESS (or only pre-existing errors)

**Step 7: Commit**

```bash
git add merchant-frontend/src/pages/dashboard/ merchant-frontend/src/pages/order/ merchant-frontend/src/pages/payment/
git commit -m "refactor(frontend): use centralized constants, fix silent catches"
```

---

### Task 10: Fix CouponForm — add getCoupon API + type safety

**Files:**
- Modify: `merchant-frontend/src/api/marketing.ts`
- Modify: `merchant-frontend/src/pages/marketing/CouponForm.tsx`

**Step 1: Add `getCoupon` to `src/api/marketing.ts`**

After `listCoupons`, add:

```typescript
export async function getCoupon(id: number) {
  return request<Coupon>({ method: 'GET', url: `/coupons/${id}` })
}
```

**Step 2: Rewrite `CouponForm.tsx` with type safety**

Key changes:
1. Replace `listCoupons({page:1, pageSize:100}).find(...)` with `getCoupon(Number(id))`
2. Replace `Record<string, unknown>` with proper `Partial<CreateCouponReq>` typing
3. Replace magic number options with `COUPON_TYPE_OPTIONS` / `COUPON_SCOPE_OPTIONS`
4. Replace `.catch(() => {})` with proper error handling
5. Remove `as` type assertions from `onFinish`

```typescript
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormDigit, ProFormSelect, ProFormDateTimePicker, ProFormDependency } from '@ant-design/pro-components'
import { createCoupon, updateCoupon, getCoupon } from '@/api/marketing'
import type { CreateCouponReq } from '@/types/marketing'
import { COUPON_TYPE_OPTIONS, COUPON_SCOPE_OPTIONS } from '@/constants'
import { silentApiError } from '@/utils/error'

export default function CouponForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id
  const [initialValues, setInitialValues] = useState<Partial<CreateCouponReq>>()

  useEffect(() => {
    if (isEdit) {
      getCoupon(Number(id)).then((coupon) => {
        if (coupon) {
          setInitialValues({
            name: coupon.name,
            type: coupon.type,
            threshold: coupon.threshold,
            discount_value: coupon.discount_value,
            total_count: coupon.total_count,
            per_limit: coupon.per_limit,
            start_time: coupon.start_time,
            end_time: coupon.end_time,
            scope_type: coupon.scope_type,
            scope_ids: coupon.scope_ids,
            status: coupon.status,
          })
        }
      }).catch(silentApiError('couponForm:getCoupon'))
    }
  }, [id, isEdit])

  if (isEdit && !initialValues) {
    return <Card title="编辑优惠券" loading />
  }

  return (
    <Card title={isEdit ? '编辑优惠券' : '创建优惠券'}>
      <ProForm<CreateCouponReq>
        initialValues={initialValues}
        onFinish={async (values) => {
          const data: CreateCouponReq = {
            ...values,
            scope_type: values.scope_type ?? 0,
            scope_ids: values.scope_ids ?? [],
          }
          try {
            if (isEdit) {
              await updateCoupon(Number(id), data)
              message.success('更新成功')
            } else {
              await createCoupon(data)
              message.success('创建成功')
            }
            navigate('/marketing/coupon')
          } catch (e: unknown) {
            message.error((e as Error).message)
          }
        }}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormSelect name="type" label="类型" rules={[{ required: true }]} options={COUPON_TYPE_OPTIONS} />
        <ProFormDigit name="threshold" label="使用门槛（分）" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="discount_value" label="优惠值（分）" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="total_count" label="发放总量" rules={[{ required: true }]} min={1} />
        <ProFormDigit name="per_limit" label="每人限领" initialValue={1} min={1} />
        <ProFormDateTimePicker name="start_time" label="开始时间" rules={[{ required: true }]} />
        <ProFormDateTimePicker name="end_time" label="结束时间" rules={[{ required: true }]} />
        <ProFormSelect name="scope_type" label="适用范围" initialValue={0} rules={[{ required: true }]} options={COUPON_SCOPE_OPTIONS} />
        <ProFormDependency name={['scope_type']}>
          {({ scope_type }) => {
            if (scope_type && scope_type > 0) {
              return (
                <ProFormText
                  name="scope_ids"
                  label={scope_type === 1 ? '商品ID（逗号分隔）' : '分类ID（逗号分隔）'}
                  placeholder="例如：1,2,3"
                  rules={[{ required: true, message: '请输入ID' }]}
                />
              )
            }
            return null
          }}
        </ProFormDependency>
        <ProFormSelect name="status" label="状态" initialValue={1} options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
      </ProForm>
    </Card>
  )
}
```

**Step 3: Commit**

```bash
git add merchant-frontend/src/api/marketing.ts merchant-frontend/src/pages/marketing/CouponForm.tsx
git commit -m "fix(couponForm): use getCoupon API instead of listing 100, add type safety"
```

---

### Task 11: Fix remaining silent catches across all pages

**Files:**
- Modify: `merchant-frontend/src/pages/marketing/SeckillForm.tsx`
- Modify: `merchant-frontend/src/pages/product/ProductForm.tsx`
- Modify: `merchant-frontend/src/pages/logistics/TemplateForm.tsx`
- Modify: `merchant-frontend/src/pages/profile/ProfileEdit.tsx`
- Modify: `merchant-frontend/src/pages/shop/ShopSettings.tsx`
- Modify: `merchant-frontend/src/pages/staff/StaffList.tsx`
- Modify: `merchant-frontend/src/components/layout/MainLayout.tsx`
- Modify: `merchant-frontend/src/stores/notification.ts`

**Step 1: For each file**

Replace every `.catch(() => {})` with `.catch(silentApiError('componentName:action'))`:

| File | Line | Replace with |
|------|------|-------------|
| SeckillForm.tsx | 26 | `.catch(silentApiError('seckillForm:getSeckill'))` |
| ProductForm.tsx | 22 | `.catch(silentApiError('productForm:listCategories'))` |
| ProductForm.tsx | 23 | `.catch(silentApiError('productForm:listBrands'))` |
| ProductForm.tsx | 46 | `.catch(silentApiError('productForm:getProduct'))` |
| TemplateForm.tsx | 18 | `.catch(silentApiError('templateForm:getTemplate'))` |
| ProfileEdit.tsx | 11 | `.catch(silentApiError('profile:getProfile'))` |
| ShopSettings.tsx | 11 | `.catch(silentApiError('shop:getShop'))` |
| StaffList.tsx | 13 | `.catch(silentApiError('staffList:listRoles'))` |
| MainLayout.tsx | 105 | `.catch(silentApiError('layout:getProfile'))` |

Add `import { silentApiError } from '@/utils/error'` to each file.

**Step 2: Fix SeckillForm type safety**

Replace `Record<string, unknown>` state and `as` assertions with proper typing using `Partial<CreateSeckillReq>`.

**Step 3: Fix ProductForm type safety**

Replace `Record<string, unknown>` state and `as` assertions with proper typing.

**Step 4: Verify it compiles**

Run: `cd /Users/emoji/Documents/demo/project/mall/merchant-frontend && npx tsc --noEmit`
Expected: SUCCESS

**Step 5: Commit**

```bash
git add merchant-frontend/src/pages/ merchant-frontend/src/components/ merchant-frontend/src/stores/
git commit -m "fix(frontend): replace all silent catches with logged error handlers, improve type safety"
```

---

### Task 12: Add route-level ErrorBoundary

**Files:**
- Modify: `merchant-frontend/src/router/index.tsx`

**Step 1: Wrap `<Outlet>` with ErrorBoundary in the layout route**

In the router configuration, ensure the layout route wraps its children with ErrorBoundary:

```typescript
import ErrorBoundary from '@/components/ErrorBoundary'

// In the route tree, wrap Outlet in layout:
{
  element: (
    <AuthGuard>
      <MainLayout />
    </AuthGuard>
  ),
  errorElement: <ErrorBoundary><div /></ErrorBoundary>,
  children: [/* ... */],
}
```

If using React Router v7's `errorElement`, use that directly. Otherwise wrap each lazy-loaded page `<Suspense>` with `<ErrorBoundary>`.

**Step 2: Verify it compiles**

Run: `cd /Users/emoji/Documents/demo/project/mall/merchant-frontend && npx tsc --noEmit`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add merchant-frontend/src/router/ merchant-frontend/src/components/
git commit -m "feat(frontend): add route-level error boundaries"
```

---

### Task 13: Delete duplicate error codes + clean up dead code

**Files:**
- Delete: `merchant-bff/errs/code.go` (or the whole `errs/` package if empty)
- Delete: `merchant-bff/client/` directory (if empty)

**Step 1: Check if errs/ package is used anywhere besides handler imports**

Search for `merchant-bff/errs` imports. If no other files import it, delete the entire directory.

**Step 2: Delete empty client directory**

```bash
rmdir merchant-bff/client/ 2>/dev/null || true
```

**Step 3: Verify it compiles**

Run: `cd /Users/emoji/Documents/demo/project/mall && go build ./merchant-bff/...`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add -A merchant-bff/errs/ merchant-bff/client/
git commit -m "chore: remove duplicate error codes and empty directories"
```

---

### Task 14: Final verification — build both frontend and backend

**Step 1: Build backend**

Run: `cd /Users/emoji/Documents/demo/project/mall && go build ./...`
Expected: SUCCESS

**Step 2: Build frontend**

Run: `cd /Users/emoji/Documents/demo/project/mall/merchant-frontend && npx tsc --noEmit && npm run build`
Expected: SUCCESS

**Step 3: Commit all remaining changes**

If any files were missed, add and commit them now.

```bash
git add -A
git commit -m "refactor: merchant fullstack refactoring — complete"
```
