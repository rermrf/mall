# Merchant Frontend Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a complete merchant management frontend (`merchant-frontend/`) using React 19 + TypeScript + Vite + Ant Design + ProComponents + Zustand, consuming 57+ merchant-bff API endpoints.

**Architecture:** Standard SPA with Ant Design ProLayout (sidebar + header + content area). Follows the same patterns as the consumer frontend (`frontend/`) — Axios with JWT refresh, Zustand for auth state, React Router for navigation. UI library switches from antd-mobile to antd (desktop) + @ant-design/pro-components for accelerated CRUD development.

**Tech Stack:** React 19, TypeScript 5, Vite 7, Ant Design 5, @ant-design/pro-components, Zustand 5, Axios 1.x, React Router 7

---

## Task 1: Project Scaffolding

**Files:**
- Create: `merchant-frontend/package.json`
- Create: `merchant-frontend/index.html`
- Create: `merchant-frontend/vite.config.ts`
- Create: `merchant-frontend/tsconfig.json`
- Create: `merchant-frontend/tsconfig.app.json`
- Create: `merchant-frontend/src/main.tsx`
- Create: `merchant-frontend/src/App.tsx`
- Create: `merchant-frontend/src/vite-env.d.ts`

**Step 1: Create project directory**

```bash
mkdir -p merchant-frontend/src
```

**Step 2: Create package.json**

Create `merchant-frontend/package.json`:

```json
{
  "name": "mall-merchant",
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

**Step 3: Create index.html**

Create `merchant-frontend/index.html`:

```html
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>商家管理后台</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

**Step 4: Create vite.config.ts**

Create `merchant-frontend/vite.config.ts`:

```typescript
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
    port: 3001,
    proxy: {
      '/api': {
        target: 'http://localhost:8281',
        changeOrigin: true,
      },
    },
  },
})
```

**Step 5: Create tsconfig files**

Create `merchant-frontend/tsconfig.json`:

```json
{
  "files": [],
  "references": [
    { "path": "./tsconfig.app.json" }
  ]
}
```

Create `merchant-frontend/tsconfig.app.json`:

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

**Step 6: Create entry files**

Create `merchant-frontend/src/vite-env.d.ts`:

```typescript
/// <reference types="vite/client" />
```

Create `merchant-frontend/src/main.tsx`:

```typescript
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
```

Create `merchant-frontend/src/App.tsx` (placeholder):

```typescript
export default function App() {
  return <div>Merchant App</div>
}
```

**Step 7: Install dependencies and verify build**

```bash
cd merchant-frontend && npm install
npm run build
```

Expected: Build succeeds with no errors.

**Step 8: Commit**

```bash
git add merchant-frontend/
git commit -m "feat(merchant): scaffold merchant-frontend project with React + Vite + Ant Design"
```

---

## Task 2: API Client & Types

**Files:**
- Create: `merchant-frontend/src/types/api.ts`
- Create: `merchant-frontend/src/types/product.ts`
- Create: `merchant-frontend/src/types/order.ts`
- Create: `merchant-frontend/src/types/inventory.ts`
- Create: `merchant-frontend/src/types/marketing.ts`
- Create: `merchant-frontend/src/types/logistics.ts`
- Create: `merchant-frontend/src/types/user.ts`
- Create: `merchant-frontend/src/types/shop.ts`
- Create: `merchant-frontend/src/types/notification.ts`
- Create: `merchant-frontend/src/types/payment.ts`
- Create: `merchant-frontend/src/api/client.ts`

**Step 1: Create type definitions**

Create `merchant-frontend/src/types/api.ts`:

```typescript
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

Create `merchant-frontend/src/types/product.ts`:

```typescript
export interface Product {
  id: number
  category_id: number
  brand_id: number
  name: string
  subtitle: string
  main_image: string
  images: string[]
  description: string
  status: number
  skus: ProductSKU[]
  specs: ProductSpec[]
  created_at: string
  updated_at: string
}

export interface ProductSKU {
  id: number
  sku_code: string
  price: number
  original_price: number
  cost_price: number
  bar_code: string
  spec_values: string
  status: number
}

export interface ProductSpec {
  name: string
  values: string[]
}

export interface CreateProductReq {
  category_id: number
  brand_id: number
  name: string
  subtitle: string
  main_image: string
  images: string[]
  description: string
  status: number
  skus: CreateSKUReq[]
  specs: ProductSpec[]
}

export interface CreateSKUReq {
  sku_code: string
  price: number
  original_price: number
  cost_price: number
  bar_code: string
  spec_values: string
  status: number
}

export type UpdateProductReq = CreateProductReq

export interface Category {
  id: number
  parent_id: number
  name: string
  level: number
  sort: number
  icon: string
  status: number
  children?: Category[]
}

export interface CreateCategoryReq {
  parent_id: number
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

export interface CreateBrandReq {
  name: string
  logo: string
  status: number
}
```

Create `merchant-frontend/src/types/order.ts`:

```typescript
export interface Order {
  id: number
  order_no: string
  user_id: number
  total_amount: number
  pay_amount: number
  freight_amount: number
  status: number
  payment_no: string
  receiver_name: string
  receiver_phone: string
  receiver_address: string
  remark: string
  items: OrderItem[]
  created_at: string
  updated_at: string
}

export interface OrderItem {
  id: number
  product_id: number
  product_name: string
  product_image: string
  sku_id: number
  sku_code: string
  spec_values: string
  price: number
  quantity: number
}

export interface RefundOrder {
  id: number
  order_no: string
  refund_no: string
  user_id: number
  amount: number
  reason: string
  status: number
  created_at: string
  updated_at: string
}

export interface HandleRefundReq {
  refund_no: string
  approved: boolean
  reason: string
}
```

Create `merchant-frontend/src/types/inventory.ts`:

```typescript
export interface Inventory {
  sku_id: number
  total: number
  locked: number
  available: number
  alert_threshold: number
}

export interface SetStockReq {
  sku_id: number
  total: number
  alert_threshold: number
}

export interface InventoryLog {
  id: number
  sku_id: number
  change_type: string
  change_amount: number
  before_total: number
  after_total: number
  order_no: string
  created_at: string
}
```

Create `merchant-frontend/src/types/marketing.ts`:

```typescript
export interface Coupon {
  id: number
  name: string
  type: number
  threshold: number
  discount_value: number
  total_count: number
  used_count: number
  per_limit: number
  start_time: string
  end_time: string
  scope_type: number
  scope_ids: number[]
  status: number
  created_at: string
}

export interface CreateCouponReq {
  name: string
  type: number
  threshold: number
  discount_value: number
  total_count: number
  per_limit: number
  start_time: string
  end_time: string
  scope_type: number
  scope_ids: number[]
  status: number
}

export interface SeckillActivity {
  id: number
  name: string
  start_time: string
  end_time: string
  status: number
  items: SeckillItem[]
  created_at: string
}

export interface SeckillItem {
  sku_id: number
  seckill_price: number
  seckill_stock: number
  per_limit: number
}

export interface CreateSeckillReq {
  name: string
  start_time: string
  end_time: string
  status: number
  items: SeckillItem[]
}

export interface PromotionRule {
  id: number
  name: string
  type: number
  threshold: number
  discount_value: number
  start_time: string
  end_time: string
  status: number
  created_at: string
}

export interface CreatePromotionReq {
  name: string
  type: number
  threshold: number
  discount_value: number
  start_time: string
  end_time: string
  status: number
}
```

Create `merchant-frontend/src/types/logistics.ts`:

```typescript
export interface FreightTemplate {
  id: number
  name: string
  charge_type: number
  free_threshold: number
  rules: FreightRule[]
  created_at: string
}

export interface FreightRule {
  regions: string[]
  first_unit: number
  first_price: number
  additional_unit: number
  additional_price: number
}

export interface CreateFreightTemplateReq {
  name: string
  charge_type: number
  free_threshold: number
  rules: FreightRule[]
}

export interface Shipment {
  order_no: string
  carrier_code: string
  carrier_name: string
  tracking_no: string
  status: number
  created_at: string
}

export interface ShipOrderReq {
  carrier_code: string
  carrier_name: string
  tracking_no: string
}
```

Create `merchant-frontend/src/types/user.ts`:

```typescript
export interface User {
  id: number
  phone: string
  nickname: string
  avatar: string
  email: string
  role: string
  created_at: string
}

export interface Role {
  id: number
  name: string
  code: string
  description: string
}

export interface CreateRoleReq {
  name: string
  code: string
  description: string
}
```

Create `merchant-frontend/src/types/shop.ts`:

```typescript
export interface Shop {
  id: number
  name: string
  logo: string
  description: string
  subdomain: string
  custom_domain: string
  plan: string
  status: number
}

export interface UpdateShopReq {
  name: string
  logo: string
  description: string
  subdomain: string
  custom_domain: string
}

export interface QuotaInfo {
  type: string
  used: number
  limit: number
}
```

Create `merchant-frontend/src/types/notification.ts`:

```typescript
export interface Notification {
  id: number
  title: string
  content: string
  channel: string
  is_read: boolean
  created_at: string
}
```

Create `merchant-frontend/src/types/payment.ts`:

```typescript
export interface Payment {
  id: number
  payment_no: string
  order_no: string
  amount: number
  status: number
  channel: string
  created_at: string
}

export interface Refund {
  refund_no: string
  payment_no: string
  amount: number
  reason: string
  status: number
  created_at: string
}

export interface RefundReq {
  amount: number
  reason: string
}
```

**Step 2: Create Axios client**

Create `merchant-frontend/src/api/client.ts` — copied from consumer frontend pattern with same JWT refresh logic:

```typescript
import axios from 'axios'
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
    return Promise.reject(error)
  },
)

export async function request<T>(config: Parameters<typeof client.request>[0]): Promise<T> {
  const res = await client.request(config)
  const body = res.data as ApiResult<T>
  if (body.code !== 0) {
    throw new Error(body.msg || '请求失败')
  }
  return body.data
}

