# Platform Admin Frontend Design

**Date**: 2026-03-17
**Status**: Approved

## Overview

Add a new platform administration frontend (`admin-frontend/`) to the mall microservices platform. This frontend serves platform administrators (tenant_id=0) and provides full management and supervision capabilities over the entire platform. It connects to the existing `admin-bff` service (port 8280) which already exposes 54+ API endpoints.

## Architecture Decision

**Approach**: Independent project, reusing merchant-frontend patterns.

Create a standalone React project at `/admin-frontend/` with identical tech stack and code patterns as `merchant-frontend`, but with fully independent codebase.

**Rationale**: Independent deployment, no coupling with merchant-frontend, consistent patterns for easy maintenance. Minor code duplication (api/client.ts, AuthGuard) is acceptable trade-off.

## Tech Stack

| Component | Choice | Version |
|-----------|--------|---------|
| Framework | React | 19.x |
| Build Tool | Vite | 7.x |
| Language | TypeScript | 5.9.x |
| UI Library | Ant Design + Pro Components | 5.x + 2.x |
| State Management | Zustand | 5.x |
| HTTP Client | Axios | 1.x |
| Routing | React Router | 7.x |

**Dev Server**: Port 3002, proxy `/api` to `localhost:8280` (admin-bff).

## Project Structure

```
admin-frontend/
├── package.json
├── vite.config.ts            # port 3002, proxy → localhost:8280
├── tsconfig.json
├── index.html
└── src/
    ├── main.tsx
    ├── App.tsx               # ConfigProvider (zh_CN) + RouterProvider
    ├── api/
    │   ├── client.ts         # Axios instance, 401 refresh queue, Bearer token
    │   ├── auth.ts           # login (no tenantId), logout, refreshToken
    │   ├── user.ts           # listUsers, updateUserStatus
    │   ├── role.ts           # listRoles, createRole, updateRole
    │   ├── tenant.ts         # createTenant, listTenants, getTenant, approve, freeze
    │   ├── plan.ts           # listPlans, createPlan, updatePlan
    │   ├── category.ts       # createCategory, updateCategory, listCategories
    │   ├── brand.ts          # createBrand, updateBrand, listBrands
    │   ├── order.ts          # listOrders, getOrder
    │   ├── payment.ts        # getPayment, getRefund
    │   ├── notification.ts   # template CRUD + sendNotification
    │   ├── inventory.ts      # getStock, batchStock, listLogs
    │   ├── marketing.ts      # listCoupons, listSeckill, getSeckill, listPromotions
    │   └── logistics.ts      # listFreightTemplates, getFreightTemplate, getShipment, getOrderLogistics
    ├── components/
    │   ├── AuthGuard.tsx
    │   ├── ErrorBoundary.tsx
    │   └── layout/
    │       └── MainLayout.tsx
    ├── pages/
    │   ├── login/index.tsx
    │   ├── dashboard/index.tsx
    │   ├── user/UserList.tsx
    │   ├── role/RoleList.tsx
    │   ├── tenant/TenantList.tsx
    │   ├── tenant/TenantDetail.tsx
    │   ├── plan/PlanList.tsx
    │   ├── category/CategoryList.tsx
    │   ├── brand/BrandList.tsx
    │   ├── order/OrderList.tsx
    │   ├── order/OrderDetail.tsx
    │   ├── payment/PaymentDetail.tsx
    │   ├── payment/RefundDetail.tsx
    │   ├── notification/TemplateList.tsx
    │   ├── notification/TemplateForm.tsx
    │   ├── notification/SendNotification.tsx
    │   ├── inventory/StockQuery.tsx
    │   ├── inventory/StockLog.tsx
    │   ├── marketing/CouponList.tsx
    │   ├── marketing/SeckillList.tsx
    │   ├── marketing/SeckillDetail.tsx
    │   ├── marketing/PromotionList.tsx
    │   ├── logistics/FreightList.tsx
    │   ├── logistics/FreightDetail.tsx
    │   └── logistics/ShipmentDetail.tsx
    ├── router/index.tsx
    ├── stores/
    │   └── auth.ts
    ├── types/
    │   ├── api.ts
    │   ├── user.ts
    │   ├── tenant.ts
    │   ├── product.ts
    │   ├── order.ts
    │   ├── payment.ts
    │   ├── notification.ts
    │   ├── inventory.ts
    │   ├── marketing.ts
    │   └── logistics.ts
    ├── constants/
    │   ├── index.ts          # formatPrice, parsePriceToFen
    │   ├── order.ts
    │   ├── marketing.ts
    │   └── payment.ts
    └── utils/
        └── error.ts
```

## Sidebar Menu Structure

