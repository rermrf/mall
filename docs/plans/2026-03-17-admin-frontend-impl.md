# Admin Frontend Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a full platform administration frontend at `/admin-frontend/` covering all 54+ admin-bff API endpoints with React 19, Ant Design Pro, Zustand, and Vite.

**Architecture:** Independent React SPA reusing merchant-frontend patterns. Connects to admin-bff on port 8280 via Vite dev proxy. Sidebar layout with ProLayout, lazy-loaded pages, Axios with automatic token refresh.

**Tech Stack:** React 19, Vite 7, TypeScript 5.9, Ant Design 5 + Pro Components 2, Zustand 5, Axios, React Router 7, dayjs.

**Reference:** All code patterns taken from `/merchant-frontend/src/`. Admin-bff handlers at `/admin-bff/handler/`.

---

### Task 1: Scaffold project with Vite + React + TypeScript

**Files:**
- Create: `admin-frontend/package.json`
- Create: `admin-frontend/vite.config.ts`
- Create: `admin-frontend/tsconfig.json`
- Create: `admin-frontend/tsconfig.app.json`
- Create: `admin-frontend/index.html`
- Create: `admin-frontend/src/vite-env.d.ts`

**Step 1: Create package.json**

```json
{
  "name": "mall-admin",
  "private": true,
  "version": "0.0.1",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "@ant-design/icons": "^5.6.1",
    "@ant-design/pro-components": "^2.8.7",
    "antd": "^5.25.1",
    "axios": "^1.13.6",
    "dayjs": "^1.11.13",
    "react": "^19.2.4",
    "react-dom": "^19.2.4",
    "react-router-dom": "^7.13.1",
    "zustand": "^5.0.11"
  },
  "devDependencies": {
    "@types/node": "^25.4.0",
    "@types/react": "^19.2.14",
    "@types/react-dom": "^19.2.3",
    "@vitejs/plugin-react": "^5.1.4",
    "typescript": "^5.9.3",
    "vite": "^7.3.1"
  }
}
```

**Step 2: Create vite.config.ts**

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3002,
    proxy: {
      '/api': {
        target: 'http://localhost:8280',
        changeOrigin: true,
      },
    },
  },
})
```

**Step 3: Create tsconfig.json**

```json
{
  "files": [],
  "references": [
    { "path": "./tsconfig.app.json" }
  ]
}
```

**Step 4: Create tsconfig.app.json**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": false,
    "noUnusedParameters": false,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src"]
}
```

**Step 5: Create index.html**

```html
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>平台管理后台</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

**Step 6: Create src/vite-env.d.ts**

```ts
/// <reference types="vite/client" />
```

**Step 7: Install dependencies**

Run: `cd admin-frontend && npm install`
Expected: node_modules created, lock file generated

**Step 8: Commit**

```bash
git add admin-frontend/package.json admin-frontend/vite.config.ts admin-frontend/tsconfig.json admin-frontend/tsconfig.app.json admin-frontend/index.html admin-frontend/src/vite-env.d.ts admin-frontend/package-lock.json
git commit -m "feat(admin-frontend): scaffold project with Vite + React + TypeScript"
```

---

### Task 2: Core infrastructure — types, utils, constants, stores, API client

**Files:**
- Create: `admin-frontend/src/types/api.ts`
- Create: `admin-frontend/src/utils/error.ts`
- Create: `admin-frontend/src/constants/index.ts`
- Create: `admin-frontend/src/constants/order.ts`
- Create: `admin-frontend/src/constants/payment.ts`
- Create: `admin-frontend/src/constants/marketing.ts`
- Create: `admin-frontend/src/stores/auth.ts`
- Create: `admin-frontend/src/api/client.ts`
- Create: `admin-frontend/src/api/auth.ts`

**Step 1: Create src/types/api.ts**

```ts
export interface ApiResult<T = unknown> {
  code: number
  msg: string
  data: T
}

export interface PageResult<T> {
  list: T[]
  total: number
}

export interface PageParams {
  page?: number
  pageSize?: number
}
```

**Step 2: Create src/utils/error.ts**

```ts
export function handleApiError(context: string) {
  return (err: unknown) => {
    console.error(`[${context}]`, err)
    if (err instanceof Error && err.message === 'canceled') return
  }
}

export function silentApiError(context: string) {
  return (err: unknown) => {
    console.error(`[${context}]`, err)
  }
}
```

**Step 3: Create src/constants/index.ts**

```ts
export * from './order'
export * from './payment'
export * from './marketing'

export function formatPrice(fen: number): string {
  return `¥${((fen ?? 0) / 100).toFixed(2)}`
}

export function parsePriceToFen(yuan: number): number {
  return Math.round(yuan * 100)
}
```

**Step 4: Create src/constants/order.ts**

```ts
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

**Step 5: Create src/constants/payment.ts**

```ts
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

**Step 6: Create src/constants/marketing.ts**

```ts
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

**Step 7: Create src/stores/auth.ts**

```ts
import { create } from 'zustand'

interface AuthState {
  isLoggedIn: boolean
  checkAuth: () => void
  setLoggedIn: (v: boolean) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  isLoggedIn: !!localStorage.getItem('access_token'),
  checkAuth: () => {
    set({ isLoggedIn: !!localStorage.getItem('access_token') })
  },
  setLoggedIn: (v: boolean) => set({ isLoggedIn: v }),
  clearAuth: () => {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    set({ isLoggedIn: false })
  },
}))
```

**Step 8: Create src/api/client.ts**

```ts
import axios from 'axios'
import { message } from 'antd'
import type { ApiResult } from '@/types/api'

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let isRefreshing = false
let pendingRequests: Array<(token: string) => void> = []

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        return new Promise((resolve) => {
          pendingRequests.push((token: string) => {
            originalRequest.headers.Authorization = `Bearer ${token}`
            resolve(client(originalRequest))
          })
        })
      }
      originalRequest._retry = true
      isRefreshing = true
      try {
        const refreshToken = localStorage.getItem('refresh_token')
        if (!refreshToken) throw new Error('No refresh token')
        const res = await axios.post('/api/v1/refresh-token', {}, {
          headers: { Authorization: `Bearer ${refreshToken}` },
        })
        const newAccessToken = res.headers['x-jwt-token']
        const newRefreshToken = res.headers['x-refresh-token']
        if (newAccessToken) {
          localStorage.setItem('access_token', newAccessToken)
          if (newRefreshToken) localStorage.setItem('refresh_token', newRefreshToken)
          pendingRequests.forEach((cb) => cb(newAccessToken))
          pendingRequests = []
          originalRequest.headers.Authorization = `Bearer ${newAccessToken}`
          return client(originalRequest)
        }
        throw new Error('No token in refresh response')
      } catch {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        window.location.href = '/login'
        return Promise.reject(error)
      } finally {
        isRefreshing = false
      }
    }
    const msg = error.response?.data?.msg || error.message || '网络错误'
    if (error.response?.status !== 401) {
      message.error(msg)
    }
    return Promise.reject(error)
  },
)

export async function request<T>(config: Parameters<typeof client.request>[0]): Promise<T> {
  const res = await client.request(config)
  const body = res.data as ApiResult<T>
  if (body.code !== 0) {
    message.error(body.msg || '请求失败')
    throw new Error(body.msg || '请求失败')
  }
  return body.data
}

export { client }
export default client
```

**Step 9: Create src/api/auth.ts**

Admin login does NOT send tenantId (admin-bff hardcodes tenant_id=0). See `/admin-bff/handler/jwt/handler.go:52`.

```ts
import { client } from './client'

export interface LoginParams {
  phone: string
  password: string
}

function extractTokens(headers: Record<string, string>) {
  const accessToken = headers['x-jwt-token']
  const refreshToken = headers['x-refresh-token']
  if (accessToken) localStorage.setItem('access_token', accessToken)
  if (refreshToken) localStorage.setItem('refresh_token', refreshToken)
}

export async function login(params: LoginParams) {
  const res = await client.post('/login', params)
  const body = res.data
  if (body.code !== 0) {
    throw new Error(body.msg || '登录失败')
  }
  extractTokens(res.headers as Record<string, string>)
  return body.data
}

export async function logout() {
  await client.post('/logout', {})
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}
```

**Step 10: Commit**

```bash
git add admin-frontend/src/types/ admin-frontend/src/utils/ admin-frontend/src/constants/ admin-frontend/src/stores/ admin-frontend/src/api/client.ts admin-frontend/src/api/auth.ts
git commit -m "feat(admin-frontend): add core infrastructure — types, utils, stores, API client"
```

---

### Task 3: All domain API modules

**Files:**
- Create: `admin-frontend/src/types/user.ts`
- Create: `admin-frontend/src/types/tenant.ts`
- Create: `admin-frontend/src/types/product.ts`
- Create: `admin-frontend/src/types/order.ts`
- Create: `admin-frontend/src/types/payment.ts`
- Create: `admin-frontend/src/types/notification.ts`
- Create: `admin-frontend/src/types/inventory.ts`
- Create: `admin-frontend/src/types/marketing.ts`
- Create: `admin-frontend/src/types/logistics.ts`
- Create: `admin-frontend/src/api/user.ts`
- Create: `admin-frontend/src/api/role.ts`
- Create: `admin-frontend/src/api/tenant.ts`
- Create: `admin-frontend/src/api/plan.ts`
- Create: `admin-frontend/src/api/category.ts`
- Create: `admin-frontend/src/api/brand.ts`
- Create: `admin-frontend/src/api/order.ts`
- Create: `admin-frontend/src/api/payment.ts`
- Create: `admin-frontend/src/api/notification.ts`
- Create: `admin-frontend/src/api/inventory.ts`
- Create: `admin-frontend/src/api/marketing.ts`
- Create: `admin-frontend/src/api/logistics.ts`

**Step 1: Create type definitions**

`src/types/user.ts`:
```ts
export interface User {
  id: number
  phone: string
  nickname: string
  avatar: string
  email: string
  role: string
  status: number
  tenantId: number
  createdAt: string
}

export interface Role {
  id: number
  tenantId: number
  name: string
  code: string
  description: string
}
```