export { client }
export default client
```

**Step 3: Verify build**

```bash
cd merchant-frontend && npm run build
```

**Step 4: Commit**

```bash
git add merchant-frontend/src/types/ merchant-frontend/src/api/client.ts
git commit -m "feat(merchant): add API client with JWT refresh and all type definitions"
```

---

## Task 3: API Modules

**Files:**
- Create: `merchant-frontend/src/api/auth.ts`
- Create: `merchant-frontend/src/api/product.ts`
- Create: `merchant-frontend/src/api/order.ts`
- Create: `merchant-frontend/src/api/inventory.ts`
- Create: `merchant-frontend/src/api/marketing.ts`
- Create: `merchant-frontend/src/api/logistics.ts`
- Create: `merchant-frontend/src/api/shop.ts`
- Create: `merchant-frontend/src/api/staff.ts`
- Create: `merchant-frontend/src/api/notification.ts`
- Create: `merchant-frontend/src/api/payment.ts`

**Step 1: Create all API modules**

Create `merchant-frontend/src/api/auth.ts`:

```typescript
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

Create `merchant-frontend/src/api/product.ts`:

```typescript
import { request } from './client'
import type { Product, CreateProductReq, Category, CreateCategoryReq, Brand, CreateBrandReq } from '@/types/product'

export async function createProduct(data: CreateProductReq) {
  return request<{ id: number }>({ method: 'POST', url: '/products', data })
}

export async function updateProduct(id: number, data: CreateProductReq) {
  return request<null>({ method: 'PUT', url: `/products/${id}`, data })
}

export async function getProduct(id: number) {
  return request<Product>({ method: 'GET', url: `/products/${id}` })
}

export async function listProducts(params: { category_id?: number; status?: number; page?: number; pageSize?: number }) {
  return request<{ products: Product[]; total: number }>({ method: 'GET', url: '/products', params })
}

export async function updateProductStatus(id: number, status: number) {
  return request<null>({ method: 'PUT', url: `/products/${id}/status`, data: { status } })
}

export async function createCategory(data: CreateCategoryReq) {
  return request<{ id: number }>({ method: 'POST', url: '/categories', data })
}

export async function updateCategory(id: number, data: CreateCategoryReq) {
  return request<null>({ method: 'PUT', url: `/categories/${id}`, data })
}

export async function listCategories() {
  return request<Category[]>({ method: 'GET', url: '/categories' })
}

export async function createBrand(data: CreateBrandReq) {
  return request<{ id: number }>({ method: 'POST', url: '/brands', data })
}

export async function updateBrand(id: number, data: CreateBrandReq) {
  return request<null>({ method: 'PUT', url: `/brands/${id}`, data })
}

export async function listBrands(params?: { page?: number; pageSize?: number }) {
  return request<{ brands: Brand[]; total: number }>({ method: 'GET', url: '/brands', params })
}
```

Create `merchant-frontend/src/api/order.ts`:

```typescript
import { request } from './client'
import type { Order, RefundOrder, HandleRefundReq } from '@/types/order'

export async function listOrders(params: { status?: number; page?: number; pageSize?: number }) {
  return request<{ orders: Order[]; total: number }>({ method: 'GET', url: '/orders', params })
}

export async function getOrder(orderNo: string) {
  return request<Order>({ method: 'GET', url: `/orders/${orderNo}` })
}

export async function handleRefund(orderNo: string, data: HandleRefundReq) {
  return request<null>({ method: 'POST', url: `/orders/${orderNo}/refund/handle`, data })
}

export async function listRefunds(params: { status?: number; page?: number; pageSize?: number }) {
  return request<{ refund_orders: RefundOrder[]; total: number }>({ method: 'GET', url: '/refunds', params })
}
```

Create `merchant-frontend/src/api/inventory.ts`:

```typescript
import { request } from './client'
import type { Inventory, SetStockReq, InventoryLog } from '@/types/inventory'

export async function setStock(data: SetStockReq) {
  return request<null>({ method: 'POST', url: '/inventory/stock', data })
}

export async function getStock(skuId: number) {
  return request<Inventory>({ method: 'GET', url: `/inventory/stock/${skuId}` })
}

export async function batchGetStock(skuIds: number[]) {
  return request<Inventory[]>({ method: 'POST', url: '/inventory/stock/batch', data: { sku_ids: skuIds } })
}

export async function listInventoryLogs(params: { sku_id?: number; page?: number; pageSize?: number }) {
  return request<{ logs: InventoryLog[]; total: number }>({ method: 'GET', url: '/inventory/logs', params })
}
```

Create `merchant-frontend/src/api/marketing.ts`:

```typescript
import { request } from './client'
import type { Coupon, CreateCouponReq, SeckillActivity, CreateSeckillReq, PromotionRule, CreatePromotionReq } from '@/types/marketing'

export async function createCoupon(data: CreateCouponReq) {
  return request<{ id: number }>({ method: 'POST', url: '/coupons', data })
}

export async function updateCoupon(id: number, data: CreateCouponReq) {
  return request<null>({ method: 'PUT', url: `/coupons/${id}`, data })
}

export async function listCoupons(params: { status?: number; page?: number; pageSize?: number }) {
  return request<{ coupons: Coupon[]; total: number }>({ method: 'GET', url: '/coupons', params })
}

export async function createSeckill(data: CreateSeckillReq) {
  return request<{ id: number }>({ method: 'POST', url: '/seckill', data })
}

export async function updateSeckill(id: number, data: CreateSeckillReq) {
  return request<null>({ method: 'PUT', url: `/seckill/${id}`, data })
}

export async function listSeckill(params: { status?: number; page?: number; pageSize?: number }) {
  return request<{ activities: SeckillActivity[]; total: number }>({ method: 'GET', url: '/seckill', params })
}

export async function getSeckill(id: number) {
  return request<SeckillActivity>({ method: 'GET', url: `/seckill/${id}` })
}

export async function createPromotion(data: CreatePromotionReq) {
  return request<{ id: number }>({ method: 'POST', url: '/promotions', data })
}

export async function updatePromotion(id: number, data: CreatePromotionReq) {
  return request<null>({ method: 'PUT', url: `/promotions/${id}`, data })
}

export async function listPromotions(params?: { status?: number }) {
  return request<PromotionRule[]>({ method: 'GET', url: '/promotions', params })
}
```

Create `merchant-frontend/src/api/logistics.ts`:

```typescript
import { request } from './client'
import type { FreightTemplate, CreateFreightTemplateReq, Shipment, ShipOrderReq } from '@/types/logistics'

export async function createFreightTemplate(data: CreateFreightTemplateReq) {
  return request<{ id: number }>({ method: 'POST', url: '/freight-templates', data })
}

export async function updateFreightTemplate(id: number, data: CreateFreightTemplateReq) {
  return request<null>({ method: 'PUT', url: `/freight-templates/${id}`, data })
}

export async function getFreightTemplate(id: number) {
  return request<FreightTemplate>({ method: 'GET', url: `/freight-templates/${id}` })
}

export async function listFreightTemplates() {
  return request<FreightTemplate[]>({ method: 'GET', url: '/freight-templates' })
}

export async function deleteFreightTemplate(id: number) {
  return request<null>({ method: 'DELETE', url: `/freight-templates/${id}` })
}

export async function shipOrder(orderNo: string, data: ShipOrderReq) {
  return request<null>({ method: 'POST', url: `/orders/${orderNo}/ship`, data })
}

export async function getOrderLogistics(orderNo: string) {
  return request<Shipment>({ method: 'GET', url: `/orders/${orderNo}/logistics` })
}
```

Create `merchant-frontend/src/api/shop.ts`:

```typescript
import { request } from './client'
import type { Shop, UpdateShopReq, QuotaInfo } from '@/types/shop'

export async function getShop() {
  return request<Shop>({ method: 'GET', url: '/shop' })
}

export async function updateShop(data: UpdateShopReq) {
  return request<null>({ method: 'PUT', url: '/shop', data })
}

export async function checkQuota(type: string) {
  return request<QuotaInfo>({ method: 'GET', url: `/quotas/${type}` })
}
```

Create `merchant-frontend/src/api/staff.ts`:

```typescript
import { request } from './client'
import type { User, Role, CreateRoleReq } from '@/types/user'

export async function getProfile() {
  return request<User>({ method: 'GET', url: '/profile' })
}

export async function updateProfile(data: { nickname: string; avatar: string }) {
  return request<null>({ method: 'PUT', url: '/profile', data })
}

export async function listStaff(params: { page?: number; pageSize?: number }) {
  return request<{ users: User[]; total: number }>({ method: 'GET', url: '/staff', params })
}

export async function assignRole(userId: number, roleId: number) {
  return request<null>({ method: 'POST', url: `/staff/${userId}/role`, data: { role_id: roleId } })
}

export async function listRoles() {
  return request<Role[]>({ method: 'GET', url: '/roles' })
}

export async function createRole(data: CreateRoleReq) {
  return request<{ id: number }>({ method: 'POST', url: '/roles', data })
}

export async function updateRole(id: number, data: CreateRoleReq) {
  return request<null>({ method: 'PUT', url: `/roles/${id}`, data })
}
```

Create `merchant-frontend/src/api/notification.ts`:

```typescript
import { request } from './client'
import type { Notification } from '@/types/notification'

export async function listNotifications(params: { channel?: string; unread_only?: boolean; page?: number; pageSize?: number }) {
  return request<{ notifications: Notification[]; total: number }>({ method: 'GET', url: '/notifications', params })
}

export async function getUnreadCount() {
  return request<number>({ method: 'GET', url: '/notifications/unread-count' })
}

export async function markRead(id: number) {
  return request<null>({ method: 'PUT', url: `/notifications/${id}/read` })
}

export async function markAllRead() {
  return request<null>({ method: 'PUT', url: '/notifications/read-all' })
}
```

Create `merchant-frontend/src/api/payment.ts`:

```typescript
import { request } from './client'
import type { Payment, Refund, RefundReq } from '@/types/payment'

export async function listPayments(params: { status?: number; page?: number; pageSize?: number }) {
  return request<{ payments: Payment[]; total: number }>({ method: 'GET', url: '/payments', params })
}

export async function getPayment(paymentNo: string) {
  return request<Payment>({ method: 'GET', url: `/payments/${paymentNo}` })
}

export async function refundPayment(paymentNo: string, data: RefundReq) {
  return request<{ refund_no: string }>({ method: 'POST', url: `/payments/${paymentNo}/refund`, data })
}

export async function getRefund(refundNo: string) {
  return request<Refund>({ method: 'GET', url: `/refunds/${refundNo}/payment` })
}
```