```
Platform Admin
├── Dashboard              /dashboard
├── User Management
│   ├── User List          /users
│   └── Role Management    /roles
├── Tenant Management
│   ├── Tenant List        /tenants
│   └── Plan Management    /plans
├── Product Management
│   ├── Category Mgmt      /categories
│   └── Brand Management   /brands
├── Order Supervision
│   └── Order List         /orders
├── Notification Mgmt
│   ├── Template List      /notification-templates
│   └── Send Notification  /notifications/send
├── Inventory Supervision
│   ├── Stock Query        /inventory
│   └── Stock Logs         /inventory/logs
├── Marketing Supervision
│   ├── Coupons            /coupons
│   ├── Seckill            /seckill
│   └── Promotions         /promotions
└── Logistics Supervision
    └── Freight Templates  /freight-templates
```

Hidden routes (accessible via navigation, not in menu):
- `/tenants/:id` - Tenant detail
- `/orders/:orderNo` - Order detail (includes payment/refund/logistics info)
- `/notification-templates/create` - Create template
- `/notification-templates/:id/edit` - Edit template
- `/seckill/:id` - Seckill detail
- `/freight-templates/:id` - Freight template detail
- `/payments/:paymentNo` - Payment detail
- `/refunds/:refundNo` - Refund detail

## Page Designs

### Dashboard
- 4 stat cards: total tenants, total users, total orders, today's transaction volume
- Quick action shortcuts to major modules
- Aggregates data from multiple list APIs

### User Management
- **UserList**: ProTable with tenant filter, status filter, keyword search. Inline Switch for enable/disable user status.
- **RoleList**: ProTable with tenant filter. Create/Edit via Modal form (name, description, permissions).

### Tenant Management
- **TenantList**: ProTable with status filter, pagination. Action buttons: View Detail, Approve/Reject, Freeze/Unfreeze (with confirmation dialogs).
- **TenantDetail**: Descriptions component showing full tenant info.
- **PlanList**: ProTable listing subscription plans. Create/Edit via Modal form.

### Product Management
- **CategoryList**: Table/Tree displaying categories. Create/Edit via Modal form (name, parent_id, sort_order).
- **BrandList**: ProTable with pagination. Create/Edit via Modal form (name, logo, description).

### Order Supervision (Read-only)
- **OrderList**: ProTable with tenant filter, status filter, pagination. Click to view detail.
- **OrderDetail**: Descriptions with order info + payment info (via getPayment) + refund info (via getRefund) + logistics tracking (via getOrderLogistics). All read-only supervision.

### Notification Management
- **TemplateList**: ProTable with tenant and channel filters. Actions: Edit, Delete.
- **TemplateForm**: Full-page form for create/edit template (code, title, content, channel, params).
- **SendNotification**: Form to send notification - select template, channel, enter user_id and params.

### Inventory Supervision (Read-only)
- **StockQuery**: Input SKU ID to query stock, or batch query multiple SKUs.
- **StockLog**: ProTable with tenant and SKU filters, showing inventory change history.

### Marketing Supervision (Read-only)
- **CouponList**: ProTable with tenant/status filters.
- **SeckillList**: ProTable with tenant/status filters. Click for detail.
- **SeckillDetail**: Descriptions showing seckill activity details.
- **PromotionList**: ProTable with tenant/status filters.

### Logistics Supervision (Read-only)
- **FreightList**: ProTable with tenant filter. Click for detail.
- **FreightDetail**: Descriptions showing freight template configuration.
- **ShipmentDetail**: Accessible from order detail, shows shipment tracking.

## Authentication

### Login
- Simple form: phone + password (no tenantId field, platform admin has tenant_id=0)
- POST `/api/v1/login` with `{ phone, password }`
- Tokens from response headers: `x-jwt-token`, `x-refresh-token`
- Store in localStorage

### Token Management
- Identical to merchant-frontend pattern
- Axios request interceptor: adds `Authorization: Bearer {token}`
- Response interceptor: detects 401, queues requests, refreshes token, retries
- Logout: POST `/api/v1/logout`, clear localStorage, redirect to `/login`

### Route Protection
- AuthGuard component checks auth store
- Redirects to `/login?redirect={path}` if unauthenticated

## API Response Format

All APIs follow the unified format:
```json
{
  "code": 0,
  "msg": "success",
  "data": { ... }
}
```

Paginated responses:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [...],
    "total": 100
  }
}
```

## Key Differences from Merchant Frontend

| Aspect | Merchant Frontend | Admin Frontend |
|--------|------------------|----------------|
| Dev Port | 3001 | 3002 |
| BFF Port | 8180 | 8280 |
| Login | Requires tenantId | No tenantId (admin=0) |
| Role | Merchant staff | Platform admin |
| CRUD scope | Own tenant data | Cross-tenant supervision |
| Supervision pages | N/A | Orders, inventory, marketing, logistics |
| Tenant management | N/A | Approve/freeze tenants, manage plans |

## Implementation Notes

- Follow merchant-frontend coding patterns exactly (ProTable, Modal forms, Descriptions, etc.)
- All pages use lazy loading with React.lazy + Suspense
- Chinese locale for Ant Design (`zh_CN`)
- Error handling: global message.error() + ErrorBoundary
- Price values stored in fen (分), displayed in yuan with formatPrice utility