`src/types/tenant.ts`:
```ts
export interface Tenant {
  id: number
  name: string
  contactName: string
  contactPhone: string
  businessLicense: string
  planId: number
  status: number
  createdAt: string
  updatedAt: string
}

export interface TenantPlan {
  id: number
  name: string
  price: number
  durationDays: number
  maxProducts: number
  maxStaff: number
  features: string
}
```

`src/types/product.ts`:
```ts
export interface Category {
  id: number
  parentId: number
  name: string
  level: number
  sort: number
  icon: string
  status: number
}

export interface Brand {
  id: number
  name: string
  logo: string
  status: number
}
```

`src/types/order.ts`:
```ts
export interface Order {
  id: number
  orderNo: string
  tenantId: number
  userId: number
  totalAmount: number
  payAmount: number
  status: number
  receiverName: string
  receiverPhone: string
  receiverAddress: string
  createdAt: string
  updatedAt: string
  items: OrderItem[]
}

export interface OrderItem {
  id: number
  productId: number
  skuId: number
  title: string
  image: string
  price: number
  quantity: number
}
```

`src/types/payment.ts`:
```ts
export interface Payment {
  id: number
  paymentNo: string
  orderNo: string
  amount: number
  status: number
  channel: string
  paidAt: string
  createdAt: string
}

export interface Refund {
  id: number
  refundNo: string
  orderNo: string
  paymentNo: string
  amount: number
  status: number
  reason: string
  createdAt: string
}
```

`src/types/notification.ts`:
```ts
export interface NotificationTemplate {
  id: number
  tenantId: number
  code: string
  channel: number
  title: string
  content: string
  status: number
  createdAt: string
}
```

`src/types/inventory.ts`:
```ts
export interface Inventory {
  skuId: number
  tenantId: number
  stock: number
  locked: number
  available: number
}

export interface InventoryLog {
  id: number
  skuId: number
  tenantId: number
  change: number
  type: string
  orderNo: string
  createdAt: string
}
```

`src/types/marketing.ts`:
```ts
export interface Coupon {
  id: number
  tenantId: number
  name: string
  type: number
  value: number
  minAmount: number
  scope: number
  status: number
  totalCount: number
  usedCount: number
  startTime: string
  endTime: string
}

export interface SeckillActivity {
  id: number
  tenantId: number
  title: string
  startTime: string
  endTime: string
  status: number
  items: SeckillItem[]
}

export interface SeckillItem {
  id: number
  productId: number
  skuId: number
  seckillPrice: number
  stock: number
  limit: number
}

export interface PromotionRule {
  id: number
  tenantId: number
  name: string
  type: number
  threshold: number
  discount: number
  status: number
  startTime: string
  endTime: string
}
```

`src/types/logistics.ts`:
```ts
export interface FreightTemplate {
  id: number
  tenantId: number
  name: string
  chargeType: number
  regions: FreightRegion[]
}

export interface FreightRegion {
  region: string
  firstWeight: number
  firstFee: number
  continueWeight: number
  continueFee: number
}

export interface Shipment {
  id: number
  orderId: number
  orderNo: string
  company: string
  trackingNo: string
  status: number
  createdAt: string
}
```

**Step 2: Create API modules**

All APIs follow the pattern from `admin-bff/handler/*.go`. Each module uses the `request<T>()` helper from `client.ts`.

`src/api/user.ts`:
```ts
import { request } from './client'
import type { User } from '@/types/user'

export interface ListUsersParams {
  tenantId?: number
  page?: number
  pageSize?: number
  status?: number
  keyword?: string
}

export async function listUsers(params: ListUsersParams) {
  return request<{ users: User[]; total: number }>({
    method: 'GET',
    url: '/users',
    params: { tenant_id: params.tenantId, page: params.page, page_size: params.pageSize, status: params.status, keyword: params.keyword },
  })
}

export async function updateUserStatus(id: number, status: number) {
  return request<null>({
    method: 'POST',
    url: `/users/${id}/status`,
    data: { status },
  })
}
```

`src/api/role.ts`:
```ts
import { request } from './client'
import type { Role } from '@/types/user'

export async function listRoles(tenantId?: number) {
  return request<Role[]>({
    method: 'GET',
    url: '/roles',
    params: { tenant_id: tenantId },
  })
}

export async function createRole(data: { tenantId?: number; name: string; code: string; description: string }) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/roles',
    data: { tenant_id: data.tenantId, name: data.name, code: data.code, description: data.description },
  })
}

export async function updateRole(id: number, data: { name: string; code: string; description: string }) {
  return request<null>({
    method: 'PUT',
    url: `/roles/${id}`,
    data,
  })
}
```

`src/api/tenant.ts`:
```ts
import { request } from './client'
import type { Tenant } from '@/types/tenant'

export interface ListTenantsParams {
  page?: number
  pageSize?: number
  status?: number
}

export async function listTenants(params: ListTenantsParams) {
  return request<{ tenants: Tenant[]; total: number }>({
    method: 'GET',
    url: '/tenants',
    params: { page: params.page, page_size: params.pageSize, status: params.status },
  })
}

export async function getTenant(id: number) {
  return request<Tenant>({
    method: 'GET',
    url: `/tenants/${id}`,
  })
}

export async function createTenant(data: { name: string; contactName: string; contactPhone: string; businessLicense: string; planId: number }) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/tenants',
    data: { name: data.name, contact_name: data.contactName, contact_phone: data.contactPhone, business_license: data.businessLicense, plan_id: data.planId },
  })
}

export async function approveTenant(id: number, approved: boolean, reason?: string) {
  return request<null>({
    method: 'POST',
    url: `/tenants/${id}/approve`,
    data: { approved, reason },
  })
}

export async function freezeTenant(id: number, freeze: boolean) {
  return request<null>({
    method: 'POST',
    url: `/tenants/${id}/freeze`,
    data: { freeze },
  })
}
```

`src/api/plan.ts`:
```ts
import { request } from './client'
import type { TenantPlan } from '@/types/tenant'

export async function listPlans() {
  return request<{ plans: TenantPlan[] }>({
    method: 'GET',
    url: '/plans',
  })
}

export interface PlanData {
  name: string
  price: number
  durationDays: number
  maxProducts: number
  maxStaff: number
  features: string
}

export async function createPlan(data: PlanData) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/plans',
    data: { name: data.name, price: data.price, duration_days: data.durationDays, max_products: data.maxProducts, max_staff: data.maxStaff, features: data.features },
  })
}

export async function updatePlan(id: number, data: PlanData) {
  return request<null>({
    method: 'PUT',
    url: `/plans/${id}`,
    data: { name: data.name, price: data.price, duration_days: data.durationDays, max_products: data.maxProducts, max_staff: data.maxStaff, features: data.features },
  })
}
```

`src/api/category.ts`:
```ts
import { request } from './client'
import type { Category } from '@/types/product'

export async function listCategories() {
  return request<Category[]>({
    method: 'GET',
    url: '/categories',
  })
}

export interface CategoryData {
  parentId?: number
  name: string
  level?: number
  sort?: number
  icon?: string
  status?: number
}

export async function createCategory(data: CategoryData) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/categories',
    data: { parent_id: data.parentId, name: data.name, level: data.level, sort: data.sort, icon: data.icon, status: data.status },
  })
}

export async function updateCategory(id: number, data: CategoryData) {
  return request<null>({
    method: 'PUT',
    url: `/categories/${id}`,
    data: { parent_id: data.parentId, name: data.name, level: data.level, sort: data.sort, icon: data.icon, status: data.status },
  })
}
```

`src/api/brand.ts`:
```ts
import { request } from './client'
import type { Brand } from '@/types/product'

export interface ListBrandsParams {
  page: number
  pageSize: number
}

export async function listBrands(params: ListBrandsParams) {
  return request<{ brands: Brand[]; total: number }>({
    method: 'GET',
    url: '/brands',
    params: { page: params.page, page_size: params.pageSize },
  })
}

export async function createBrand(data: { name: string; logo?: string; status?: number }) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/brands',
    data,
  })
}

export async function updateBrand(id: number, data: { name: string; logo?: string; status?: number }) {
  return request<null>({
    method: 'PUT',
    url: `/brands/${id}`,
    data,
  })
}
```

`src/api/order.ts`:
```ts
import { request } from './client'
import type { Order } from '@/types/order'

export interface ListOrdersParams {
  tenantId?: number
  status?: number
  page: number
  pageSize: number
}

export async function listOrders(params: ListOrdersParams) {
  return request<{ orders: Order[]; total: number }>({
    method: 'GET',
    url: '/orders',
    params: { tenant_id: params.tenantId, status: params.status, page: params.page, page_size: params.pageSize },
  })
}

export async function getOrder(orderNo: string) {
  return request<Order>({
    method: 'GET',
    url: `/orders/${orderNo}`,
  })
}
```

`src/api/payment.ts`:
```ts
import { request } from './client'
import type { Payment, Refund } from '@/types/payment'

export async function getPayment(paymentNo: string) {
  return request<Payment>({
    method: 'GET',
    url: `/payments/${paymentNo}`,
  })
}

export async function getRefund(refundNo: string) {
  return request<Refund>({
    method: 'GET',
    url: `/refunds/${refundNo}`,
  })
}
```

`src/api/notification.ts`:
```ts
import { request } from './client'
import type { NotificationTemplate } from '@/types/notification'

export interface ListTemplatesParams {
  tenantId?: number
  channel?: number
}

export async function listTemplates(params: ListTemplatesParams) {
  return request<NotificationTemplate[]>({
    method: 'GET',
    url: '/notification-templates',
    params: { tenant_id: params.tenantId, channel: params.channel },
  })
}

export interface TemplateData {
  tenantId?: number
  code: string
  channel: number
  title: string
  content: string
  status?: number
}

export async function createTemplate(data: TemplateData) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/notification-templates',
    data: { tenant_id: data.tenantId, code: data.code, channel: data.channel, title: data.title, content: data.content, status: data.status },
  })
}

export async function updateTemplate(id: number, data: TemplateData) {
  return request<null>({
    method: 'PUT',
    url: `/notification-templates/${id}`,
    data: { tenant_id: data.tenantId, code: data.code, channel: data.channel, title: data.title, content: data.content, status: data.status },
  })
}

export async function deleteTemplate(id: number) {
  return request<null>({
    method: 'DELETE',
    url: `/notification-templates/${id}`,
  })
}

export interface SendNotificationData {
  userId: number
  tenantId?: number
  templateCode: string
  channel: number
  params?: Record<string, string>
}

export async function sendNotification(data: SendNotificationData) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/notifications/send',
    data: { user_id: data.userId, tenant_id: data.tenantId, template_code: data.templateCode, channel: data.channel, params: data.params },
  })
}
```