**Step 2: Verify build**

```bash
cd merchant-frontend && npm run build
```

**Step 3: Commit**

```bash
git add merchant-frontend/src/api/
git commit -m "feat(merchant): add all API modules covering 57+ merchant-bff endpoints"
```

---

## Task 4: Auth Store, Layout & Router

**Files:**
- Create: `merchant-frontend/src/stores/auth.ts`
- Create: `merchant-frontend/src/stores/notification.ts`
- Create: `merchant-frontend/src/components/layout/MainLayout.tsx`
- Create: `merchant-frontend/src/components/AuthGuard.tsx`
- Create: `merchant-frontend/src/router/index.tsx`
- Modify: `merchant-frontend/src/App.tsx`

**Step 1: Create Zustand stores**

Create `merchant-frontend/src/stores/auth.ts`:

```typescript
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

Create `merchant-frontend/src/stores/notification.ts`:

```typescript
import { create } from 'zustand'
import { getUnreadCount } from '@/api/notification'

interface NotificationState {
  unreadCount: number
  fetchUnreadCount: () => Promise<void>
}

export const useNotificationStore = create<NotificationState>((set) => ({
  unreadCount: 0,
  fetchUnreadCount: async () => {
    try {
      const count = await getUnreadCount()
      set({ unreadCount: count ?? 0 })
    } catch {
      // ignore
    }
  },
}))
```

**Step 2: Create AuthGuard**

Create `merchant-frontend/src/components/AuthGuard.tsx`:

```typescript
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

**Step 3: Create MainLayout with ProLayout**

Create `merchant-frontend/src/components/layout/MainLayout.tsx`:

```typescript
import { useState, useEffect } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { ProLayout } from '@ant-design/pro-components'
import { Badge, Dropdown, Avatar, message } from 'antd'
import {
  DashboardOutlined,
  ShoppingOutlined,
  OrderedListOutlined,
  InboxOutlined,
  GiftOutlined,
  CarOutlined,
  ShopOutlined,
  TeamOutlined,
  BellOutlined,
  LogoutOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '@/stores/auth'
import { useNotificationStore } from '@/stores/notification'
import { logout } from '@/api/auth'
import { getProfile } from '@/api/staff'

const menuRoutes = {
  path: '/',
  routes: [
    { path: '/', name: '仪表盘', icon: <DashboardOutlined /> },
    {
      path: '/product',
      name: '商品管理',
      icon: <ShoppingOutlined />,
      routes: [
        { path: '/product/list', name: '商品列表' },
        { path: '/product/category', name: '分类管理' },
        { path: '/product/brand', name: '品牌管理' },
      ],
    },
    {
      path: '/order',
      name: '订单管理',
      icon: <OrderedListOutlined />,
      routes: [
        { path: '/order/list', name: '订单列表' },
        { path: '/order/refund', name: '退款管理' },
      ],
    },
    {
      path: '/inventory',
      name: '库存管理',
      icon: <InboxOutlined />,
      routes: [
        { path: '/inventory', name: '库存查看' },
        { path: '/inventory/log', name: '变更日志' },
      ],
    },
    {
      path: '/marketing',
      name: '营销管理',
      icon: <GiftOutlined />,
      routes: [
        { path: '/marketing/coupon', name: '优惠券' },
        { path: '/marketing/seckill', name: '秒杀活动' },
        { path: '/marketing/promotion', name: '促销规则' },
      ],
    },
    {
      path: '/logistics',
      name: '物流管理',
      icon: <CarOutlined />,
      routes: [
        { path: '/logistics/template', name: '运费模板' },
      ],
    },
    { path: '/shop/settings', name: '店铺设置', icon: <ShopOutlined /> },
    {
      path: '/staff',
      name: '团队管理',
      icon: <TeamOutlined />,
      routes: [
        { path: '/staff/list', name: '员工列表' },
        { path: '/staff/role', name: '角色管理' },
      ],
    },
    { path: '/notification', name: '消息中心', icon: <BellOutlined /> },
  ],
}

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const clearAuth = useAuthStore((s) => s.clearAuth)
  const { unreadCount, fetchUnreadCount } = useNotificationStore()
  const [nickname, setNickname] = useState('商家')

  useEffect(() => {
    fetchUnreadCount()
    getProfile().then((u) => { if (u?.nickname) setNickname(u.nickname) }).catch(() => {})
  }, [fetchUnreadCount])

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
      title="商家管理后台"
      logo={<ShopOutlined style={{ fontSize: 28, color: '#1890ff' }} />}
      route={menuRoutes}
      location={{ pathname: location.pathname }}
      menuItemRender={(item, dom) => (
        <span onClick={() => item.path && navigate(item.path)}>{dom}</span>
      )}
      actionsRender={() => [
        <Badge key="bell" count={unreadCount} size="small" offset={[-2, 2]}>
          <BellOutlined style={{ fontSize: 18, cursor: 'pointer' }} onClick={() => navigate('/notification')} />
        </Badge>,
        <Dropdown
          key="user"
          menu={{
            items: [
              { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
            ],
          }}
        >
          <span style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 8 }}>
            <Avatar size="small" icon={<UserOutlined />} />
            <span>{nickname}</span>
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

**Step 4: Create router**

Create `merchant-frontend/src/router/index.tsx`:

```typescript
import { lazy, Suspense } from 'react'
import { createBrowserRouter } from 'react-router-dom'
import { Spin } from 'antd'
import MainLayout from '@/components/layout/MainLayout'
import AuthGuard from '@/components/AuthGuard'

const LoginPage = lazy(() => import('@/pages/login'))
const Dashboard = lazy(() => import('@/pages/dashboard'))
const ProductList = lazy(() => import('@/pages/product/ProductList'))
const ProductForm = lazy(() => import('@/pages/product/ProductForm'))
const CategoryList = lazy(() => import('@/pages/product/CategoryList'))
const BrandList = lazy(() => import('@/pages/product/BrandList'))
const OrderList = lazy(() => import('@/pages/order/OrderList'))
const OrderDetail = lazy(() => import('@/pages/order/OrderDetail'))
const RefundList = lazy(() => import('@/pages/order/RefundList'))
const StockList = lazy(() => import('@/pages/inventory/StockList'))
const StockLog = lazy(() => import('@/pages/inventory/StockLog'))
const CouponList = lazy(() => import('@/pages/marketing/CouponList'))
const CouponForm = lazy(() => import('@/pages/marketing/CouponForm'))
const SeckillList = lazy(() => import('@/pages/marketing/SeckillList'))
const SeckillForm = lazy(() => import('@/pages/marketing/SeckillForm'))
const PromotionList = lazy(() => import('@/pages/marketing/PromotionList'))
const TemplateList = lazy(() => import('@/pages/logistics/TemplateList'))
const TemplateForm = lazy(() => import('@/pages/logistics/TemplateForm'))
const ShopSettings = lazy(() => import('@/pages/shop/ShopSettings'))
const StaffList = lazy(() => import('@/pages/staff/StaffList'))
const RoleList = lazy(() => import('@/pages/staff/RoleList'))
const NotificationList = lazy(() => import('@/pages/notification/NotificationList'))

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
    children: [
      { path: '/', element: <L><Dashboard /></L> },
      { path: '/product/list', element: <L><ProductList /></L> },
      { path: '/product/create', element: <L><ProductForm /></L> },
      { path: '/product/edit/:id', element: <L><ProductForm /></L> },
      { path: '/product/category', element: <L><CategoryList /></L> },
      { path: '/product/brand', element: <L><BrandList /></L> },
      { path: '/order/list', element: <L><OrderList /></L> },
      { path: '/order/:orderNo', element: <L><OrderDetail /></L> },
      { path: '/order/refund', element: <L><RefundList /></L> },
      { path: '/inventory', element: <L><StockList /></L> },
      { path: '/inventory/log', element: <L><StockLog /></L> },
      { path: '/marketing/coupon', element: <L><CouponList /></L> },
      { path: '/marketing/coupon/create', element: <L><CouponForm /></L> },
      { path: '/marketing/coupon/edit/:id', element: <L><CouponForm /></L> },
      { path: '/marketing/seckill', element: <L><SeckillList /></L> },
      { path: '/marketing/seckill/create', element: <L><SeckillForm /></L> },
      { path: '/marketing/seckill/edit/:id', element: <L><SeckillForm /></L> },
      { path: '/marketing/promotion', element: <L><PromotionList /></L> },
      { path: '/logistics/template', element: <L><TemplateList /></L> },
      { path: '/logistics/template/create', element: <L><TemplateForm /></L> },
      { path: '/logistics/template/edit/:id', element: <L><TemplateForm /></L> },
      { path: '/shop/settings', element: <L><ShopSettings /></L> },
      { path: '/staff/list', element: <L><StaffList /></L> },
      { path: '/staff/role', element: <L><RoleList /></L> },
      { path: '/notification', element: <L><NotificationList /></L> },
    ],
  },
])
```

**Step 5: Update App.tsx**

Replace `merchant-frontend/src/App.tsx`:

```typescript
import { RouterProvider } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { router } from './router'

export default function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <RouterProvider router={router} />
    </ConfigProvider>
  )
}
```

**Step 6: Commit**

```bash
git add merchant-frontend/src/
git commit -m "feat(merchant): add auth store, ProLayout, router with all 25 routes"
```

---

## Task 5: Login Page

**Files:**
- Create: `merchant-frontend/src/pages/login/index.tsx`

**Step 1: Create login page**

Create `merchant-frontend/src/pages/login/index.tsx`:

```typescript
import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Form, Input, Button, Card, message } from 'antd'
import { ShopOutlined, PhoneOutlined, LockOutlined } from '@ant-design/icons'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

export default function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const setLoggedIn = useAuthStore((s) => s.setLoggedIn)
  const [loading, setLoading] = useState(false)
  const redirect = searchParams.get('redirect') || '/'

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
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    }}>
      <Card style={{ width: 400, borderRadius: 8 }} bordered={false}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <ShopOutlined style={{ fontSize: 48, color: '#1890ff' }} />
          <h2 style={{ marginTop: 16, marginBottom: 4 }}>商家管理后台</h2>
          <p style={{ color: '#999' }}>登录你的商家账户</p>
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

**Step 2: Commit**

```bash
git add merchant-frontend/src/pages/login/
git commit -m "feat(merchant): add login page"
```

---

## Task 6: Dashboard Page

**Files:**
- Create: `merchant-frontend/src/pages/dashboard/index.tsx`

**Step 1: Create dashboard**

Create `merchant-frontend/src/pages/dashboard/index.tsx`:

```typescript
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Col, Row, Statistic, List, Button, Typography } from 'antd'
import {
  ShoppingCartOutlined,
  OrderedListOutlined,
  AlertOutlined,
  GiftOutlined,
} from '@ant-design/icons'
import { listOrders } from '@/api/order'
import { listRefunds } from '@/api/order'

const { Title } = Typography

export default function Dashboard() {
  const navigate = useNavigate()
  const [pendingShip, setPendingShip] = useState(0)
  const [pendingRefund, setPendingRefund] = useState(0)
  const [todayOrders, setTodayOrders] = useState(0)

  useEffect(() => {
    // status=2: 待发货
    listOrders({ status: 2, page: 1, pageSize: 1 }).then((r) => {
      setPendingShip(r?.total ?? 0)
    }).catch(() => {})
    // status=1: 待处理退款
    listRefunds({ status: 1, page: 1, pageSize: 1 }).then((r) => {
      setPendingRefund(r?.total ?? 0)
    }).catch(() => {})
    // 所有订单
    listOrders({ page: 1, pageSize: 1 }).then((r) => {
      setTodayOrders(r?.total ?? 0)
    }).catch(() => {})
  }, [])

  const shortcuts = [
    { title: '发布商品', path: '/product/create' },
    { title: '处理订单', path: '/order/list' },
    { title: '管理库存', path: '/inventory' },
    { title: '创建优惠券', path: '/marketing/coupon/create' },
    { title: '店铺设置', path: '/shop/settings' },
  ]

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>概览</Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/order/list')}>
            <Statistic title="总订单数" value={todayOrders} prefix={<ShoppingCartOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/order/list')}>
            <Statistic title="待发货" value={pendingShip} prefix={<OrderedListOutlined />} valueStyle={pendingShip > 0 ? { color: '#faad14' } : undefined} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/order/refund')}>
            <Statistic title="待处理退款" value={pendingRefund} prefix={<AlertOutlined />} valueStyle={pendingRefund > 0 ? { color: '#ff4d4f' } : undefined} />
          </Card>
        </Col>
      </Row>

      <Card title="快捷入口" style={{ marginTop: 24 }}>
        <List
          grid={{ gutter: 16, xs: 2, sm: 3, md: 5 }}
          dataSource={shortcuts}
          renderItem={(item) => (
            <List.Item>
              <Button block onClick={() => navigate(item.path)} icon={<GiftOutlined />}>
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

**Step 2: Commit**

```bash
git add merchant-frontend/src/pages/dashboard/
git commit -m "feat(merchant): add dashboard page with stats and shortcuts"
```

---

## Task 7: Product Management Pages

**Files:**
- Create: `merchant-frontend/src/pages/product/ProductList.tsx`
- Create: `merchant-frontend/src/pages/product/ProductForm.tsx`
- Create: `merchant-frontend/src/pages/product/CategoryList.tsx`
- Create: `merchant-frontend/src/pages/product/BrandList.tsx`

**Step 1: Create ProductList — demonstrates the ProTable pattern used across all list pages**

Create `merchant-frontend/src/pages/product/ProductList.tsx`:

```typescript
import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button, message, Switch, Tag } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listProducts, updateProductStatus } from '@/api/product'
import type { Product } from '@/types/product'

export default function ProductList() {
  const navigate = useNavigate()
  const actionRef = useRef<ActionType>()

  const handleStatusChange = async (id: number, checked: boolean) => {
    try {
      await updateProductStatus(id, checked ? 1 : 0)
      message.success(checked ? '已上架' : '已下架')
      actionRef.current?.reload()
    } catch (e: unknown) {
      message.error((e as Error).message)
    }
  }

  const columns: ProColumns<Product>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '商品图片', dataIndex: 'main_image', valueType: 'image', width: 80, search: false },
    { title: '商品名称', dataIndex: 'name', ellipsis: true },
    { title: '分类ID', dataIndex: 'category_id', width: 80, search: false },
    {
      title: '价格',
      dataIndex: 'skus',
      search: false,
      width: 120,
      render: (_, record) => {
        const prices = (record.skus ?? []).map((s) => s.price)
        if (prices.length === 0) return '-'
        const min = Math.min(...prices)
        const max = Math.max(...prices)
        return min === max ? `¥${(min / 100).toFixed(2)}` : `¥${(min / 100).toFixed(2)} - ¥${(max / 100).toFixed(2)}`
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      valueEnum: { 0: { text: '下架', status: 'Default' }, 1: { text: '上架', status: 'Success' } },
      render: (_, record) => (
        <Switch
          checked={record.status === 1}
          checkedChildren="上架"
          unCheckedChildren="下架"
          onChange={(checked) => handleStatusChange(record.id, checked)}
        />
      ),
    },
    {
      title: 'SKU数',
      dataIndex: 'skus',
      width: 80,
      search: false,
      render: (_, record) => <Tag>{record.skus?.length ?? 0}</Tag>,
    },
    {
      title: '操作',
      width: 120,
      search: false,
      render: (_, record) => (
        <Button type="link" onClick={() => navigate(`/product/edit/${record.id}`)}>编辑</Button>
      ),
    },
  ]

  return (
    <ProTable<Product>
      headerTitle="商品列表"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      toolBarRender={() => [
        <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => navigate('/product/create')}>
          发布商品
        </Button>,
      ]}
      request={async (params) => {
        const { current, pageSize, status } = params
        const res = await listProducts({ page: current, pageSize, status })
        return { data: res?.products ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
```

**Step 2: Create ProductForm**

Create `merchant-frontend/src/pages/product/ProductForm.tsx`:

```typescript
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormTextArea, ProFormDigit, ProFormSelect, StepsForm } from '@ant-design/pro-components'
import { createProduct, updateProduct, getProduct, listCategories, listBrands } from '@/api/product'
import type { Category, Brand, CreateProductReq, CreateSKUReq, ProductSpec } from '@/types/product'

export default function ProductForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id

  const [categories, setCategories] = useState<Category[]>([])
  const [brands, setBrands] = useState<Brand[]>([])
  const [initialValues, setInitialValues] = useState<Record<string, unknown>>({})
  const [skus, setSkus] = useState<CreateSKUReq[]>([])
  const [specs, setSpecs] = useState<ProductSpec[]>([])

  useEffect(() => {
    listCategories().then((c) => setCategories(c ?? [])).catch(() => {})
    listBrands({ page: 1, pageSize: 100 }).then((r) => setBrands(r?.brands ?? [])).catch(() => {})
    if (isEdit) {
      getProduct(Number(id)).then((p) => {
        if (p) {
          setInitialValues({
            name: p.name,
            subtitle: p.subtitle,
            category_id: p.category_id,
            brand_id: p.brand_id,
            main_image: p.main_image,
            description: p.description,
          })
          setSkus(p.skus?.map((s) => ({
            sku_code: s.sku_code,
            price: s.price,
            original_price: s.original_price,
            cost_price: s.cost_price,
            bar_code: s.bar_code,
            spec_values: s.spec_values,
            status: s.status,
          })) ?? [])
          setSpecs(p.specs ?? [])
        }
      }).catch(() => {})
    }
  }, [id, isEdit])

  const flatCategories = (cats: Category[], prefix = ''): { label: string; value: number }[] => {
    return cats.flatMap((c) => [
      { label: prefix + c.name, value: c.id },
      ...(c.children ? flatCategories(c.children, prefix + '  ') : []),
    ])
  }

  const handleSubmit = async (values: Record<string, unknown>) => {
    const data: CreateProductReq = {
      category_id: values.category_id as number,
      brand_id: values.brand_id as number,
      name: values.name as string,
      subtitle: (values.subtitle as string) || '',
      main_image: (values.main_image as string) || '',
      images: [],
      description: (values.description as string) || '',
      status: 0,
      skus: skus.length > 0 ? skus : [{
        sku_code: 'DEFAULT',
        price: ((values.price as number) || 0) * 100,
        original_price: ((values.original_price as number) || 0) * 100,
        cost_price: 0,
        bar_code: '',
        spec_values: '',
        status: 1,
      }],
      specs,
    }

    try {
      if (isEdit) {
        await updateProduct(Number(id), data)
        message.success('更新成功')
      } else {
        await createProduct(data)
        message.success('创建成功')
      }
      navigate('/product/list')
    } catch (e: unknown) {
      message.error((e as Error).message)
    }
  }

  return (
    <Card title={isEdit ? '编辑商品' : '发布商品'}>
      <StepsForm onFinish={handleSubmit}>
        <StepsForm.StepForm name="basic" title="基本信息" initialValues={initialValues}>
          <ProFormText name="name" label="商品名称" rules={[{ required: true }]} />
          <ProFormText name="subtitle" label="副标题" />
          <ProFormSelect name="category_id" label="分类" rules={[{ required: true }]} options={flatCategories(categories)} />
          <ProFormSelect name="brand_id" label="品牌" options={brands.map((b) => ({ label: b.name, value: b.id }))} />
          <ProFormText name="main_image" label="主图URL" />
          <ProFormTextArea name="description" label="商品描述" />
        </StepsForm.StepForm>
        <StepsForm.StepForm name="sku" title="价格库存">
          <ProFormDigit name="price" label="售价（元）" rules={[{ required: true }]} min={0} fieldProps={{ precision: 2 }} />
          <ProFormDigit name="original_price" label="原价（元）" min={0} fieldProps={{ precision: 2 }} />
        </StepsForm.StepForm>
      </StepsForm>
    </Card>
  )
}
```

**Step 3: Create CategoryList**

Create `merchant-frontend/src/pages/product/CategoryList.tsx`:

```typescript
import { useRef, useState } from 'react'
import { message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable, ModalForm, ProFormText, ProFormDigit, ProFormSelect } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listCategories, createCategory, updateCategory } from '@/api/product'
import type { Category, CreateCategoryReq } from '@/types/product'