`src/api/inventory.ts`:
```ts
import { request } from './client'
import type { Inventory, InventoryLog } from '@/types/inventory'

export async function getStock(skuId: number) {
  return request<Inventory>({
    method: 'GET',
    url: `/inventory/${skuId}`,
  })
}

export async function batchGetStock(skuIds: number[]) {
  return request<Inventory[]>({
    method: 'POST',
    url: '/inventory/batch',
    data: { sku_ids: skuIds },
  })
}

export interface ListLogsParams {
  tenantId?: number
  skuId?: number
  page: number
  pageSize: number
}

export async function listLogs(params: ListLogsParams) {
  return request<{ logs: InventoryLog[]; total: number }>({
    method: 'GET',
    url: '/inventory/logs',
    params: { tenant_id: params.tenantId, sku_id: params.skuId, page: params.page, page_size: params.pageSize },
  })
}
```

`src/api/marketing.ts`:
```ts
import { request } from './client'
import type { Coupon, SeckillActivity, PromotionRule } from '@/types/marketing'

export interface ListCouponsParams {
  tenantId?: number
  status?: number
  page: number
  pageSize: number
}

export async function listCoupons(params: ListCouponsParams) {
  return request<{ coupons: Coupon[]; total: number }>({
    method: 'GET',
    url: '/coupons',
    params: { tenant_id: params.tenantId, status: params.status, page: params.page, page_size: params.pageSize },
  })
}

export interface ListSeckillParams {
  tenantId?: number
  status?: number
  page: number
  pageSize: number
}

export async function listSeckill(params: ListSeckillParams) {
  return request<{ activities: SeckillActivity[]; total: number }>({
    method: 'GET',
    url: '/seckill',
    params: { tenant_id: params.tenantId, status: params.status, page: params.page, page_size: params.pageSize },
  })
}

export async function getSeckill(id: number) {
  return request<SeckillActivity>({
    method: 'GET',
    url: `/seckill/${id}`,
  })
}

export async function listPromotions(params: { tenantId?: number; status?: number }) {
  return request<PromotionRule[]>({
    method: 'GET',
    url: '/promotions',
    params: { tenant_id: params.tenantId, status: params.status },
  })
}
```

`src/api/logistics.ts`:
```ts
import { request } from './client'
import type { FreightTemplate, Shipment } from '@/types/logistics'

export async function listFreightTemplates(tenantId?: number) {
  return request<FreightTemplate[]>({
    method: 'GET',
    url: '/freight-templates',
    params: { tenant_id: tenantId },
  })
}

export async function getFreightTemplate(id: number) {
  return request<FreightTemplate>({
    method: 'GET',
    url: `/freight-templates/${id}`,
  })
}

export async function getShipment(id: number) {
  return request<Shipment>({
    method: 'GET',
    url: `/shipments/${id}`,
  })
}

export async function getOrderLogistics(orderNo: string) {
  return request<Shipment>({
    method: 'GET',
    url: `/orders/${orderNo}/logistics`,
  })
}
```

**Step 3: Commit**

```bash
git add admin-frontend/src/types/ admin-frontend/src/api/
git commit -m "feat(admin-frontend): add all domain types and API modules"
```

---

### Task 4: Components — AuthGuard, ErrorBoundary, MainLayout

**Files:**
- Create: `admin-frontend/src/components/AuthGuard.tsx`
- Create: `admin-frontend/src/components/ErrorBoundary.tsx`
- Create: `admin-frontend/src/components/layout/MainLayout.tsx`

**Step 1: Create AuthGuard**

```tsx
import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'

export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const location = useLocation()
  if (!isLoggedIn) {
    return <Navigate to={`/login?redirect=${encodeURIComponent(location.pathname)}`} replace />
  }
  return <>{children}</>
}
```

**Step 2: Create ErrorBoundary**

```tsx
import { Component, type ReactNode } from 'react'
import { Result, Button } from 'antd'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return (
        <Result
          status="error"
          title="页面出错了"
          subTitle={this.state.error?.message || '发生了未知错误'}
          extra={
            <Button
              type="primary"
              onClick={() => this.setState({ hasError: false, error: null })}
            >
              重试
            </Button>
          }
        />
      )
    }
    return this.props.children
  }
}
```

**Step 3: Create MainLayout**