export default function CategoryList() {
  const actionRef = useRef<ActionType>()
  const [editItem, setEditItem] = useState<Category | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const columns: ProColumns<Category>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '名称', dataIndex: 'name' },
    { title: '排序', dataIndex: 'sort', width: 80 },
    { title: '层级', dataIndex: 'level', width: 80 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      valueEnum: { 0: { text: '禁用', status: 'Default' }, 1: { text: '启用', status: 'Success' } },
    },
    {
      title: '操作',
      width: 100,
      render: (_, record) => (
        <a onClick={() => { setEditItem(record); setModalOpen(true) }}>编辑</a>
      ),
    },
  ]

  return (
    <>
      <ProTable<Category>
        headerTitle="分类管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        toolBarRender={() => [
          <ModalForm<CreateCategoryReq>
            key="add"
            title="新增分类"
            trigger={<a><PlusOutlined /> 新增分类</a>}
            onFinish={async (values) => {
              await createCategory(values)
              message.success('创建成功')
              actionRef.current?.reload()
              return true
            }}
          >
            <ProFormText name="name" label="名称" rules={[{ required: true }]} />
            <ProFormDigit name="sort" label="排序" initialValue={0} />
            <ProFormDigit name="parent_id" label="父级ID" initialValue={0} />
            <ProFormDigit name="level" label="层级" initialValue={1} />
            <ProFormSelect name="status" label="状态" initialValue={1} options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
          </ModalForm>,
        ]}
        request={async () => {
          const data = await listCategories()
          return { data: data ?? [], success: true }
        }}
        pagination={false}
      />
      <ModalForm<CreateCategoryReq>
        title="编辑分类"
        open={modalOpen}
        initialValues={editItem ?? {}}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editItem) {
            await updateCategory(editItem.id, values)
            message.success('更新成功')
            actionRef.current?.reload()
          }
          return true
        }}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormDigit name="sort" label="排序" />
        <ProFormDigit name="parent_id" label="父级ID" />
        <ProFormDigit name="level" label="层级" />
        <ProFormSelect name="status" label="状态" options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
      </ModalForm>
    </>
  )
}
```

**Step 4: Create BrandList**

Create `merchant-frontend/src/pages/product/BrandList.tsx`:

```typescript
import { useRef, useState } from 'react'
import { message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable, ModalForm, ProFormText, ProFormSelect } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listBrands, createBrand, updateBrand } from '@/api/product'
import type { Brand, CreateBrandReq } from '@/types/product'

export default function BrandList() {
  const actionRef = useRef<ActionType>()
  const [editItem, setEditItem] = useState<Brand | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const columns: ProColumns<Brand>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: 'Logo', dataIndex: 'logo', valueType: 'image', width: 80, search: false },
    { title: '品牌名称', dataIndex: 'name' },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      valueEnum: { 0: { text: '禁用', status: 'Default' }, 1: { text: '启用', status: 'Success' } },
    },
    {
      title: '操作',
      width: 100,
      search: false,
      render: (_, record) => (
        <a onClick={() => { setEditItem(record); setModalOpen(true) }}>编辑</a>
      ),
    },
  ]

  return (
    <>
      <ProTable<Brand>
        headerTitle="品牌管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        toolBarRender={() => [
          <ModalForm<CreateBrandReq>
            key="add"
            title="新增品牌"
            trigger={<a><PlusOutlined /> 新增品牌</a>}
            onFinish={async (values) => {
              await createBrand(values)
              message.success('创建成功')
              actionRef.current?.reload()
              return true
            }}
          >
            <ProFormText name="name" label="品牌名称" rules={[{ required: true }]} />
            <ProFormText name="logo" label="Logo URL" />
            <ProFormSelect name="status" label="状态" initialValue={1} options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
          </ModalForm>,
        ]}
        request={async (params) => {
          const res = await listBrands({ page: params.current, pageSize: params.pageSize })
          return { data: res?.brands ?? [], total: res?.total ?? 0, success: true }
        }}
        pagination={{ defaultPageSize: 20 }}
      />
      <ModalForm<CreateBrandReq>
        title="编辑品牌"
        open={modalOpen}
        initialValues={editItem ?? {}}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editItem) {
            await updateBrand(editItem.id, values)
            message.success('更新成功')
            actionRef.current?.reload()
          }
          return true
        }}
      >
        <ProFormText name="name" label="品牌名称" rules={[{ required: true }]} />
        <ProFormText name="logo" label="Logo URL" />
        <ProFormSelect name="status" label="状态" options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
      </ModalForm>
    </>
  )
}
```

**Step 5: Verify build**

```bash
cd merchant-frontend && npm run build
```

**Step 6: Commit**

```bash
git add merchant-frontend/src/pages/product/
git commit -m "feat(merchant): add product management pages (list, form, category, brand)"
```

---

## Task 8: Order Management Pages

**Files:**
- Create: `merchant-frontend/src/pages/order/OrderList.tsx`
- Create: `merchant-frontend/src/pages/order/OrderDetail.tsx`
- Create: `merchant-frontend/src/pages/order/RefundList.tsx`

**Step 1: Create OrderList**

Create `merchant-frontend/src/pages/order/OrderList.tsx`:

```typescript
import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Tag } from 'antd'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listOrders } from '@/api/order'
import type { Order } from '@/types/order'

const statusMap: Record<number, { text: string; color: string }> = {
  0: { text: '已取消', color: 'default' },
  1: { text: '待付款', color: 'orange' },
  2: { text: '待发货', color: 'blue' },
  3: { text: '已发货', color: 'cyan' },
  4: { text: '已完成', color: 'green' },
  5: { text: '退款中', color: 'red' },
}

export default function OrderList() {
  const navigate = useNavigate()
  const actionRef = useRef<ActionType>()

  const columns: ProColumns<Order>[] = [
    { title: '订单号', dataIndex: 'order_no', copyable: true },
    {
      title: '金额',
      dataIndex: 'pay_amount',
      search: false,
      render: (_, r) => `¥${((r.pay_amount ?? 0) / 100).toFixed(2)}`,
    },
    { title: '收货人', dataIndex: 'receiver_name', search: false },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: Object.fromEntries(Object.entries(statusMap).map(([k, v]) => [k, { text: v.text }])),
      render: (_, r) => {
        const s = statusMap[r.status] ?? { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
    },
    { title: '下单时间', dataIndex: 'created_at', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => (
        <a onClick={() => navigate(`/order/${r.order_no}`)}>详情</a>
      ),
    },
  ]

  return (
    <ProTable<Order>
      headerTitle="订单列表"
      actionRef={actionRef}
      rowKey="order_no"
      columns={columns}
      request={async (params) => {
        const res = await listOrders({ status: params.status, page: params.current, pageSize: params.pageSize })
        return { data: res?.orders ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
```

**Step 2: Create OrderDetail**

Create `merchant-frontend/src/pages/order/OrderDetail.tsx`:

```typescript
import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Descriptions, Table, Tag, Button, Space, message, Modal, Input } from 'antd'
import { getOrder } from '@/api/order'
import { shipOrder, getOrderLogistics } from '@/api/logistics'
import type { Order } from '@/types/order'
import type { Shipment, ShipOrderReq } from '@/types/logistics'

export default function OrderDetail() {
  const { orderNo } = useParams<{ orderNo: string }>()
  const navigate = useNavigate()
  const [order, setOrder] = useState<Order | null>(null)
  const [logistics, setLogistics] = useState<Shipment | null>(null)
  const [shipModal, setShipModal] = useState(false)
  const [shipForm, setShipForm] = useState<ShipOrderReq>({ carrier_code: '', carrier_name: '', tracking_no: '' })
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (orderNo) {
      getOrder(orderNo).then(setOrder).catch(() => {})
      getOrderLogistics(orderNo).then(setLogistics).catch(() => {})
    }
  }, [orderNo])

  const handleShip = async () => {
    if (!orderNo || !shipForm.tracking_no) {
      message.warning('请填写运单号')
      return
    }
    setLoading(true)
    try {
      await shipOrder(orderNo, shipForm)
      message.success('发货成功')
      setShipModal(false)
      getOrder(orderNo).then(setOrder).catch(() => {})
    } catch (e: unknown) {
      message.error((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  if (!order) return null

  return (
    <div>
      <Card title="订单信息" extra={<Button onClick={() => navigate(-1)}>返回</Button>}>
        <Descriptions column={2}>
          <Descriptions.Item label="订单号">{order.order_no}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag>{order.status}</Tag></Descriptions.Item>
          <Descriptions.Item label="支付金额">¥{((order.pay_amount ?? 0) / 100).toFixed(2)}</Descriptions.Item>
          <Descriptions.Item label="运费">¥{((order.freight_amount ?? 0) / 100).toFixed(2)}</Descriptions.Item>
          <Descriptions.Item label="收货人">{order.receiver_name}</Descriptions.Item>
          <Descriptions.Item label="联系电话">{order.receiver_phone}</Descriptions.Item>
          <Descriptions.Item label="收货地址" span={2}>{order.receiver_address}</Descriptions.Item>
          <Descriptions.Item label="备注">{order.remark || '-'}</Descriptions.Item>
          <Descriptions.Item label="下单时间">{order.created_at}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="商品明细" style={{ marginTop: 16 }}>
        <Table
          dataSource={order.items ?? []}
          rowKey="id"
          pagination={false}
          columns={[
            { title: '商品', dataIndex: 'product_name' },
            { title: '规格', dataIndex: 'spec_values' },
            { title: '单价', dataIndex: 'price', render: (v: number) => `¥${((v ?? 0) / 100).toFixed(2)}` },
            { title: '数量', dataIndex: 'quantity' },
          ]}
        />
      </Card>

      {logistics && (
        <Card title="物流信息" style={{ marginTop: 16 }}>
          <Descriptions>
            <Descriptions.Item label="物流公司">{logistics.carrier_name}</Descriptions.Item>
            <Descriptions.Item label="运单号">{logistics.tracking_no}</Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {order.status === 2 && (
        <Card style={{ marginTop: 16 }}>
          <Space>
            <Button type="primary" onClick={() => setShipModal(true)}>发货</Button>
          </Space>
        </Card>
      )}

      <Modal title="发货" open={shipModal} onOk={handleShip} confirmLoading={loading} onCancel={() => setShipModal(false)}>
        <Input placeholder="物流公司编码" value={shipForm.carrier_code} onChange={(e) => setShipForm({ ...shipForm, carrier_code: e.target.value })} style={{ marginBottom: 12 }} />
        <Input placeholder="物流公司名称" value={shipForm.carrier_name} onChange={(e) => setShipForm({ ...shipForm, carrier_name: e.target.value })} style={{ marginBottom: 12 }} />
        <Input placeholder="运单号" value={shipForm.tracking_no} onChange={(e) => setShipForm({ ...shipForm, tracking_no: e.target.value })} />
      </Modal>
    </div>
  )
}
```

**Step 3: Create RefundList**

Create `merchant-frontend/src/pages/order/RefundList.tsx`:

```typescript
import { useRef } from 'react'
import { Tag, Button, Popconfirm, message } from 'antd'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listRefunds, handleRefund } from '@/api/order'
import type { RefundOrder } from '@/types/order'

export default function RefundList() {
  const actionRef = useRef<ActionType>()

  const handleAction = async (orderNo: string, refundNo: string, approved: boolean) => {
    try {
      await handleRefund(orderNo, { refund_no: refundNo, approved, reason: approved ? '同意退款' : '拒绝退款' })
      message.success(approved ? '已同意退款' : '已拒绝退款')
      actionRef.current?.reload()
    } catch (e: unknown) {
      message.error((e as Error).message)
    }
  }

  const columns: ProColumns<RefundOrder>[] = [
    { title: '退款单号', dataIndex: 'refund_no', copyable: true },
    { title: '订单号', dataIndex: 'order_no', copyable: true },
    { title: '退款金额', dataIndex: 'amount', search: false, render: (_, r) => `¥${((r.amount ?? 0) / 100).toFixed(2)}` },
    { title: '原因', dataIndex: 'reason', search: false, ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: { 1: { text: '待处理', status: 'Warning' }, 2: { text: '已同意', status: 'Success' }, 3: { text: '已拒绝', status: 'Error' } },
      render: (_, r) => {
        const map: Record<number, { text: string; color: string }> = { 1: { text: '待处理', color: 'orange' }, 2: { text: '已同意', color: 'green' }, 3: { text: '已拒绝', color: 'red' } }
        const s = map[r.status] ?? { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
    },
    { title: '申请时间', dataIndex: 'created_at', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) =>
        r.status === 1 ? (
          <>
            <Popconfirm title="确认同意退款？" onConfirm={() => handleAction(r.order_no, r.refund_no, true)}>
              <Button type="link" size="small">同意</Button>
            </Popconfirm>
            <Popconfirm title="确认拒绝退款？" onConfirm={() => handleAction(r.order_no, r.refund_no, false)}>
              <Button type="link" size="small" danger>拒绝</Button>
            </Popconfirm>
          </>
        ) : '-',
    },
  ]

  return (
    <ProTable<RefundOrder>
      headerTitle="退款管理"
      actionRef={actionRef}
      rowKey="refund_no"
      columns={columns}
      request={async (params) => {
        const res = await listRefunds({ status: params.status, page: params.current, pageSize: params.pageSize })
        return { data: res?.refund_orders ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
```

**Step 4: Commit**

```bash
git add merchant-frontend/src/pages/order/
git commit -m "feat(merchant): add order management pages (list, detail, refund)"
```

---

## Task 9: Inventory Management Pages

**Files:**
- Create: `merchant-frontend/src/pages/inventory/StockList.tsx`
- Create: `merchant-frontend/src/pages/inventory/StockLog.tsx`

**Step 1: Create StockList**

Create `merchant-frontend/src/pages/inventory/StockList.tsx`:

```typescript
import { useRef, useState } from 'react'
import { Button, InputNumber, message, Space } from 'antd'
import { ProTable, ModalForm, ProFormDigit } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listProducts } from '@/api/product'
import { batchGetStock, setStock } from '@/api/inventory'
import type { Product } from '@/types/product'
import type { Inventory } from '@/types/inventory'

interface StockRow {
  sku_id: number
  sku_code: string
  product_name: string
  spec_values: string
  total: number
  locked: number
  available: number
  alert_threshold: number
}

export default function StockList() {
  const actionRef = useRef<ActionType>()
  const [editSku, setEditSku] = useState<StockRow | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const columns: ProColumns<StockRow>[] = [
    { title: 'SKU ID', dataIndex: 'sku_id', width: 80 },
    { title: '商品', dataIndex: 'product_name', ellipsis: true },
    { title: 'SKU编码', dataIndex: 'sku_code' },
    { title: '规格', dataIndex: 'spec_values' },
    { title: '总库存', dataIndex: 'total', search: false },
    { title: '锁定', dataIndex: 'locked', search: false },
    { title: '可用', dataIndex: 'available', search: false },
    { title: '预警值', dataIndex: 'alert_threshold', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => (
        <a onClick={() => { setEditSku(r); setModalOpen(true) }}>设置库存</a>
      ),
    },
  ]

  return (
    <>
      <ProTable<StockRow>
        headerTitle="库存管理"
        actionRef={actionRef}
        rowKey="sku_id"
        columns={columns}
        search={false}
        request={async (params) => {
          const productRes = await listProducts({ page: params.current, pageSize: params.pageSize })
          const products = productRes?.products ?? []
          const allSkus = products.flatMap((p: Product) =>
            (p.skus ?? []).map((s) => ({ ...s, product_name: p.name }))
          )
          if (allSkus.length === 0) return { data: [], total: 0, success: true }
          const stocks = await batchGetStock(allSkus.map((s) => s.id)).catch(() => [] as Inventory[])
          const stockMap = new Map((stocks ?? []).map((s) => [s.sku_id, s]))
          const rows: StockRow[] = allSkus.map((s) => {
            const inv = stockMap.get(s.id)
            return {
              sku_id: s.id,
              sku_code: s.sku_code,
              product_name: s.product_name,
              spec_values: s.spec_values,
              total: inv?.total ?? 0,
              locked: inv?.locked ?? 0,
              available: inv?.available ?? 0,
              alert_threshold: inv?.alert_threshold ?? 0,
            }
          })
          return { data: rows, total: productRes?.total ?? 0, success: true }
        }}
        pagination={{ defaultPageSize: 20 }}
      />
      <ModalForm
        title="设置库存"
        open={modalOpen}
        initialValues={editSku ? { total: editSku.total, alert_threshold: editSku.alert_threshold } : {}}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editSku) {
            await setStock({ sku_id: editSku.sku_id, total: values.total, alert_threshold: values.alert_threshold })
            message.success('设置成功')
            actionRef.current?.reload()
          }
          return true
        }}
      >
        <ProFormDigit name="total" label="总库存" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="alert_threshold" label="预警阈值" min={0} />
      </ModalForm>
    </>
  )
}
```

**Step 2: Create StockLog**

Create `merchant-frontend/src/pages/inventory/StockLog.tsx`:

```typescript
import { ProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
import { listInventoryLogs } from '@/api/inventory'
import type { InventoryLog } from '@/types/inventory'

export default function StockLog() {
  const columns: ProColumns<InventoryLog>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: 'SKU ID', dataIndex: 'sku_id' },
    { title: '变更类型', dataIndex: 'change_type' },
    { title: '变更数量', dataIndex: 'change_amount', search: false },
    { title: '变更前', dataIndex: 'before_total', search: false },
    { title: '变更后', dataIndex: 'after_total', search: false },
    { title: '关联订单', dataIndex: 'order_no', search: false },
    { title: '时间', dataIndex: 'created_at', valueType: 'dateTime', search: false },
  ]

  return (
    <ProTable<InventoryLog>
      headerTitle="库存变更日志"
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listInventoryLogs({ sku_id: params.sku_id, page: params.current, pageSize: params.pageSize })
        return { data: res?.logs ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
```

**Step 3: Commit**

```bash
git add merchant-frontend/src/pages/inventory/
git commit -m "feat(merchant): add inventory management pages (stock list, logs)"
```

---

## Task 10: Marketing Management Pages

**Files:**
- Create: `merchant-frontend/src/pages/marketing/CouponList.tsx`
- Create: `merchant-frontend/src/pages/marketing/CouponForm.tsx`
- Create: `merchant-frontend/src/pages/marketing/SeckillList.tsx`
- Create: `merchant-frontend/src/pages/marketing/SeckillForm.tsx`
- Create: `merchant-frontend/src/pages/marketing/PromotionList.tsx`

**Step 1: Create CouponList**

Create `merchant-frontend/src/pages/marketing/CouponList.tsx`:

```typescript
import { useNavigate } from 'react-router-dom'
import { Button, Tag } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
import { listCoupons } from '@/api/marketing'
import type { Coupon } from '@/types/marketing'

const typeMap: Record<number, string> = { 1: '满减', 2: '折扣', 3: '固定金额' }

export default function CouponList() {
  const navigate = useNavigate()

  const columns: ProColumns<Coupon>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '名称', dataIndex: 'name' },
    { title: '类型', dataIndex: 'type', search: false, render: (_, r) => <Tag>{typeMap[r.type] ?? '未知'}</Tag> },
    { title: '门槛(分)', dataIndex: 'threshold', search: false },
    { title: '优惠值(分)', dataIndex: 'discount_value', search: false },
    { title: '总数', dataIndex: 'total_count', search: false },
    { title: '已用', dataIndex: 'used_count', search: false },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: { 0: { text: '禁用' }, 1: { text: '启用' } },
    },
    { title: '开始时间', dataIndex: 'start_time', valueType: 'dateTime', search: false },
    { title: '结束时间', dataIndex: 'end_time', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => <a onClick={() => navigate(`/marketing/coupon/edit/${r.id}`)}>编辑</a>,
    },
  ]

  return (
    <ProTable<Coupon>
      headerTitle="优惠券管理"
      rowKey="id"
      columns={columns}
      toolBarRender={() => [
        <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => navigate('/marketing/coupon/create')}>
          创建优惠券
        </Button>,
      ]}
      request={async (params) => {
        const res = await listCoupons({ status: params.status, page: params.current, pageSize: params.pageSize })
        return { data: res?.coupons ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
```

**Step 2: Create CouponForm**

Create `merchant-frontend/src/pages/marketing/CouponForm.tsx`:

```typescript
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormDigit, ProFormSelect, ProFormDateTimePicker } from '@ant-design/pro-components'
import { createCoupon, updateCoupon } from '@/api/marketing'
import type { CreateCouponReq } from '@/types/marketing'

export default function CouponForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id

  return (
    <Card title={isEdit ? '编辑优惠券' : '创建优惠券'}>
      <ProForm<CreateCouponReq>
        onFinish={async (values) => {
          try {
            if (isEdit) {
              await updateCoupon(Number(id), values)
              message.success('更新成功')
            } else {
              await createCoupon(values)
              message.success('创建成功')
            }
            navigate('/marketing/coupon')
          } catch (e: unknown) {
            message.error((e as Error).message)
          }
        }}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormSelect name="type" label="类型" rules={[{ required: true }]} options={[
          { label: '满减', value: 1 },
          { label: '折扣', value: 2 },
          { label: '固定金额', value: 3 },
        ]} />
        <ProFormDigit name="threshold" label="使用门槛（分）" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="discount_value" label="优惠值（分）" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="total_count" label="发放总量" rules={[{ required: true }]} min={1} />
        <ProFormDigit name="per_limit" label="每人限领" initialValue={1} min={1} />
        <ProFormDateTimePicker name="start_time" label="开始时间" rules={[{ required: true }]} />
        <ProFormDateTimePicker name="end_time" label="结束时间" rules={[{ required: true }]} />
        <ProFormSelect name="status" label="状态" initialValue={1} options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
      </ProForm>
    </Card>
  )
}
```

**Step 3: Create SeckillList**

Create `merchant-frontend/src/pages/marketing/SeckillList.tsx`:

```typescript
import { useNavigate } from 'react-router-dom'
import { Button, Tag } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
import { listSeckill } from '@/api/marketing'
import type { SeckillActivity } from '@/types/marketing'

export default function SeckillList() {
  const navigate = useNavigate()

  const columns: ProColumns<SeckillActivity>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '活动名称', dataIndex: 'name' },
    { title: '商品数', dataIndex: 'items', search: false, render: (_, r) => r.items?.length ?? 0 },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: { 0: { text: '未开始' }, 1: { text: '进行中' }, 2: { text: '已结束' } },
    },
    { title: '开始时间', dataIndex: 'start_time', valueType: 'dateTime', search: false },
    { title: '结束时间', dataIndex: 'end_time', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => <a onClick={() => navigate(`/marketing/seckill/edit/${r.id}`)}>编辑</a>,
    },
  ]

  return (
    <ProTable<SeckillActivity>
      headerTitle="秒杀活动"
      rowKey="id"
      columns={columns}
      toolBarRender={() => [
        <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => navigate('/marketing/seckill/create')}>
          创建秒杀
        </Button>,
      ]}
      request={async (params) => {
        const res = await listSeckill({ status: params.status, page: params.current, pageSize: params.pageSize })
        return { data: res?.activities ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
```

**Step 4: Create SeckillForm**

Create `merchant-frontend/src/pages/marketing/SeckillForm.tsx`:

```typescript
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormSelect, ProFormDateTimePicker } from '@ant-design/pro-components'
import { createSeckill, updateSeckill } from '@/api/marketing'
import type { CreateSeckillReq } from '@/types/marketing'

export default function SeckillForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id

  return (
    <Card title={isEdit ? '编辑秒杀活动' : '创建秒杀活动'}>
      <ProForm
        onFinish={async (values: Record<string, unknown>) => {
          const data: CreateSeckillReq = {
            name: values.name as string,
            start_time: values.start_time as string,
            end_time: values.end_time as string,
            status: values.status as number,
            items: [],
          }
          try {
            if (isEdit) {
              await updateSeckill(Number(id), data)
              message.success('更新成功')
            } else {
              await createSeckill(data)
              message.success('创建成功')
            }
            navigate('/marketing/seckill')
          } catch (e: unknown) {
            message.error((e as Error).message)
          }
        }}
      >
        <ProFormText name="name" label="活动名称" rules={[{ required: true }]} />
        <ProFormDateTimePicker name="start_time" label="开始时间" rules={[{ required: true }]} />
        <ProFormDateTimePicker name="end_time" label="结束时间" rules={[{ required: true }]} />
        <ProFormSelect name="status" label="状态" initialValue={0} options={[
          { label: '未开始', value: 0 },
          { label: '进行中', value: 1 },
        ]} />
      </ProForm>
    </Card>
  )
}
```

**Step 5: Create PromotionList**

Create `merchant-frontend/src/pages/marketing/PromotionList.tsx`:

```typescript
import { useRef, useState } from 'react'
import { message, Tag } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable, ModalForm, ProFormText, ProFormDigit, ProFormSelect, ProFormDateTimePicker } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listPromotions, createPromotion, updatePromotion } from '@/api/marketing'
import type { PromotionRule, CreatePromotionReq } from '@/types/marketing'

export default function PromotionList() {
  const actionRef = useRef<ActionType>()
  const [editItem, setEditItem] = useState<PromotionRule | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const columns: ProColumns<PromotionRule>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '名称', dataIndex: 'name' },
    { title: '类型', dataIndex: 'type', render: (_, r) => <Tag>{r.type === 1 ? '满减' : '满赠'}</Tag> },
    { title: '门槛(分)', dataIndex: 'threshold' },
    { title: '优惠值(分)', dataIndex: 'discount_value' },
    { title: '开始时间', dataIndex: 'start_time', valueType: 'dateTime' },
    { title: '结束时间', dataIndex: 'end_time', valueType: 'dateTime' },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: { 0: { text: '禁用' }, 1: { text: '启用' } },
    },
    {
      title: '操作',
      render: (_, r) => <a onClick={() => { setEditItem(r); setModalOpen(true) }}>编辑</a>,
    },
  ]

  return (
    <>
      <ProTable<PromotionRule>
        headerTitle="促销规则"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        toolBarRender={() => [
          <ModalForm<CreatePromotionReq>
            key="add"
            title="创建促销"
            trigger={<a><PlusOutlined /> 创建促销</a>}
            onFinish={async (values) => {
              await createPromotion(values)
              message.success('创建成功')
              actionRef.current?.reload()
              return true
            }}
          >
            <ProFormText name="name" label="名称" rules={[{ required: true }]} />
            <ProFormSelect name="type" label="类型" rules={[{ required: true }]} options={[{ label: '满减', value: 1 }, { label: '满赠', value: 2 }]} />
            <ProFormDigit name="threshold" label="门槛(分)" rules={[{ required: true }]} min={0} />
            <ProFormDigit name="discount_value" label="优惠值(分)" rules={[{ required: true }]} min={0} />
            <ProFormDateTimePicker name="start_time" label="开始时间" rules={[{ required: true }]} />
            <ProFormDateTimePicker name="end_time" label="结束时间" rules={[{ required: true }]} />
            <ProFormSelect name="status" label="状态" initialValue={1} options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
          </ModalForm>,
        ]}
        request={async () => {
          const data = await listPromotions()
          return { data: data ?? [], success: true }
        }}
        pagination={false}
      />
      <ModalForm<CreatePromotionReq>
        title="编辑促销"
        open={modalOpen}
        initialValues={editItem ?? {}}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editItem) {
            await updatePromotion(editItem.id, values)
            message.success('更新成功')
            actionRef.current?.reload()
          }
          return true
        }}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormSelect name="type" label="类型" options={[{ label: '满减', value: 1 }, { label: '满赠', value: 2 }]} />
        <ProFormDigit name="threshold" label="门槛(分)" min={0} />
        <ProFormDigit name="discount_value" label="优惠值(分)" min={0} />
        <ProFormDateTimePicker name="start_time" label="开始时间" />
        <ProFormDateTimePicker name="end_time" label="结束时间" />
        <ProFormSelect name="status" label="状态" options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
      </ModalForm>
    </>
  )
}
```

**Step 6: Commit**

```bash
git add merchant-frontend/src/pages/marketing/
git commit -m "feat(merchant): add marketing pages (coupon, seckill, promotion)"
```

---

## Task 11: Logistics, Shop, Staff, Notification Pages

**Files:**
- Create: `merchant-frontend/src/pages/logistics/TemplateList.tsx`
- Create: `merchant-frontend/src/pages/logistics/TemplateForm.tsx`
- Create: `merchant-frontend/src/pages/shop/ShopSettings.tsx`
- Create: `merchant-frontend/src/pages/staff/StaffList.tsx`
- Create: `merchant-frontend/src/pages/staff/RoleList.tsx`
- Create: `merchant-frontend/src/pages/notification/NotificationList.tsx`

**Step 1: Create TemplateList**

Create `merchant-frontend/src/pages/logistics/TemplateList.tsx`:

```typescript
import { useNavigate } from 'react-router-dom'
import { Button, Popconfirm, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
import { listFreightTemplates, deleteFreightTemplate } from '@/api/logistics'
import type { FreightTemplate } from '@/types/logistics'

const chargeTypeMap: Record<number, string> = { 1: '按重量', 2: '按件数' }

export default function TemplateList() {
  const navigate = useNavigate()

  const columns: ProColumns<FreightTemplate>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '模板名称', dataIndex: 'name' },
    { title: '计费方式', dataIndex: 'charge_type', render: (_, r) => chargeTypeMap[r.charge_type] ?? '未知' },
    { title: '免邮门槛(分)', dataIndex: 'free_threshold' },
    { title: '规则数', dataIndex: 'rules', render: (_, r) => r.rules?.length ?? 0 },
    { title: '创建时间', dataIndex: 'created_at', valueType: 'dateTime' },
    {
      title: '操作',
      render: (_, r) => (
        <>
          <a onClick={() => navigate(`/logistics/template/edit/${r.id}`)}>编辑</a>
          <Popconfirm title="确认删除？" onConfirm={async () => {
            await deleteFreightTemplate(r.id)
            message.success('已删除')
          }}>
            <a style={{ marginLeft: 8, color: 'red' }}>删除</a>
          </Popconfirm>
        </>
      ),
    },
  ]

  return (
    <ProTable<FreightTemplate>
      headerTitle="运费模板"
      rowKey="id"
      columns={columns}
      search={false}
      toolBarRender={() => [
        <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => navigate('/logistics/template/create')}>
          创建模板
        </Button>,
      ]}
      request={async () => {
        const data = await listFreightTemplates()
        return { data: data ?? [], success: true }
      }}
      pagination={false}
    />
  )
}
```

**Step 2: Create TemplateForm**

Create `merchant-frontend/src/pages/logistics/TemplateForm.tsx`:

```typescript
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormDigit, ProFormSelect } from '@ant-design/pro-components'
import { createFreightTemplate, updateFreightTemplate, getFreightTemplate } from '@/api/logistics'
import type { CreateFreightTemplateReq } from '@/types/logistics'