The admin layout differs from merchant: different menu structure, different title/logo, no notification store (admin doesn't receive notifications), no profile fetch.

```tsx
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { ProLayout } from '@ant-design/pro-components'
import { Dropdown, Avatar, message } from 'antd'
import {
  DashboardOutlined,
  TeamOutlined,
  ShopOutlined,
  AppstoreOutlined,
  TagsOutlined,
  OrderedListOutlined,
  BellOutlined,
  InboxOutlined,
  GiftOutlined,
  CarOutlined,
  LogoutOutlined,
  UserOutlined,
  CrownOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '@/stores/auth'
import { logout } from '@/api/auth'

const menuRoutes = {
  path: '/',
  routes: [
    { path: '/dashboard', name: '仪表盘', icon: <DashboardOutlined /> },
    {
      path: '/user-mgmt',
      name: '用户管理',
      icon: <TeamOutlined />,
      routes: [
        { path: '/users', name: '用户列表' },
        { path: '/roles', name: '角色管理' },
      ],
    },
    {
      path: '/tenant-mgmt',
      name: '租户管理',
      icon: <SafetyCertificateOutlined />,
      routes: [
        { path: '/tenants', name: '租户列表' },
        { path: '/plans', name: '套餐管理' },
      ],
    },
    {
      path: '/product-mgmt',
      name: '商品管理',
      icon: <AppstoreOutlined />,
      routes: [
        { path: '/categories', name: '分类管理' },
        { path: '/brands', name: '品牌管理' },
      ],
    },
    {
      path: '/order-supervision',
      name: '订单监管',
      icon: <OrderedListOutlined />,
      routes: [
        { path: '/orders', name: '订单列表' },
      ],
    },
    {
      path: '/notification-mgmt',
      name: '通知管理',
      icon: <BellOutlined />,
      routes: [
        { path: '/notification-templates', name: '模板列表' },
        { path: '/notifications/send', name: '发送通知' },
      ],
    },
    {
      path: '/inventory-supervision',
      name: '库存监管',
      icon: <InboxOutlined />,
      routes: [
        { path: '/inventory', name: '库存查询' },
        { path: '/inventory/logs', name: '库存日志' },
      ],
    },
    {
      path: '/marketing-supervision',
      name: '营销监管',
      icon: <GiftOutlined />,
      routes: [
        { path: '/coupons', name: '优惠券' },
        { path: '/seckill', name: '秒杀活动' },
        { path: '/promotions', name: '促销规则' },
      ],
    },
    {
      path: '/logistics-supervision',
      name: '物流监管',
      icon: <CarOutlined />,
      routes: [
        { path: '/freight-templates', name: '运费模板' },
      ],
    },
  ],
}

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const clearAuth = useAuthStore((s) => s.clearAuth)

  const handleLogout = async () => {
    try {
      await logout()
    } catch {
      // ignore
    }
    clearAuth()
    navigate('/login', { replace: true })
    message.success('已退出登录')
  }

  return (
    <ProLayout
      title="平台管理后台"
      logo={<CrownOutlined style={{ fontSize: 28, color: '#722ed1' }} />}
      route={menuRoutes}
      location={{ pathname: location.pathname }}
      menuItemRender={(item, dom) => (
        <span onClick={() => item.path && navigate(item.path)}>{dom}</span>
      )}
      actionsRender={() => [
        <Dropdown
          key="user"
          menu={{
            items: [
              { type: 'divider' as const },
              { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
            ],
          }}
        >
          <span style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 8 }}>
            <Avatar size="small" icon={<UserOutlined />} />
            <span>管理员</span>
          </span>
        </Dropdown>,
      ]}
      fixSiderbar
      layout="mix"
    >
      <Outlet />
    </ProLayout>
  )
}
```

**Step 4: Commit**

```bash
git add admin-frontend/src/components/
git commit -m "feat(admin-frontend): add AuthGuard, ErrorBoundary, and MainLayout"
```

---

### Task 5: Router, App entry, Login page, Dashboard

**Files:**
- Create: `admin-frontend/src/pages/login/index.tsx`
- Create: `admin-frontend/src/pages/dashboard/index.tsx`
- Create: `admin-frontend/src/router/index.tsx`
- Create: `admin-frontend/src/App.tsx`
- Create: `admin-frontend/src/main.tsx`

**Step 1: Create Login page**

No tenantId field — admin login only needs phone + password.

```tsx
import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Form, Input, Button, Card, message } from 'antd'
import { CrownOutlined, PhoneOutlined, LockOutlined } from '@ant-design/icons'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

export default function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const setLoggedIn = useAuthStore((s) => s.setLoggedIn)
  const [loading, setLoading] = useState(false)
  const redirect = searchParams.get('redirect') || '/dashboard'

  const onFinish = async (values: { phone: string; password: string }) => {
    setLoading(true)
    try {
      await login(values)
      setLoggedIn(true)
      message.success('登录成功')
      navigate(redirect, { replace: true })
    } catch (e: unknown) {
      message.error((e as Error).message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(135deg, #722ed1 0%, #2f54eb 100%)',
    }}>
      <Card style={{ width: 400, borderRadius: 8 }} bordered={false}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <CrownOutlined style={{ fontSize: 48, color: '#722ed1' }} />
          <h2 style={{ marginTop: 16, marginBottom: 4 }}>平台管理后台</h2>
          <p style={{ color: '#999' }}>登录平台管理员账户</p>
        </div>
        <Form onFinish={onFinish} size="large">
          <Form.Item name="phone" rules={[{ required: true, message: '请输入手机号' }]}>
            <Input prefix={<PhoneOutlined />} placeholder="手机号" maxLength={11} />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={loading}>
              登录
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}
```

**Step 2: Create Dashboard**

```tsx
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Col, Row, Statistic, List, Button, Typography } from 'antd'
import {
  SafetyCertificateOutlined,
  TeamOutlined,
  OrderedListOutlined,
  AppstoreOutlined,
} from '@ant-design/icons'
import { listTenants } from '@/api/tenant'
import { listUsers } from '@/api/user'
import { listOrders } from '@/api/order'
import { silentApiError } from '@/utils/error'

const { Title } = Typography

export default function Dashboard() {
  const navigate = useNavigate()
  const [tenantCount, setTenantCount] = useState(0)
  const [userCount, setUserCount] = useState(0)
  const [orderCount, setOrderCount] = useState(0)

  useEffect(() => {
    listTenants({ page: 1, pageSize: 1 }).then((r) => setTenantCount(r?.total ?? 0)).catch(silentApiError('dashboard:tenants'))
    listUsers({ page: 1, pageSize: 1 }).then((r) => setUserCount(r?.total ?? 0)).catch(silentApiError('dashboard:users'))
    listOrders({ page: 1, pageSize: 1 }).then((r) => setOrderCount(r?.total ?? 0)).catch(silentApiError('dashboard:orders'))
  }, [])

  const shortcuts = [
    { title: '租户管理', path: '/tenants' },
    { title: '用户管理', path: '/users' },
    { title: '订单监管', path: '/orders' },
    { title: '分类管理', path: '/categories' },
    { title: '通知管理', path: '/notification-templates' },
  ]

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>平台概览</Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/tenants')}>
            <Statistic title="租户总数" value={tenantCount} prefix={<SafetyCertificateOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/users')}>
            <Statistic title="用户总数" value={userCount} prefix={<TeamOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/orders')}>
            <Statistic title="订单总数" value={orderCount} prefix={<OrderedListOutlined />} />
          </Card>
        </Col>
      </Row>

      <Card title="快捷入口" style={{ marginTop: 24 }}>
        <List
          grid={{ gutter: 16, xs: 2, sm: 3, md: 5 }}
          dataSource={shortcuts}
          renderItem={(item) => (
            <List.Item>
              <Button block onClick={() => navigate(item.path)} icon={<AppstoreOutlined />}>
                {item.title}
              </Button>
            </List.Item>
          )}
        />
      </Card>
    </div>
  )
}
```

**Step 3: Create Router**

```tsx
import { lazy, Suspense } from 'react'
import { createBrowserRouter } from 'react-router-dom'
import { Spin } from 'antd'
import MainLayout from '@/components/layout/MainLayout'
import AuthGuard from '@/components/AuthGuard'
import ErrorBoundary from '@/components/ErrorBoundary'

const LoginPage = lazy(() => import('@/pages/login'))
const Dashboard = lazy(() => import('@/pages/dashboard'))
const UserList = lazy(() => import('@/pages/user/UserList'))
const RoleList = lazy(() => import('@/pages/role/RoleList'))
const TenantList = lazy(() => import('@/pages/tenant/TenantList'))
const TenantDetail = lazy(() => import('@/pages/tenant/TenantDetail'))
const PlanList = lazy(() => import('@/pages/plan/PlanList'))
const CategoryList = lazy(() => import('@/pages/category/CategoryList'))
const BrandList = lazy(() => import('@/pages/brand/BrandList'))
const OrderList = lazy(() => import('@/pages/order/OrderList'))
const OrderDetail = lazy(() => import('@/pages/order/OrderDetail'))
const TemplateList = lazy(() => import('@/pages/notification/TemplateList'))
const TemplateForm = lazy(() => import('@/pages/notification/TemplateForm'))
const SendNotification = lazy(() => import('@/pages/notification/SendNotification'))
const StockQuery = lazy(() => import('@/pages/inventory/StockQuery'))
const StockLog = lazy(() => import('@/pages/inventory/StockLog'))
const CouponList = lazy(() => import('@/pages/marketing/CouponList'))
const SeckillList = lazy(() => import('@/pages/marketing/SeckillList'))
const SeckillDetail = lazy(() => import('@/pages/marketing/SeckillDetail'))
const PromotionList = lazy(() => import('@/pages/marketing/PromotionList'))
const FreightList = lazy(() => import('@/pages/logistics/FreightList'))
const FreightDetail = lazy(() => import('@/pages/logistics/FreightDetail'))

function Loading() {
  return <div style={{ display: 'flex', justifyContent: 'center', padding: '20vh 0' }}><Spin size="large" /></div>
}

function L({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<Loading />}>{children}</Suspense>
}

export const router = createBrowserRouter([
  { path: '/login', element: <L><LoginPage /></L> },
  {
    element: <AuthGuard><MainLayout /></AuthGuard>,
    errorElement: <ErrorBoundary><div /></ErrorBoundary>,
    children: [
      { path: '/', element: <L><Dashboard /></L> },
      { path: '/dashboard', element: <L><Dashboard /></L> },
      { path: '/users', element: <L><UserList /></L> },
      { path: '/roles', element: <L><RoleList /></L> },
      { path: '/tenants', element: <L><TenantList /></L> },
      { path: '/tenants/:id', element: <L><TenantDetail /></L> },
      { path: '/plans', element: <L><PlanList /></L> },
      { path: '/categories', element: <L><CategoryList /></L> },
      { path: '/brands', element: <L><BrandList /></L> },
      { path: '/orders', element: <L><OrderList /></L> },
      { path: '/orders/:orderNo', element: <L><OrderDetail /></L> },
      { path: '/notification-templates', element: <L><TemplateList /></L> },
      { path: '/notification-templates/create', element: <L><TemplateForm /></L> },
      { path: '/notification-templates/:id/edit', element: <L><TemplateForm /></L> },
      { path: '/notifications/send', element: <L><SendNotification /></L> },
      { path: '/inventory', element: <L><StockQuery /></L> },
      { path: '/inventory/logs', element: <L><StockLog /></L> },
      { path: '/coupons', element: <L><CouponList /></L> },
      { path: '/seckill', element: <L><SeckillList /></L> },
      { path: '/seckill/:id', element: <L><SeckillDetail /></L> },
      { path: '/promotions', element: <L><PromotionList /></L> },
      { path: '/freight-templates', element: <L><FreightList /></L> },
      { path: '/freight-templates/:id', element: <L><FreightDetail /></L> },
    ],
  },
])
```

**Step 4: Create App.tsx**

```tsx
import { RouterProvider } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { router } from './router'
import ErrorBoundary from './components/ErrorBoundary'

export default function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <ErrorBoundary>
        <RouterProvider router={router} />
      </ErrorBoundary>
    </ConfigProvider>
  )
}
```

**Step 5: Create main.tsx**

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
```

**Step 6: Verify build**

Run: `cd admin-frontend && npx tsc --noEmit`
Expected: Should fail because page components don't exist yet. That's OK — we'll create placeholder pages next.

**Step 7: Commit**

```bash
git add admin-frontend/src/pages/login/ admin-frontend/src/pages/dashboard/ admin-frontend/src/router/ admin-frontend/src/App.tsx admin-frontend/src/main.tsx
git commit -m "feat(admin-frontend): add router, App entry, login page, and dashboard"
```

---

### Task 6: User Management pages — UserList, RoleList

**Files:**
- Create: `admin-frontend/src/pages/user/UserList.tsx`
- Create: `admin-frontend/src/pages/role/RoleList.tsx`

**Step 1: Create UserList**

ProTable with tenant filter, status filter, keyword search. Inline Switch for status toggle. Based on `GET /users` and `POST /users/:id/status`.

```tsx
import { useRef } from 'react'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Switch, message } from 'antd'
import { listUsers, updateUserStatus } from '@/api/user'
import type { User } from '@/types/user'