export default function TemplateForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id
  const [initialValues, setInitialValues] = useState<Record<string, unknown>>({})

  useEffect(() => {
    if (isEdit) {
      getFreightTemplate(Number(id)).then((t) => {
        if (t) setInitialValues({ name: t.name, charge_type: t.charge_type, free_threshold: t.free_threshold })
      }).catch(() => {})
    }
  }, [id, isEdit])

  return (
    <Card title={isEdit ? '编辑运费模板' : '创建运费模板'}>
      <ProForm<CreateFreightTemplateReq>
        initialValues={initialValues}
        onFinish={async (values) => {
          const data = { ...values, rules: [] }
          try {
            if (isEdit) {
              await updateFreightTemplate(Number(id), data)
              message.success('更新成功')
            } else {
              await createFreightTemplate(data)
              message.success('创建成功')
            }
            navigate('/logistics/template')
          } catch (e: unknown) {
            message.error((e as Error).message)
          }
        }}
      >
        <ProFormText name="name" label="模板名称" rules={[{ required: true }]} />
        <ProFormSelect name="charge_type" label="计费方式" rules={[{ required: true }]} options={[
          { label: '按重量', value: 1 },
          { label: '按件数', value: 2 },
        ]} />
        <ProFormDigit name="free_threshold" label="免邮门槛（分）" initialValue={0} min={0} />
      </ProForm>
    </Card>
  )
}
```

**Step 3: Create ShopSettings**

Create `merchant-frontend/src/pages/shop/ShopSettings.tsx`:

```typescript
import { useEffect, useState } from 'react'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormTextArea } from '@ant-design/pro-components'
import { getShop, updateShop } from '@/api/shop'
import type { Shop, UpdateShopReq } from '@/types/shop'

export default function ShopSettings() {
  const [shop, setShop] = useState<Shop | null>(null)

  useEffect(() => {
    getShop().then(setShop).catch(() => {})
  }, [])

  return (
    <Card title="店铺设置">
      {shop && (
        <ProForm<UpdateShopReq>
          initialValues={{
            name: shop.name,
            logo: shop.logo,
            description: shop.description,
            subdomain: shop.subdomain,
            custom_domain: shop.custom_domain,
          }}
          onFinish={async (values) => {
            try {
              await updateShop(values)
              message.success('保存成功')
            } catch (e: unknown) {
              message.error((e as Error).message)
            }
          }}
        >
          <ProFormText name="name" label="店铺名称" rules={[{ required: true }]} />
          <ProFormText name="logo" label="Logo URL" />
          <ProFormTextArea name="description" label="店铺描述" />
          <ProFormText name="subdomain" label="子域名" />
          <ProFormText name="custom_domain" label="自定义域名" />
        </ProForm>
      )}
    </Card>
  )
}
```

**Step 4: Create StaffList**

Create `merchant-frontend/src/pages/staff/StaffList.tsx`:

```typescript
import { useRef, useState, useEffect } from 'react'
import { message, Select } from 'antd'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listStaff, assignRole, listRoles } from '@/api/staff'
import type { User, Role } from '@/types/user'

export default function StaffList() {
  const actionRef = useRef<ActionType>()
  const [roles, setRoles] = useState<Role[]>([])

  useEffect(() => {
    listRoles().then((r) => setRoles(r ?? [])).catch(() => {})
  }, [])

  const handleAssignRole = async (userId: number, roleId: number) => {
    try {
      await assignRole(userId, roleId)
      message.success('角色已分配')
      actionRef.current?.reload()
    } catch (e: unknown) {
      message.error((e as Error).message)
    }
  }

  const columns: ProColumns<User>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '昵称', dataIndex: 'nickname' },
    { title: '手机号', dataIndex: 'phone' },
    { title: '角色', dataIndex: 'role', search: false },
    {
      title: '分配角色',
      search: false,
      render: (_, record) => (
        <Select
          style={{ width: 140 }}
          placeholder="选择角色"
          onChange={(v) => handleAssignRole(record.id, v)}
          options={roles.map((r) => ({ label: r.name, value: r.id }))}
        />
      ),
    },
    { title: '加入时间', dataIndex: 'created_at', valueType: 'dateTime', search: false },
  ]

  return (
    <ProTable<User>
      headerTitle="员工管理"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listStaff({ page: params.current, pageSize: params.pageSize })
        return { data: res?.users ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
```

**Step 5: Create RoleList**

Create `merchant-frontend/src/pages/staff/RoleList.tsx`:

```typescript
import { useRef, useState } from 'react'
import { message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable, ModalForm, ProFormText, ProFormTextArea } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listRoles, createRole, updateRole } from '@/api/staff'
import type { Role, CreateRoleReq } from '@/types/user'

export default function RoleList() {
  const actionRef = useRef<ActionType>()
  const [editItem, setEditItem] = useState<Role | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const columns: ProColumns<Role>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '角色名称', dataIndex: 'name' },
    { title: '角色编码', dataIndex: 'code' },
    { title: '描述', dataIndex: 'description', ellipsis: true },
    {
      title: '操作',
      render: (_, r) => <a onClick={() => { setEditItem(r); setModalOpen(true) }}>编辑</a>,
    },
  ]

  return (
    <>
      <ProTable<Role>
        headerTitle="角色管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        toolBarRender={() => [
          <ModalForm<CreateRoleReq>
            key="add"
            title="新增角色"
            trigger={<a><PlusOutlined /> 新增角色</a>}
            onFinish={async (values) => {
              await createRole(values)
              message.success('创建成功')
              actionRef.current?.reload()
              return true
            }}
          >
            <ProFormText name="name" label="角色名称" rules={[{ required: true }]} />
            <ProFormText name="code" label="角色编码" rules={[{ required: true }]} />
            <ProFormTextArea name="description" label="描述" />
          </ModalForm>,
        ]}
        request={async () => {
          const data = await listRoles()
          return { data: data ?? [], success: true }
        }}
        pagination={false}
      />
      <ModalForm<CreateRoleReq>
        title="编辑角色"
        open={modalOpen}
        initialValues={editItem ?? {}}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editItem) {
            await updateRole(editItem.id, values)
            message.success('更新成功')
            actionRef.current?.reload()
          }
          return true
        }}
      >
        <ProFormText name="name" label="角色名称" rules={[{ required: true }]} />
        <ProFormText name="code" label="角色编码" rules={[{ required: true }]} />
        <ProFormTextArea name="description" label="描述" />
      </ModalForm>
    </>
  )
}
```

**Step 6: Create NotificationList**

Create `merchant-frontend/src/pages/notification/NotificationList.tsx`:

```typescript
import { useRef } from 'react'
import { Button, Tag, message } from 'antd'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listNotifications, markRead, markAllRead } from '@/api/notification'
import { useNotificationStore } from '@/stores/notification'
import type { Notification } from '@/types/notification'

export default function NotificationList() {
  const actionRef = useRef<ActionType>()
  const fetchUnreadCount = useNotificationStore((s) => s.fetchUnreadCount)

  const handleMarkRead = async (id: number) => {
    await markRead(id)
    actionRef.current?.reload()
    fetchUnreadCount()
  }

  const handleMarkAllRead = async () => {
    await markAllRead()
    message.success('全部已读')
    actionRef.current?.reload()
    fetchUnreadCount()
  }

  const columns: ProColumns<Notification>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '标题', dataIndex: 'title' },
    { title: '内容', dataIndex: 'content', ellipsis: true, search: false },
    { title: '渠道', dataIndex: 'channel', search: false },
    {
      title: '状态',
      dataIndex: 'is_read',
      search: false,
      render: (_, r) => r.is_read ? <Tag>已读</Tag> : <Tag color="blue">未读</Tag>,
    },
    { title: '时间', dataIndex: 'created_at', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => !r.is_read ? <a onClick={() => handleMarkRead(r.id)}>标记已读</a> : '-',
    },
  ]

  return (
    <ProTable<Notification>
      headerTitle="消息中心"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      toolBarRender={() => [
        <Button key="readAll" onClick={handleMarkAllRead}>全部已读</Button>,
      ]}
      request={async (params) => {
        const res = await listNotifications({ page: params.current, pageSize: params.pageSize })
        return { data: res?.notifications ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
```

**Step 7: Verify full build**

```bash
cd merchant-frontend && npm run build
```

**Step 8: Commit**

```bash
git add merchant-frontend/src/pages/
git commit -m "feat(merchant): add logistics, shop, staff, notification pages"
```

---

## Task 12: Final Build Verification & Dev Test

**Step 1: Full build**

```bash
cd merchant-frontend && npm run build
```

Expected: Build succeeds with zero errors.

**Step 2: Start dev server and smoke test**

```bash
cd merchant-frontend && npm run dev
```

Open http://localhost:3001 in browser. Verify:
- Login page renders at `/login`
- After login, redirects to dashboard with sidebar layout
- All sidebar menu items navigate correctly
- ProTable pages render without errors

**Step 3: Final commit**

```bash
git add -A merchant-frontend/
git commit -m "feat(merchant): complete merchant management frontend with all modules"
```