export default function UserList() {
  const actionRef = useRef<ActionType>()

  const handleStatusChange = async (id: number, checked: boolean) => {
    try {
      await updateUserStatus(id, checked ? 1 : 0)
      message.success('状态更新成功')
      actionRef.current?.reload()
    } catch { /* handled by interceptor */ }
  }

  const columns: ProColumns<User>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '手机号', dataIndex: 'phone', search: false },
    { title: '昵称', dataIndex: 'nickname', search: false },
    { title: '邮箱', dataIndex: 'email', search: false },
    { title: '角色', dataIndex: 'role', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: true,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '搜索手机号/昵称' },
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (_, record) => (
        <Switch
          checked={record.status === 1}
          onChange={(checked) => handleStatusChange(record.id, checked)}
          checkedChildren="启用"
          unCheckedChildren="禁用"
        />
      ),
      valueType: 'select',
      valueEnum: { 0: { text: '禁用' }, 1: { text: '启用' } },
    },
    { title: '注册时间', dataIndex: 'createdAt', search: false, valueType: 'dateTime' },
  ]

  return (
    <ProTable<User>
      headerTitle="用户列表"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listUsers({
          tenantId: params.tenantId,
          page: params.current,
          pageSize: params.pageSize,
          status: params.status !== undefined ? Number(params.status) : undefined,
          keyword: params.keyword,
        })
        return { data: res?.users ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
```

**Step 2: Create RoleList**

ProTable with tenant filter. Create/Edit via Modal.

```tsx
import { useRef, useState } from 'react'
import { ProTable, type ActionType, type ProColumns, ModalForm, ProFormText, ProFormDigit } from '@ant-design/pro-components'
import { Button, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { listRoles, createRole, updateRole } from '@/api/role'
import type { Role } from '@/types/user'

export default function RoleList() {
  const actionRef = useRef<ActionType>()
  const [modalOpen, setModalOpen] = useState(false)
  const [editingRole, setEditingRole] = useState<Role | null>(null)

  const columns: ProColumns<Role>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '角色名', dataIndex: 'name', search: false },
    { title: '角色编码', dataIndex: 'code', search: false },
    { title: '描述', dataIndex: 'description', search: false, ellipsis: true },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: true,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="edit" onClick={() => { setEditingRole(record); setModalOpen(true) }}>编辑</a>,
      ],
    },
  ]

  const handleSubmit = async (values: { name: string; code: string; description: string; tenantId?: number }) => {
    try {
      if (editingRole) {
        await updateRole(editingRole.id, { name: values.name, code: values.code, description: values.description })
        message.success('更新成功')
      } else {
        await createRole({ tenantId: values.tenantId, name: values.name, code: values.code, description: values.description })
        message.success('创建成功')
      }
      setModalOpen(false)
      setEditingRole(null)
      actionRef.current?.reload()
      return true
    } catch {
      return false
    }
  }

  return (
    <>
      <ProTable<Role>
        headerTitle="角色管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        request={async (params) => {
          const res = await listRoles(params.tenantId)
          return { data: res ?? [], total: res?.length ?? 0, success: true }
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => { setEditingRole(null); setModalOpen(true) }}>
            新建角色
          </Button>,
        ]}
        search={{ labelWidth: 'auto' }}
      />
      <ModalForm
        title={editingRole ? '编辑角色' : '新建角色'}
        open={modalOpen}
        onOpenChange={setModalOpen}
        onFinish={handleSubmit}
        initialValues={editingRole ? { name: editingRole.name, code: editingRole.code, description: editingRole.description } : {}}
        modalProps={{ destroyOnClose: true }}
      >
        {!editingRole && (
          <ProFormDigit name="tenantId" label="租户ID" placeholder="留空为平台角色" />
        )}
        <ProFormText name="name" label="角色名" rules={[{ required: true }]} />
        <ProFormText name="code" label="角色编码" rules={[{ required: true }]} />
        <ProFormText name="description" label="描述" />
      </ModalForm>
    </>
  )
}
```

**Step 3: Commit**

```bash
git add admin-frontend/src/pages/user/ admin-frontend/src/pages/role/
git commit -m "feat(admin-frontend): add UserList and RoleList pages"
```

---

### Task 7: Tenant Management pages — TenantList, TenantDetail, PlanList

**Files:**
- Create: `admin-frontend/src/pages/tenant/TenantList.tsx`
- Create: `admin-frontend/src/pages/tenant/TenantDetail.tsx`
- Create: `admin-frontend/src/pages/plan/PlanList.tsx`

**Step 1: Create TenantList**

```tsx
import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Button, Tag, Modal, Input, message } from 'antd'
import { listTenants, approveTenant, freezeTenant } from '@/api/tenant'
import type { Tenant } from '@/types/tenant'

const TENANT_STATUS_MAP: Record<number, { text: string; color: string }> = {
  0: { text: '待审核', color: 'orange' },
  1: { text: '正常', color: 'green' },
  2: { text: '已冻结', color: 'red' },
  3: { text: '已拒绝', color: 'default' },
}

export default function TenantList() {
  const actionRef = useRef<ActionType>()
  const navigate = useNavigate()

  const handleApprove = (id: number, approved: boolean) => {
    if (approved) {
      Modal.confirm({
        title: '审批通过',
        content: '确认通过该租户申请？',
        onOk: async () => {
          await approveTenant(id, true)
          message.success('已通过')
          actionRef.current?.reload()
        },
      })
    } else {
      let reason = ''
      Modal.confirm({
        title: '拒绝申请',
        content: <Input.TextArea placeholder="拒绝原因" onChange={(e) => { reason = e.target.value }} />,
        onOk: async () => {
          await approveTenant(id, false, reason)
          message.success('已拒绝')
          actionRef.current?.reload()
        },
      })
    }
  }

  const handleFreeze = (id: number, freeze: boolean) => {
    Modal.confirm({
      title: freeze ? '冻结租户' : '解冻租户',
      content: freeze ? '确认冻结该租户？' : '确认解冻该租户？',
      onOk: async () => {
        await freezeTenant(id, freeze)
        message.success(freeze ? '已冻结' : '已解冻')
        actionRef.current?.reload()
      },
    })
  }

  const columns: ProColumns<Tenant>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '租户名称', dataIndex: 'name', search: false },
    { title: '联系人', dataIndex: 'contactName', search: false },
    { title: '联系电话', dataIndex: 'contactPhone', search: false },
    {
      title: '状态',
      dataIndex: 'status',
      render: (_, record) => {
        const s = TENANT_STATUS_MAP[record.status] || { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
      valueType: 'select',
      valueEnum: { 0: { text: '待审核' }, 1: { text: '正常' }, 2: { text: '已冻结' }, 3: { text: '已拒绝' } },
    },
    { title: '创建时间', dataIndex: 'createdAt', search: false, valueType: 'dateTime' },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => {
        const actions: React.ReactNode[] = [
          <a key="detail" onClick={() => navigate(`/tenants/${record.id}`)}>详情</a>,
        ]
        if (record.status === 0) {
          actions.push(<a key="approve" onClick={() => handleApprove(record.id, true)}>通过</a>)
          actions.push(<a key="reject" style={{ color: '#ff4d4f' }} onClick={() => handleApprove(record.id, false)}>拒绝</a>)
        }
        if (record.status === 1) {
          actions.push(<a key="freeze" style={{ color: '#ff4d4f' }} onClick={() => handleFreeze(record.id, true)}>冻结</a>)
        }
        if (record.status === 2) {
          actions.push(<a key="unfreeze" onClick={() => handleFreeze(record.id, false)}>解冻</a>)
        }
        return actions
      },
    },
  ]

  return (
    <ProTable<Tenant>
      headerTitle="租户列表"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listTenants({
          page: params.current,
          pageSize: params.pageSize,
          status: params.status !== undefined ? Number(params.status) : undefined,
        })
        return { data: res?.tenants ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
```

**Step 2: Create TenantDetail**

```tsx
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Descriptions, Card, Spin, Tag } from 'antd'
import { getTenant } from '@/api/tenant'
import type { Tenant } from '@/types/tenant'

const TENANT_STATUS_MAP: Record<number, { text: string; color: string }> = {
  0: { text: '待审核', color: 'orange' },
  1: { text: '正常', color: 'green' },
  2: { text: '已冻结', color: 'red' },
  3: { text: '已拒绝', color: 'default' },
}

export default function TenantDetail() {
  const { id } = useParams<{ id: string }>()
  const [tenant, setTenant] = useState<Tenant | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    getTenant(Number(id)).then(setTenant).catch(() => {}).finally(() => setLoading(false))
  }, [id])

  if (loading) return <Spin style={{ display: 'block', margin: '20vh auto' }} size="large" />
  if (!tenant) return <div>租户不存在</div>

  const s = TENANT_STATUS_MAP[tenant.status] || { text: '未知', color: 'default' }

  return (
    <Card title="租户详情">
      <Descriptions column={2} bordered>
        <Descriptions.Item label="ID">{tenant.id}</Descriptions.Item>
        <Descriptions.Item label="名称">{tenant.name}</Descriptions.Item>
        <Descriptions.Item label="联系人">{tenant.contactName}</Descriptions.Item>
        <Descriptions.Item label="联系电话">{tenant.contactPhone}</Descriptions.Item>
        <Descriptions.Item label="营业执照">{tenant.businessLicense}</Descriptions.Item>
        <Descriptions.Item label="套餐ID">{tenant.planId}</Descriptions.Item>
        <Descriptions.Item label="状态"><Tag color={s.color}>{s.text}</Tag></Descriptions.Item>
        <Descriptions.Item label="创建时间">{tenant.createdAt}</Descriptions.Item>
      </Descriptions>
    </Card>
  )
}
```

**Step 3: Create PlanList**

```tsx
import { useRef, useState } from 'react'
import { ProTable, type ActionType, type ProColumns, ModalForm, ProFormText, ProFormDigit, ProFormTextArea } from '@ant-design/pro-components'
import { Button, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { listPlans, createPlan, updatePlan, type PlanData } from '@/api/plan'
import type { TenantPlan } from '@/types/tenant'
import { formatPrice, parsePriceToFen } from '@/constants'

export default function PlanList() {
  const actionRef = useRef<ActionType>()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TenantPlan | null>(null)

  const columns: ProColumns<TenantPlan>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '套餐名称', dataIndex: 'name' },
    { title: '价格', dataIndex: 'price', render: (_, r) => formatPrice(r.price) },
    { title: '有效天数', dataIndex: 'durationDays' },
    { title: '最大商品数', dataIndex: 'maxProducts' },
    { title: '最大员工数', dataIndex: 'maxStaff' },
    { title: '特性说明', dataIndex: 'features', ellipsis: true },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="edit" onClick={() => { setEditing(record); setModalOpen(true) }}>编辑</a>,
      ],
    },
  ]

  const handleSubmit = async (values: { name: string; price: number; durationDays: number; maxProducts: number; maxStaff: number; features: string }) => {
    const data: PlanData = { ...values, price: parsePriceToFen(values.price) }
    try {
      if (editing) {
        await updatePlan(editing.id, data)
        message.success('更新成功')
      } else {
        await createPlan(data)
        message.success('创建成功')
      }
      setModalOpen(false)
      setEditing(null)
      actionRef.current?.reload()
      return true
    } catch {
      return false
    }
  }

  return (
    <>
      <ProTable<TenantPlan>
        headerTitle="套餐管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        request={async () => {
          const res = await listPlans()
          return { data: res?.plans ?? [], total: res?.plans?.length ?? 0, success: true }
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => { setEditing(null); setModalOpen(true) }}>
            新建套餐
          </Button>,
        ]}
      />
      <ModalForm
        title={editing ? '编辑套餐' : '新建套餐'}
        open={modalOpen}
        onOpenChange={setModalOpen}
        onFinish={handleSubmit}
        initialValues={editing ? { ...editing, price: editing.price / 100 } : {}}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="套餐名称" rules={[{ required: true }]} />
        <ProFormDigit name="price" label="价格（元）" rules={[{ required: true }]} min={0} fieldProps={{ precision: 2 }} />
        <ProFormDigit name="durationDays" label="有效天数" rules={[{ required: true }]} min={1} />
        <ProFormDigit name="maxProducts" label="最大商品数" rules={[{ required: true }]} min={1} />
        <ProFormDigit name="maxStaff" label="最大员工数" rules={[{ required: true }]} min={1} />
        <ProFormTextArea name="features" label="特性说明" />
      </ModalForm>
    </>
  )
}
```

**Step 4: Commit**

```bash
git add admin-frontend/src/pages/tenant/ admin-frontend/src/pages/plan/
git commit -m "feat(admin-frontend): add TenantList, TenantDetail, and PlanList pages"
```

---

### Task 8: Product Management pages — CategoryList, BrandList

**Files:**
- Create: `admin-frontend/src/pages/category/CategoryList.tsx`
- Create: `admin-frontend/src/pages/brand/BrandList.tsx`

**Step 1: Create CategoryList**

```tsx
import { useRef, useState } from 'react'
import { ProTable, type ActionType, type ProColumns, ModalForm, ProFormText, ProFormDigit } from '@ant-design/pro-components'
import { Button, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { listCategories, createCategory, updateCategory } from '@/api/category'
import type { Category } from '@/types/product'

export default function CategoryList() {
  const actionRef = useRef<ActionType>()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Category | null>(null)

  const columns: ProColumns<Category>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '分类名称', dataIndex: 'name' },
    { title: '父级ID', dataIndex: 'parentId' },
    { title: '层级', dataIndex: 'level' },
    { title: '排序', dataIndex: 'sort' },
    { title: '图标', dataIndex: 'icon', ellipsis: true },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="edit" onClick={() => { setEditing(record); setModalOpen(true) }}>编辑</a>,
      ],
    },
  ]

  const handleSubmit = async (values: { name: string; parentId?: number; level?: number; sort?: number; icon?: string }) => {
    try {
      if (editing) {
        await updateCategory(editing.id, values)
        message.success('更新成功')
      } else {
        await createCategory(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      setEditing(null)
      actionRef.current?.reload()
      return true
    } catch {
      return false
    }
  }

  return (
    <>
      <ProTable<Category>
        headerTitle="分类管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        request={async () => {
          const res = await listCategories()
          return { data: res ?? [], total: res?.length ?? 0, success: true }
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => { setEditing(null); setModalOpen(true) }}>
            新建分类
          </Button>,
        ]}
      />
      <ModalForm
        title={editing ? '编辑分类' : '新建分类'}
        open={modalOpen}
        onOpenChange={setModalOpen}
        onFinish={handleSubmit}
        initialValues={editing ?? {}}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="分类名称" rules={[{ required: true }]} />
        <ProFormDigit name="parentId" label="父级ID" placeholder="顶级分类留空" min={0} />
        <ProFormDigit name="level" label="层级" min={1} />
        <ProFormDigit name="sort" label="排序值" min={0} />
        <ProFormText name="icon" label="图标" />
      </ModalForm>
    </>
  )
}
```

**Step 2: Create BrandList**

```tsx
import { useRef, useState } from 'react'
import { ProTable, type ActionType, type ProColumns, ModalForm, ProFormText } from '@ant-design/pro-components'
import { Button, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { listBrands, createBrand, updateBrand } from '@/api/brand'
import type { Brand } from '@/types/product'

export default function BrandList() {
  const actionRef = useRef<ActionType>()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Brand | null>(null)

  const columns: ProColumns<Brand>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '品牌名称', dataIndex: 'name' },
    { title: 'Logo', dataIndex: 'logo', ellipsis: true },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="edit" onClick={() => { setEditing(record); setModalOpen(true) }}>编辑</a>,
      ],
    },
  ]

  const handleSubmit = async (values: { name: string; logo?: string }) => {
    try {
      if (editing) {
        await updateBrand(editing.id, values)
        message.success('更新成功')
      } else {
        await createBrand(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      setEditing(null)
      actionRef.current?.reload()
      return true
    } catch {
      return false
    }
  }

  return (
    <>
      <ProTable<Brand>
        headerTitle="品牌管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        request={async (params) => {
          const res = await listBrands({ page: params.current ?? 1, pageSize: params.pageSize ?? 20 })
          return { data: res?.brands ?? [], total: res?.total ?? 0, success: true }
        }}
        pagination={{ defaultPageSize: 20 }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => { setEditing(null); setModalOpen(true) }}>
            新建品牌
          </Button>,
        ]}
      />
      <ModalForm
        title={editing ? '编辑品牌' : '新建品牌'}
        open={modalOpen}
        onOpenChange={setModalOpen}
        onFinish={handleSubmit}
        initialValues={editing ?? {}}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="品牌名称" rules={[{ required: true }]} />
        <ProFormText name="logo" label="Logo URL" />
      </ModalForm>
    </>
  )
}
```

**Step 3: Commit**

```bash
git add admin-frontend/src/pages/category/ admin-frontend/src/pages/brand/
git commit -m "feat(admin-frontend): add CategoryList and BrandList pages"
```

---

### Task 9: Order Supervision pages — OrderList, OrderDetail

**Files:**
- Create: `admin-frontend/src/pages/order/OrderList.tsx`
- Create: `admin-frontend/src/pages/order/OrderDetail.tsx`

**Step 1: Create OrderList**

```tsx
import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Tag } from 'antd'
import { listOrders } from '@/api/order'
import type { Order } from '@/types/order'
import { ORDER_STATUS_MAP, formatPrice } from '@/constants'

export default function OrderList() {
  const actionRef = useRef<ActionType>()
  const navigate = useNavigate()

  const columns: ProColumns<Order>[] = [
    { title: '订单号', dataIndex: 'orderNo', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '金额', dataIndex: 'totalAmount', search: false, render: (_, r) => formatPrice(r.totalAmount) },
    { title: '收货人', dataIndex: 'receiverName', search: false },
    {
      title: '状态',
      dataIndex: 'status',
      render: (_, record) => {
        const s = ORDER_STATUS_MAP[record.status] || { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(ORDER_STATUS_MAP).map(([k, v]) => [k, { text: v.text }])
      ),
    },
    { title: '创建时间', dataIndex: 'createdAt', search: false, valueType: 'dateTime' },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="detail" onClick={() => navigate(`/orders/${record.orderNo}`)}>详情</a>,
      ],
    },
  ]

  return (
    <ProTable<Order>
      headerTitle="订单监管"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listOrders({
          tenantId: params.tenantId,
          status: params.status !== undefined ? Number(params.status) : undefined,
          page: params.current ?? 1,
          pageSize: params.pageSize ?? 20,
        })
        return { data: res?.orders ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
```

**Step 2: Create OrderDetail**

```tsx
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Descriptions, Card, Spin, Tag, Table, Divider } from 'antd'
import { getOrder } from '@/api/order'
import { getOrderLogistics } from '@/api/logistics'
import type { Order } from '@/types/order'
import type { Shipment } from '@/types/logistics'
import { ORDER_STATUS_MAP, formatPrice } from '@/constants'
import { silentApiError } from '@/utils/error'

export default function OrderDetail() {
  const { orderNo } = useParams<{ orderNo: string }>()
  const [order, setOrder] = useState<Order | null>(null)
  const [shipment, setShipment] = useState<Shipment | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!orderNo) return
    getOrder(orderNo).then(setOrder).catch(() => {}).finally(() => setLoading(false))
    getOrderLogistics(orderNo).then(setShipment).catch(silentApiError('orderDetail:logistics'))
  }, [orderNo])

  if (loading) return <Spin style={{ display: 'block', margin: '20vh auto' }} size="large" />
  if (!order) return <div>订单不存在</div>

  const s = ORDER_STATUS_MAP[order.status] || { text: '未知', color: 'default' }

  return (
    <div>
      <Card title="订单详情">
        <Descriptions column={2} bordered>
          <Descriptions.Item label="订单号">{order.orderNo}</Descriptions.Item>
          <Descriptions.Item label="租户ID">{order.tenantId}</Descriptions.Item>
          <Descriptions.Item label="总金额">{formatPrice(order.totalAmount)}</Descriptions.Item>
          <Descriptions.Item label="实付金额">{formatPrice(order.payAmount)}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={s.color}>{s.text}</Tag></Descriptions.Item>
          <Descriptions.Item label="收货人">{order.receiverName}</Descriptions.Item>
          <Descriptions.Item label="收货电话">{order.receiverPhone}</Descriptions.Item>
          <Descriptions.Item label="收货地址">{order.receiverAddress}</Descriptions.Item>
          <Descriptions.Item label="创建时间">{order.createdAt}</Descriptions.Item>
        </Descriptions>
      </Card>

      {order.items && order.items.length > 0 && (
        <>
          <Divider />
          <Card title="商品明细">
            <Table
              dataSource={order.items}
              rowKey="id"
              pagination={false}
              columns={[
                { title: '商品', dataIndex: 'title' },
                { title: '单价', dataIndex: 'price', render: (v: number) => formatPrice(v) },
                { title: '数量', dataIndex: 'quantity' },
                { title: '小计', render: (_, r) => formatPrice(r.price * r.quantity) },
              ]}
            />
          </Card>
        </>
      )}

      {shipment && (
        <>
          <Divider />
          <Card title="物流信息">
            <Descriptions column={2} bordered>
              <Descriptions.Item label="物流公司">{shipment.company}</Descriptions.Item>
              <Descriptions.Item label="运单号">{shipment.trackingNo}</Descriptions.Item>
              <Descriptions.Item label="状态">{shipment.status}</Descriptions.Item>
              <Descriptions.Item label="创建时间">{shipment.createdAt}</Descriptions.Item>
            </Descriptions>
          </Card>
        </>
      )}
    </div>
  )
}
```

**Step 3: Commit**

```bash
git add admin-frontend/src/pages/order/
git commit -m "feat(admin-frontend): add OrderList and OrderDetail pages"
```

---

### Task 10: Notification Management pages — TemplateList, TemplateForm, SendNotification

**Files:**
- Create: `admin-frontend/src/pages/notification/TemplateList.tsx`
- Create: `admin-frontend/src/pages/notification/TemplateForm.tsx`
- Create: `admin-frontend/src/pages/notification/SendNotification.tsx`

**Step 1: Create TemplateList**

```tsx
import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Button, Popconfirm, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { listTemplates, deleteTemplate } from '@/api/notification'
import type { NotificationTemplate } from '@/types/notification'

const CHANNEL_MAP: Record<number, string> = { 1: '站内信', 2: '短信', 3: '邮件' }

export default function TemplateList() {
  const actionRef = useRef<ActionType>()
  const navigate = useNavigate()

  const handleDelete = async (id: number) => {
    await deleteTemplate(id)
    message.success('已删除')
    actionRef.current?.reload()
  }

  const columns: ProColumns<NotificationTemplate>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '模板编码', dataIndex: 'code', search: false },
    { title: '标题', dataIndex: 'title', search: false },
    {
      title: '渠道',
      dataIndex: 'channel',
      render: (_, r) => CHANNEL_MAP[r.channel] || r.channel,
      valueType: 'select',
      valueEnum: { 1: { text: '站内信' }, 2: { text: '短信' }, 3: { text: '邮件' } },
    },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: true,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="edit" onClick={() => navigate(`/notification-templates/${record.id}/edit`)}>编辑</a>,
        <Popconfirm key="delete" title="确认删除？" onConfirm={() => handleDelete(record.id)}>
          <a style={{ color: '#ff4d4f' }}>删除</a>
        </Popconfirm>,
      ],
    },
  ]

  return (
    <ProTable<NotificationTemplate>
      headerTitle="通知模板"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listTemplates({
          tenantId: params.tenantId,
          channel: params.channel !== undefined ? Number(params.channel) : undefined,
        })
        return { data: res ?? [], total: res?.length ?? 0, success: true }
      }}
      toolBarRender={() => [
        <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => navigate('/notification-templates/create')}>
          新建模板
        </Button>,
      ]}
      search={{ labelWidth: 'auto' }}
    />
  )
}
```

**Step 2: Create TemplateForm**

```tsx
import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ProForm, ProFormText, ProFormSelect, ProFormTextArea, ProFormDigit } from '@ant-design/pro-components'
import { Card, message, Spin } from 'antd'
import { createTemplate, updateTemplate, listTemplates } from '@/api/notification'

export default function TemplateForm() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isEdit = !!id
  const [loading, setLoading] = useState(isEdit)
  const [initialValues, setInitialValues] = useState<Record<string, unknown>>({})

  useEffect(() => {
    if (!isEdit) return
    listTemplates({}).then((templates) => {
      const tpl = templates?.find((t) => t.id === Number(id))
      if (tpl) setInitialValues({ tenantId: tpl.tenantId, code: tpl.code, channel: tpl.channel, title: tpl.title, content: tpl.content, status: tpl.status })
    }).finally(() => setLoading(false))
  }, [id, isEdit])

  if (loading) return <Spin style={{ display: 'block', margin: '20vh auto' }} size="large" />

  const handleFinish = async (values: { tenantId?: number; code: string; channel: number; title: string; content: string; status?: number }) => {
    try {
      if (isEdit) {
        await updateTemplate(Number(id), values)
        message.success('更新成功')
      } else {
        await createTemplate(values)
        message.success('创建成功')
      }
      navigate('/notification-templates')
    } catch { /* handled */ }
  }

  return (
    <Card title={isEdit ? '编辑通知模板' : '新建通知模板'}>
      <ProForm onFinish={handleFinish} initialValues={initialValues}>
        <ProFormDigit name="tenantId" label="租户ID" placeholder="留空为平台级模板" />
        <ProFormText name="code" label="模板编码" rules={[{ required: true }]} />
        <ProFormSelect name="channel" label="渠道" rules={[{ required: true }]} options={[
          { label: '站内信', value: 1 },
          { label: '短信', value: 2 },
          { label: '邮件', value: 3 },
        ]} />
        <ProFormText name="title" label="标题" rules={[{ required: true }]} />
        <ProFormTextArea name="content" label="内容" rules={[{ required: true }]} fieldProps={{ rows: 6 }} />
      </ProForm>
    </Card>
  )
}
```

**Step 3: Create SendNotification**

```tsx
import { useNavigate } from 'react-router-dom'
import { ProForm, ProFormText, ProFormDigit, ProFormSelect } from '@ant-design/pro-components'
import { Card, message } from 'antd'
import { sendNotification } from '@/api/notification'

export default function SendNotification() {
  const navigate = useNavigate()

  const handleFinish = async (values: { userId: number; tenantId?: number; templateCode: string; channel: number }) => {
    try {
      await sendNotification(values)
      message.success('发送成功')
      navigate('/notification-templates')
    } catch { /* handled */ }
  }

  return (
    <Card title="发送通知">
      <ProForm onFinish={handleFinish}>
        <ProFormDigit name="userId" label="用户ID" rules={[{ required: true }]} />
        <ProFormDigit name="tenantId" label="租户ID" placeholder="可选" />
        <ProFormText name="templateCode" label="模板编码" rules={[{ required: true }]} />
        <ProFormSelect name="channel" label="渠道" rules={[{ required: true }]} options={[
          { label: '站内信', value: 1 },
          { label: '短信', value: 2 },
          { label: '邮件', value: 3 },
        ]} />
      </ProForm>
    </Card>
  )
}
```

**Step 4: Commit**

```bash
git add admin-frontend/src/pages/notification/
git commit -m "feat(admin-frontend): add notification template and send pages"
```

---

### Task 11: Inventory Supervision pages — StockQuery, StockLog

**Files:**
- Create: `admin-frontend/src/pages/inventory/StockQuery.tsx`
- Create: `admin-frontend/src/pages/inventory/StockLog.tsx`

**Step 1: Create StockQuery**

```tsx
import { useState } from 'react'
import { Card, Input, Button, Descriptions, Space, message } from 'antd'
import { getStock } from '@/api/inventory'
import type { Inventory } from '@/types/inventory'

export default function StockQuery() {
  const [skuId, setSkuId] = useState('')
  const [loading, setLoading] = useState(false)
  const [stock, setStock] = useState<Inventory | null>(null)

  const handleQuery = async () => {
    if (!skuId) { message.warning('请输入 SKU ID'); return }
    setLoading(true)
    try {
      const res = await getStock(Number(skuId))
      setStock(res)
    } catch { /* handled */ }
    setLoading(false)
  }

  return (
    <div>
      <Card title="库存查询">
        <Space>
          <Input placeholder="输入 SKU ID" value={skuId} onChange={(e) => setSkuId(e.target.value)} style={{ width: 200 }} />
          <Button type="primary" loading={loading} onClick={handleQuery}>查询</Button>
        </Space>
      </Card>
      {stock && (
        <Card title="查询结果" style={{ marginTop: 16 }}>
          <Descriptions column={2} bordered>
            <Descriptions.Item label="SKU ID">{stock.skuId}</Descriptions.Item>
            <Descriptions.Item label="租户ID">{stock.tenantId}</Descriptions.Item>
            <Descriptions.Item label="总库存">{stock.stock}</Descriptions.Item>
            <Descriptions.Item label="锁定">{stock.locked}</Descriptions.Item>
            <Descriptions.Item label="可用">{stock.available}</Descriptions.Item>
          </Descriptions>
        </Card>
      )}
    </div>
  )
}
```

**Step 2: Create StockLog**

```tsx
import { useRef } from 'react'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { listLogs } from '@/api/inventory'
import type { InventoryLog } from '@/types/inventory'

export default function StockLog() {
  const actionRef = useRef<ActionType>()

  const columns: ProColumns<InventoryLog>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: 'SKU ID', dataIndex: 'skuId', valueType: 'digit', fieldProps: { placeholder: '按SKU筛选' } },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '变动量', dataIndex: 'change', search: false },
    { title: '类型', dataIndex: 'type', search: false },
    { title: '关联订单', dataIndex: 'orderNo', search: false },
    { title: '时间', dataIndex: 'createdAt', search: false, valueType: 'dateTime' },
  ]

  return (
    <ProTable<InventoryLog>
      headerTitle="库存日志"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listLogs({
          tenantId: params.tenantId,
          skuId: params.skuId,
          page: params.current ?? 1,
          pageSize: params.pageSize ?? 20,
        })
        return { data: res?.logs ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
```

**Step 3: Commit**

```bash
git add admin-frontend/src/pages/inventory/
git commit -m "feat(admin-frontend): add StockQuery and StockLog pages"
```

---

### Task 12: Marketing Supervision pages — CouponList, SeckillList, SeckillDetail, PromotionList

**Files:**
- Create: `admin-frontend/src/pages/marketing/CouponList.tsx`
- Create: `admin-frontend/src/pages/marketing/SeckillList.tsx`
- Create: `admin-frontend/src/pages/marketing/SeckillDetail.tsx`
- Create: `admin-frontend/src/pages/marketing/PromotionList.tsx`

**Step 1: Create CouponList**

```tsx
import { useRef } from 'react'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { listCoupons } from '@/api/marketing'
import type { Coupon } from '@/types/marketing'
import { formatPrice, COUPON_TYPE_OPTIONS } from '@/constants'

export default function CouponList() {
  const actionRef = useRef<ActionType>()

  const columns: ProColumns<Coupon>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '名称', dataIndex: 'name', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '类型', dataIndex: 'type', search: false, render: (_, r) => COUPON_TYPE_OPTIONS.find((o) => o.value === r.type)?.label || r.type },
    { title: '面额', dataIndex: 'value', search: false, render: (_, r) => formatPrice(r.value) },
    { title: '门槛', dataIndex: 'minAmount', search: false, render: (_, r) => formatPrice(r.minAmount) },
    { title: '已用/总量', search: false, render: (_, r) => `${r.usedCount}/${r.totalCount}` },
    { title: '开始时间', dataIndex: 'startTime', search: false, valueType: 'dateTime' },
    { title: '结束时间', dataIndex: 'endTime', search: false, valueType: 'dateTime' },
  ]

  return (
    <ProTable<Coupon>
      headerTitle="优惠券监管"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listCoupons({
          tenantId: params.tenantId,
          page: params.current ?? 1,
          pageSize: params.pageSize ?? 20,
        })
        return { data: res?.coupons ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
```

**Step 2: Create SeckillList**

```tsx
import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Tag } from 'antd'
import { listSeckill } from '@/api/marketing'
import type { SeckillActivity } from '@/types/marketing'

export default function SeckillList() {
  const actionRef = useRef<ActionType>()
  const navigate = useNavigate()

  const columns: ProColumns<SeckillActivity>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '标题', dataIndex: 'title', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '开始时间', dataIndex: 'startTime', search: false, valueType: 'dateTime' },
    { title: '结束时间', dataIndex: 'endTime', search: false, valueType: 'dateTime' },
    {
      title: '状态',
      dataIndex: 'status',
      render: (_, r) => {
        const map: Record<number, { text: string; color: string }> = {
          0: { text: '未开始', color: 'default' },
          1: { text: '进行中', color: 'green' },
          2: { text: '已结束', color: 'red' },
        }
        const s = map[r.status] || { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
      valueType: 'select',
      valueEnum: { 0: { text: '未开始' }, 1: { text: '进行中' }, 2: { text: '已结束' } },
    },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="detail" onClick={() => navigate(`/seckill/${record.id}`)}>详情</a>,
      ],
    },
  ]

  return (
    <ProTable<SeckillActivity>
      headerTitle="秒杀活动监管"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listSeckill({
          tenantId: params.tenantId,
          status: params.status !== undefined ? Number(params.status) : undefined,
          page: params.current ?? 1,
          pageSize: params.pageSize ?? 20,
        })
        return { data: res?.activities ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
```

**Step 3: Create SeckillDetail**

```tsx
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Descriptions, Card, Spin, Table, Divider, Tag } from 'antd'
import { getSeckill } from '@/api/marketing'
import type { SeckillActivity } from '@/types/marketing'
import { formatPrice } from '@/constants'

export default function SeckillDetail() {
  const { id } = useParams<{ id: string }>()
  const [activity, setActivity] = useState<SeckillActivity | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    getSeckill(Number(id)).then(setActivity).catch(() => {}).finally(() => setLoading(false))
  }, [id])

  if (loading) return <Spin style={{ display: 'block', margin: '20vh auto' }} size="large" />
  if (!activity) return <div>活动不存在</div>

  const statusMap: Record<number, { text: string; color: string }> = {
    0: { text: '未开始', color: 'default' },
    1: { text: '进行中', color: 'green' },
    2: { text: '已结束', color: 'red' },
  }
  const s = statusMap[activity.status] || { text: '未知', color: 'default' }

  return (
    <div>
      <Card title="秒杀活动详情">
        <Descriptions column={2} bordered>
          <Descriptions.Item label="ID">{activity.id}</Descriptions.Item>
          <Descriptions.Item label="标题">{activity.title}</Descriptions.Item>
          <Descriptions.Item label="租户ID">{activity.tenantId}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={s.color}>{s.text}</Tag></Descriptions.Item>
          <Descriptions.Item label="开始时间">{activity.startTime}</Descriptions.Item>
          <Descriptions.Item label="结束时间">{activity.endTime}</Descriptions.Item>
        </Descriptions>
      </Card>

      {activity.items && activity.items.length > 0 && (
        <>
          <Divider />
          <Card title="秒杀商品">
            <Table
              dataSource={activity.items}
              rowKey="id"
              pagination={false}
              columns={[
                { title: '商品ID', dataIndex: 'productId' },
                { title: 'SKU ID', dataIndex: 'skuId' },
                { title: '秒杀价', dataIndex: 'seckillPrice', render: (v: number) => formatPrice(v) },
                { title: '库存', dataIndex: 'stock' },
                { title: '限购', dataIndex: 'limit' },
              ]}
            />
          </Card>
        </>
      )}
    </div>
  )
}
```

**Step 4: Create PromotionList**

```tsx
import { useRef } from 'react'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Tag } from 'antd'
import { listPromotions } from '@/api/marketing'
import type { PromotionRule } from '@/types/marketing'
import { formatPrice } from '@/constants'

export default function PromotionList() {
  const actionRef = useRef<ActionType>()

  const columns: ProColumns<PromotionRule>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '名称', dataIndex: 'name', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '门槛', dataIndex: 'threshold', search: false, render: (_, r) => formatPrice(r.threshold) },
    { title: '优惠', dataIndex: 'discount', search: false, render: (_, r) => formatPrice(r.discount) },
    {
      title: '状态',
      dataIndex: 'status',
      render: (_, r) => {
        const map: Record<number, { text: string; color: string }> = { 0: { text: '未开始', color: 'default' }, 1: { text: '进行中', color: 'green' }, 2: { text: '已结束', color: 'red' } }
        const s = map[r.status] || { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
      valueType: 'select',
      valueEnum: { 0: { text: '未开始' }, 1: { text: '进行中' }, 2: { text: '已结束' } },
    },
    { title: '开始时间', dataIndex: 'startTime', search: false, valueType: 'dateTime' },
    { title: '结束时间', dataIndex: 'endTime', search: false, valueType: 'dateTime' },
  ]

  return (
    <ProTable<PromotionRule>
      headerTitle="促销规则监管"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listPromotions({
          tenantId: params.tenantId,
          status: params.status !== undefined ? Number(params.status) : undefined,
        })
        return { data: res ?? [], total: res?.length ?? 0, success: true }
      }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
```

**Step 5: Commit**

```bash
git add admin-frontend/src/pages/marketing/
git commit -m "feat(admin-frontend): add marketing supervision pages"
```

---

### Task 13: Logistics Supervision pages — FreightList, FreightDetail

**Files:**
- Create: `admin-frontend/src/pages/logistics/FreightList.tsx`
- Create: `admin-frontend/src/pages/logistics/FreightDetail.tsx`

**Step 1: Create FreightList**

```tsx
import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { listFreightTemplates } from '@/api/logistics'
import type { FreightTemplate } from '@/types/logistics'

export default function FreightList() {
  const actionRef = useRef<ActionType>()
  const navigate = useNavigate()

  const columns: ProColumns<FreightTemplate>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '模板名称', dataIndex: 'name', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '计费方式', dataIndex: 'chargeType', search: false, render: (_, r) => r.chargeType === 1 ? '按重量' : '按件数' },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="detail" onClick={() => navigate(`/freight-templates/${record.id}`)}>详情</a>,
      ],
    },
  ]

  return (
    <ProTable<FreightTemplate>
      headerTitle="运费模板监管"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listFreightTemplates(params.tenantId)
        return { data: res ?? [], total: res?.length ?? 0, success: true }
      }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
```

**Step 2: Create FreightDetail**

```tsx
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Descriptions, Card, Spin, Table, Divider } from 'antd'
import { getFreightTemplate } from '@/api/logistics'
import type { FreightTemplate } from '@/types/logistics'
import { formatPrice } from '@/constants'

export default function FreightDetail() {
  const { id } = useParams<{ id: string }>()
  const [template, setTemplate] = useState<FreightTemplate | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    getFreightTemplate(Number(id)).then(setTemplate).catch(() => {}).finally(() => setLoading(false))
  }, [id])

  if (loading) return <Spin style={{ display: 'block', margin: '20vh auto' }} size="large" />
  if (!template) return <div>模板不存在</div>

  return (
    <div>
      <Card title="运费模板详情">
        <Descriptions column={2} bordered>
          <Descriptions.Item label="ID">{template.id}</Descriptions.Item>
          <Descriptions.Item label="模板名称">{template.name}</Descriptions.Item>
          <Descriptions.Item label="租户ID">{template.tenantId}</Descriptions.Item>
          <Descriptions.Item label="计费方式">{template.chargeType === 1 ? '按重量' : '按件数'}</Descriptions.Item>
        </Descriptions>
      </Card>

      {template.regions && template.regions.length > 0 && (
        <>
          <Divider />
          <Card title="区域运费规则">
            <Table
              dataSource={template.regions}
              rowKey="region"
              pagination={false}
              columns={[
                { title: '区域', dataIndex: 'region' },
                { title: '首重(g)', dataIndex: 'firstWeight' },
                { title: '首费', dataIndex: 'firstFee', render: (v: number) => formatPrice(v) },
                { title: '续重(g)', dataIndex: 'continueWeight' },
                { title: '续费', dataIndex: 'continueFee', render: (v: number) => formatPrice(v) },
              ]}
            />
          </Card>
        </>
      )}
    </div>
  )
}
```

**Step 3: Commit**

```bash
git add admin-frontend/src/pages/logistics/
git commit -m "feat(admin-frontend): add logistics supervision pages"
```

---

### Task 14: Build verification and final check

**Step 1: Verify TypeScript compilation**

Run: `cd admin-frontend && npx tsc --noEmit`
Expected: PASS with 0 errors

**Step 2: Verify Vite dev build**

Run: `cd admin-frontend && npx vite build`
Expected: Build succeeds, outputs to `dist/`

**Step 3: Fix any compilation errors found**

If there are type errors, fix them. Common issues:
- Missing imports
- Type mismatches between API response and component props
- Unused imports

**Step 4: Final commit**

```bash
git add -A admin-frontend/
git commit -m "feat(admin-frontend): complete platform admin frontend with all pages"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Scaffold project | 6 config files |
| 2 | Core infrastructure | 9 files (types, utils, constants, stores, API client) |
| 3 | Domain API modules | 20 files (9 types + 11 API modules) |
| 4 | Shared components | 3 files (AuthGuard, ErrorBoundary, MainLayout) |
| 5 | Router + Entry + Login + Dashboard | 5 files |
| 6 | User management | 2 pages |
| 7 | Tenant management | 3 pages |
| 8 | Product management | 2 pages |
| 9 | Order supervision | 2 pages |
| 10 | Notification management | 3 pages |
| 11 | Inventory supervision | 2 pages |
| 12 | Marketing supervision | 4 pages |
| 13 | Logistics supervision | 2 pages |
| 14 | Build verification | 0 new files |

**Total: ~55 files, 14 tasks, covering all 54+ admin-bff API endpoints.**
