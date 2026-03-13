# Consumer 消费者商城 Phase 1 MVP Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Mobile-First consumer mall frontend (Vite + React + TypeScript) covering the core shopping flow: login → browse → cart → checkout → payment.

**Architecture:** Single-page app with Ant Design Mobile components, Zustand for state, Axios with JWT interceptor for API calls. Pages lazy-loaded via React Router. Vite dev proxy forwards `/api` to consumer-bff with `X-Tenant-Domain` header.

**Tech Stack:** Vite 6, React 19, TypeScript 5, Ant Design Mobile 5, Zustand 5, React Router 7, Axios 1

**Design doc:** `docs/plans/2026-03-12-consumer-frontend-design.md`

---

### Task 1: Project Scaffold

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/tsconfig.app.json`
- Create: `frontend/index.html`
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/App.tsx`
- Create: `frontend/src/vite-env.d.ts`

**Step 1: Initialize Vite project**

```bash
cd /Users/emoji/Documents/demo/project/mall
npm create vite@latest frontend -- --template react-ts
```

**Step 2: Install dependencies**

```bash
cd frontend
npm install antd-mobile@^5 zustand@^5 react-router-dom@^7 axios@^1
npm install -D @types/node
```

**Step 3: Configure Vite with proxy and path aliases**

Replace `frontend/vite.config.ts`:

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
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        headers: { 'X-Tenant-Domain': 'shop1' },
      },
    },
  },
})
```

**Step 4: Configure tsconfig path alias**

Add to `frontend/tsconfig.app.json` compilerOptions:

```json
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  }
}
```

**Step 5: Create global styles**

Create `frontend/src/styles/global.css`:

```css
:root {
  --color-primary: #1A1A1A;
  --color-accent: #C9A96E;
  --color-bg: #F8F8F8;
  --color-card: #FFFFFF;
  --color-text-secondary: #999999;
  --color-border: #EEEEEE;
  --color-success: #52C41A;
  --color-danger: #FF4D4F;
  --radius: 8px;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  font-size: 14px;
  line-height: 1.6;
  color: var(--color-primary);
  background-color: var(--color-bg);
  -webkit-font-smoothing: antialiased;
}

#root {
  min-height: 100vh;
}
```

**Step 6: Update main.tsx**

Replace `frontend/src/main.tsx`:

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './styles/global.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
```

**Step 7: Verify dev server starts**

```bash
cd frontend && npm run dev
```

Expected: Vite dev server starts on port 3000

**Step 8: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): scaffold Vite + React + TS project with deps"
```

---

### Task 2: API Client with JWT Interceptor

**Files:**
- Create: `frontend/src/api/client.ts`
- Create: `frontend/src/types/api.ts`

**Step 1: Create API response types**

Create `frontend/src/types/api.ts`:

```ts
export interface ApiResult<T = unknown> {
  code: number
  msg: string
  data: T
}
```

**Step 2: Create Axios client with interceptors**

Create `frontend/src/api/client.ts`:

```ts
import axios from 'axios'
import type { ApiResult } from '@/types/api'

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

// Request interceptor: attach JWT token
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: handle 401 with token refresh
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

export async function request<T>(config: Parameters<typeof client>[0]): Promise<T> {
  const res = await client(config)
  const body = res.data as ApiResult<T>
  if (body.code !== 0) {
    throw new Error(body.msg || '请求失败')
  }
  return body.data
}

export { client }
export default client
```

**Step 3: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors

**Step 4: Commit**

```bash
git add frontend/src/api/ frontend/src/types/
git commit -m "feat(frontend): add Axios client with JWT refresh interceptor"
```

---

### Task 3: Auth Store + Auth API

**Files:**
- Create: `frontend/src/stores/auth.ts`
- Create: `frontend/src/api/auth.ts`

**Step 1: Create auth API module**

Create `frontend/src/api/auth.ts`:

```ts
import { client } from './client'

export interface LoginParams {
  phone: string
  password: string
}

export interface LoginByPhoneParams {
  phone: string
  code: string
}

export interface SignupParams {
  phone: string
  email: string
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
  extractTokens(res.headers as Record<string, string>)
  return res.data
}

export async function loginByPhone(params: LoginByPhoneParams) {
  const res = await client.post('/login/phone', params)
  extractTokens(res.headers as Record<string, string>)
  return res.data
}

export async function signup(params: SignupParams) {
  const res = await client.post('/signup', params)
  return res.data
}

export async function sendSmsCode(phone: string, scene: number = 1) {
  const res = await client.post('/sms/send', { phone, scene })
  return res.data
}

export async function logout() {
  await client.post('/logout', {})
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}
```

**Step 2: Create auth Zustand store**

Create `frontend/src/stores/auth.ts`:

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

**Step 3: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit
```

**Step 4: Commit**

```bash
git add frontend/src/api/auth.ts frontend/src/stores/auth.ts
git commit -m "feat(frontend): add auth API module and Zustand auth store"
```

---

### Task 4: Router + Layout with TabBar

**Files:**
- Create: `frontend/src/router/index.tsx`
- Create: `frontend/src/components/Layout/TabBarLayout.tsx`
- Create: `frontend/src/components/Layout/TabBarLayout.module.css`
- Create: `frontend/src/components/AuthGuard.tsx`
- Create placeholder pages (7 files)
- Modify: `frontend/src/App.tsx`

**Step 1: Create placeholder pages**

Create each of these with a minimal component:

`frontend/src/pages/home/index.tsx`:
```tsx
export default function HomePage() {
  return <div style={{ padding: 16 }}>首页</div>
}
```

`frontend/src/pages/search/index.tsx`:
```tsx
export default function SearchPage() {
  return <div style={{ padding: 16 }}>搜索</div>
}
```

`frontend/src/pages/cart/index.tsx`:
```tsx
export default function CartPage() {
  return <div style={{ padding: 16 }}>购物车</div>
}
```

`frontend/src/pages/user/index.tsx`:
```tsx
export default function UserPage() {
  return <div style={{ padding: 16 }}>我的</div>
}
```

`frontend/src/pages/auth/Login.tsx`:
```tsx
export default function LoginPage() {
  return <div style={{ padding: 16 }}>登录</div>
}
```

`frontend/src/pages/auth/Signup.tsx`:
```tsx
export default function SignupPage() {
  return <div style={{ padding: 16 }}>注册</div>
}
```

`frontend/src/pages/product/Detail.tsx`:
```tsx
export default function ProductDetail() {
  return <div style={{ padding: 16 }}>商品详情</div>
}
```

**Step 2: Create AuthGuard component**

Create `frontend/src/components/AuthGuard.tsx`:

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

**Step 3: Create TabBar layout**

Create `frontend/src/components/Layout/TabBarLayout.module.css`:

```css
.container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.content {
  flex: 1;
  padding-bottom: 50px;
}

.tabBar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: var(--color-card);
  border-top: 1px solid var(--color-border);
  z-index: 100;
}
```

Create `frontend/src/components/Layout/TabBarLayout.tsx`:

```tsx
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { TabBar } from 'antd-mobile'
import {
  AppOutline,
  SearchOutline,
  ShopbagOutline,
  UserOutline,
} from 'antd-mobile-icons'
import styles from './TabBarLayout.module.css'

const tabs = [
  { key: '/', title: '首页', icon: <AppOutline /> },
  { key: '/search', title: '搜索', icon: <SearchOutline /> },
  { key: '/cart', title: '购物车', icon: <ShopbagOutline /> },
  { key: '/me', title: '我的', icon: <UserOutline /> },
]

export default function TabBarLayout() {
  const location = useLocation()
  const navigate = useNavigate()

  return (
    <div className={styles.container}>
      <div className={styles.content}>
        <Outlet />
      </div>
      <div className={styles.tabBar}>
        <TabBar
          activeKey={location.pathname}
          onChange={(key) => navigate(key)}
        >
          {tabs.map((tab) => (
            <TabBar.Item key={tab.key} icon={tab.icon} title={tab.title} />
          ))}
        </TabBar>
      </div>
    </div>
  )
}
```

**Step 4: Create router config**

Create `frontend/src/router/index.tsx`:

```tsx
import { lazy, Suspense } from 'react'
import { createBrowserRouter } from 'react-router-dom'
import { SpinLoading } from 'antd-mobile'
import TabBarLayout from '@/components/Layout/TabBarLayout'
import AuthGuard from '@/components/AuthGuard'

const HomePage = lazy(() => import('@/pages/home'))
const SearchPage = lazy(() => import('@/pages/search'))
const CartPage = lazy(() => import('@/pages/cart'))
const UserPage = lazy(() => import('@/pages/user'))
const LoginPage = lazy(() => import('@/pages/auth/Login'))
const SignupPage = lazy(() => import('@/pages/auth/Signup'))
const ProductDetail = lazy(() => import('@/pages/product/Detail'))

function Loading() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '40vh 0' }}>
      <SpinLoading color='default' />
    </div>
  )
}

function Lazy({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<Loading />}>{children}</Suspense>
}

export const router = createBrowserRouter([
  {
    element: <TabBarLayout />,
    children: [
      { path: '/', element: <Lazy><HomePage /></Lazy> },
      { path: '/search', element: <Lazy><SearchPage /></Lazy> },
      {
        path: '/cart',
        element: <AuthGuard><Lazy><CartPage /></Lazy></AuthGuard>,
      },
      {
        path: '/me',
        element: <AuthGuard><Lazy><UserPage /></Lazy></AuthGuard>,
      },
    ],
  },
  { path: '/login', element: <Lazy><LoginPage /></Lazy> },
  { path: '/signup', element: <Lazy><SignupPage /></Lazy> },
  {
    path: '/product/:id',
    element: <Lazy><ProductDetail /></Lazy>,
  },
])
```

**Step 5: Update App.tsx**

Replace `frontend/src/App.tsx`:

```tsx
import { RouterProvider } from 'react-router-dom'
import { router } from './router'

export default function App() {
  return <RouterProvider router={router} />
}
```

**Step 6: Verify dev server renders with TabBar**

```bash
cd frontend && npm run dev
```

Open http://localhost:3000, verify 4-tab TabBar renders, navigation works.

**Step 7: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add router, TabBar layout, AuthGuard, placeholder pages"
```

---

### Task 5: Login & Signup Pages

**Files:**
- Modify: `frontend/src/pages/auth/Login.tsx`
- Modify: `frontend/src/pages/auth/Signup.tsx`
- Create: `frontend/src/pages/auth/auth.module.css`

**Step 1: Create shared auth styles**

Create `frontend/src/pages/auth/auth.module.css`:

```css
.page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  padding: 60px 24px 24px;
  background: var(--color-card);
}

.title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-primary);
  margin-bottom: 8px;
}

.subtitle {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: 40px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.submitBtn {
  margin-top: 16px;
  --background-color: var(--color-primary);
  --border-color: var(--color-primary);
  height: 48px;
  font-size: 16px;
  border-radius: var(--radius);
}

.footer {
  margin-top: 24px;
  text-align: center;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.link {
  color: var(--color-accent);
  text-decoration: none;
  font-weight: 500;
}
```

**Step 2: Implement Login page**

Replace `frontend/src/pages/auth/Login.tsx`:

```tsx
import { useState } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { Input, Button, Toast } from 'antd-mobile'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import styles from './auth.module.css'

export default function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const setLoggedIn = useAuthStore((s) => s.setLoggedIn)

  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  const handleLogin = async () => {
    if (!phone || !password) {
      Toast.show('请输入手机号和密码')
      return
    }
    setLoading(true)
    try {
      await login({ phone, password })
      setLoggedIn(true)
      const redirect = searchParams.get('redirect') || '/'
      navigate(redirect, { replace: true })
    } catch (e: unknown) {
      Toast.show((e as Error).message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.title}>欢迎回来</div>
      <div className={styles.subtitle}>登录你的账户继续购物</div>
      <div className={styles.form}>
        <Input
          placeholder='手机号'
          value={phone}
          onChange={setPhone}
          type='tel'
          maxLength={11}
          clearable
        />
        <Input
          placeholder='密码'
          value={password}
          onChange={setPassword}
          type='password'
          clearable
        />
        <Button
          block
          color='primary'
          className={styles.submitBtn}
          loading={loading}
          onClick={handleLogin}
        >
          登录
        </Button>
      </div>
      <div className={styles.footer}>
        还没有账户？<Link to='/signup' className={styles.link}>立即注册</Link>
      </div>
    </div>
  )
}
```

**Step 3: Implement Signup page**

Replace `frontend/src/pages/auth/Signup.tsx`:

```tsx
import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Input, Button, Toast } from 'antd-mobile'
import { signup } from '@/api/auth'
import styles from './auth.module.css'

export default function SignupPage() {
  const navigate = useNavigate()
  const [phone, setPhone] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSignup = async () => {
    if (!phone || !password) {
      Toast.show('请填写必填项')
      return
    }
    setLoading(true)
    try {
      await signup({ phone, email, password })
      Toast.show('注册成功，请登录')
      navigate('/login', { replace: true })
    } catch (e: unknown) {
      Toast.show((e as Error).message || '注册失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.title}>创建账户</div>
      <div className={styles.subtitle}>注册后即可开始购物</div>
      <div className={styles.form}>
        <Input placeholder='手机号' value={phone} onChange={setPhone} type='tel' maxLength={11} clearable />
        <Input placeholder='邮箱（选填）' value={email} onChange={setEmail} type='email' clearable />
        <Input placeholder='密码' value={password} onChange={setPassword} type='password' clearable />
        <Button block color='primary' className={styles.submitBtn} loading={loading} onClick={handleSignup}>
          注册
        </Button>
      </div>
      <div className={styles.footer}>
        已有账户？<Link to='/login' className={styles.link}>去登录</Link>
      </div>
    </div>
  )
}
```

**Step 4: Verify login page renders**

```bash
cd frontend && npm run dev
```

Open http://localhost:3000/login, verify form renders correctly.

**Step 5: Commit**

```bash
git add frontend/src/pages/auth/
git commit -m "feat(frontend): implement Login and Signup pages"
```

---

### Task 6: Search API + Search Page

**Files:**
- Create: `frontend/src/api/search.ts`
- Create: `frontend/src/types/product.ts`
- Create: `frontend/src/components/ProductCard/index.tsx`
- Create: `frontend/src/components/ProductCard/ProductCard.module.css`
- Create: `frontend/src/components/Price/index.tsx`
- Modify: `frontend/src/pages/search/index.tsx`

**Step 1: Create product types**

Create `frontend/src/types/product.ts`:

```ts
export interface Product {
  id: number
  name: string
  description: string
  main_image: string
  images: string[]
  price: number      // lowest SKU price in cents
  original_price: number
  sales: number
  category_id: number
  brand_id: number
  status: number
}
```

**Step 2: Create search API**

Create `frontend/src/api/search.ts`:

```ts
import { request } from './client'
import type { Product } from '@/types/product'

export interface SearchParams {
  keyword?: string
  category_id?: number
  brand_id?: number
  price_min?: number
  price_max?: number
  sort_by?: string
  page?: number
  page_size?: number
}

export interface SearchResult {
  products: Product[]
  total: number
}

export function searchProducts(params: SearchParams) {
  return request<SearchResult>({
    method: 'GET',
    url: '/search',
    params,
  })
}

export function getSuggestions(keyword: string) {
  return request<string[]>({
    method: 'GET',
    url: '/search/suggestions',
    params: { keyword },
  })
}

export function getHotWords() {
  return request<string[]>({
    method: 'GET',
    url: '/search/hot',
  })
}
```

**Step 3: Create Price component**

Create `frontend/src/components/Price/index.tsx`:

```tsx
interface PriceProps {
  value: number       // cents
  original?: number   // cents, optional strikethrough
  size?: 'sm' | 'md' | 'lg'
}

const sizes = { sm: 14, md: 18, lg: 24 }

export default function Price({ value, original, size = 'md' }: PriceProps) {
  const fontSize = sizes[size]
  return (
    <span>
      <span style={{ color: 'var(--color-accent)', fontWeight: 700, fontSize }}>
        ¥{(value / 100).toFixed(2)}
      </span>
      {original && original > value && (
        <span style={{
          color: 'var(--color-text-secondary)',
          textDecoration: 'line-through',
          fontSize: fontSize - 4,
          marginLeft: 4,
        }}>
          ¥{(original / 100).toFixed(2)}
        </span>
      )}
    </span>
  )
}
```

**Step 4: Create ProductCard component**

Create `frontend/src/components/ProductCard/ProductCard.module.css`:

```css
.card {
  background: var(--color-card);
  border-radius: var(--radius);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s;
}

.card:active {
  transform: scale(0.98);
}

.image {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  display: block;
  background: #f5f5f5;
}

.info {
  padding: 8px 12px 12px;
}

.name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-primary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  margin-bottom: 6px;
}

.meta {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.sales {
  font-size: 11px;
  color: var(--color-text-secondary);
}
```

Create `frontend/src/components/ProductCard/index.tsx`:

```tsx
import { useNavigate } from 'react-router-dom'
import Price from '@/components/Price'
import type { Product } from '@/types/product'
import styles from './ProductCard.module.css'

export default function ProductCard({ product }: { product: Product }) {
  const navigate = useNavigate()

  return (
    <div className={styles.card} onClick={() => navigate(`/product/${product.id}`)}>
      <img
        className={styles.image}
        src={product.main_image || 'https://via.placeholder.com/300'}
        alt={product.name}
        loading='lazy'
      />
      <div className={styles.info}>
        <div className={styles.name}>{product.name}</div>
        <div className={styles.meta}>
          <Price value={product.price} original={product.original_price} size='sm' />
          {product.sales > 0 && (
            <span className={styles.sales}>已售{product.sales}</span>
          )}
        </div>
      </div>
    </div>
  )
}
```

**Step 5: Implement Search page**

Create `frontend/src/pages/search/search.module.css`:

```css
.page {
  padding: 12px;
}

.searchBar {
  margin-bottom: 12px;
}

.hotSection {
  padding: 12px 0;
}

.hotTitle {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 12px;
}

.hotTags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.hotTag {
  padding: 6px 14px;
  background: var(--color-card);
  border-radius: 16px;
  font-size: 13px;
  color: var(--color-primary);
  border: 1px solid var(--color-border);
  cursor: pointer;
}

.grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}

.empty {
  text-align: center;
  padding: 60px 0;
  color: var(--color-text-secondary);
}

.loadMore {
  text-align: center;
  padding: 16px;
  color: var(--color-text-secondary);
  font-size: 13px;
}
```

Replace `frontend/src/pages/search/index.tsx`:

```tsx
import { useState, useEffect, useCallback } from 'react'
import { SearchBar, InfiniteScroll } from 'antd-mobile'
import { searchProducts, getHotWords } from '@/api/search'
import ProductCard from '@/components/ProductCard'
import type { Product } from '@/types/product'
import styles from './search.module.css'

export default function SearchPage() {
  const [keyword, setKeyword] = useState('')
  const [products, setProducts] = useState<Product[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [hotWords, setHotWords] = useState<string[]>([])
  const [searched, setSearched] = useState(false)

  useEffect(() => {
    getHotWords().then(setHotWords).catch(() => {})
  }, [])

  const doSearch = useCallback(async (kw: string, pageNum: number = 1) => {
    const res = await searchProducts({ keyword: kw, page: pageNum, page_size: 20 })
    if (pageNum === 1) {
      setProducts(res.products || [])
    } else {
      setProducts((prev) => [...prev, ...(res.products || [])])
    }
    setTotal(res.total || 0)
    setPage(pageNum)
    setSearched(true)
  }, [])

  const handleSearch = (val: string) => {
    setKeyword(val)
    if (val.trim()) {
      doSearch(val.trim())
    }
  }

  const loadMore = async () => {
    if (!keyword.trim()) return
    await doSearch(keyword.trim(), page + 1)
  }

  const hasMore = products.length < total

  return (
    <div className={styles.page}>
      <div className={styles.searchBar}>
        <SearchBar
          placeholder='搜索商品'
          value={keyword}
          onChange={setKeyword}
          onSearch={handleSearch}
          onClear={() => { setSearched(false); setProducts([]) }}
        />
      </div>

      {!searched && hotWords.length > 0 && (
        <div className={styles.hotSection}>
          <div className={styles.hotTitle}>热门搜索</div>
          <div className={styles.hotTags}>
            {hotWords.map((w) => (
              <span key={w} className={styles.hotTag} onClick={() => handleSearch(w)}>
                {w}
              </span>
            ))}
          </div>
        </div>
      )}

      {searched && (
        <>
          <div className={styles.grid}>
            {products.map((p) => (
              <ProductCard key={p.id} product={p} />
            ))}
          </div>
          {products.length === 0 && (
            <div className={styles.empty}>没有找到相关商品</div>
          )}
          <InfiniteScroll loadMore={loadMore} hasMore={hasMore} />
        </>
      )}
    </div>
  )
}
```

**Step 6: Verify search page renders**

```bash
cd frontend && npm run dev
```

Navigate to search tab, verify SearchBar and hot words section renders.

**Step 7: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add search page with ProductCard, Price, infinite scroll"
```

---

### Task 7: Home Page

**Files:**
- Create: `frontend/src/api/marketing.ts`
- Create: `frontend/src/api/shop.ts`
- Modify: `frontend/src/pages/home/index.tsx`
- Create: `frontend/src/pages/home/home.module.css`

**Step 1: Create shop and marketing API modules**

Create `frontend/src/api/shop.ts`:

```ts
import { request } from './client'

export interface Shop {
  id: number
  tenant_id: number
  name: string
  logo: string
  description: string
  status: number
}

export function getShop() {
  return request<Shop>({ method: 'GET', url: '/shop' })
}
```

Create `frontend/src/api/marketing.ts`:

```ts
import { request } from './client'

export interface SeckillActivity {
  id: number
  name: string
  start_time: string
  end_time: string
  status: number
  items: SeckillItem[]
}

export interface SeckillItem {
  id: number
  product_name: string
  product_image: string
  original_price: number
  seckill_price: number
  total_stock: number
  available_stock: number
}

export interface Coupon {
  id: number
  name: string
  type: number
  value: number
  min_spend: number
  start_time: string
  end_time: string
  total: number
  remaining: number
}

export function listSeckillActivities() {
  return request<SeckillActivity[]>({
    method: 'GET',
    url: '/seckill',
  })
}

export function listAvailableCoupons() {
  return request<Coupon[]>({
    method: 'GET',
    url: '/coupons',
  })
}
```

**Step 2: Create home page styles**

Create `frontend/src/pages/home/home.module.css`:

```css
.page {
  padding-bottom: 16px;
}

.header {
  padding: 16px;
  background: var(--color-card);
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.shopLogo {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.shopName {
  font-size: 18px;
  font-weight: 600;
}

.section {
  margin: 0 12px 16px;
}

.sectionTitle {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 12px;
  padding-left: 4px;
}

.seckillScroll {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  padding-bottom: 4px;
  -webkit-overflow-scrolling: touch;
}

.seckillScroll::-webkit-scrollbar {
  display: none;
}

.seckillCard {
  flex-shrink: 0;
  width: 120px;
  background: var(--color-card);
  border-radius: var(--radius);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  cursor: pointer;
}

.seckillImage {
  width: 120px;
  height: 120px;
  object-fit: cover;
  display: block;
}

.seckillInfo {
  padding: 6px 8px;
}

.seckillName {
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.seckillPrice {
  color: var(--color-danger);
  font-weight: 700;
  font-size: 14px;
}

.seckillOriginal {
  color: var(--color-text-secondary);
  font-size: 11px;
  text-decoration: line-through;
}

.couponScroll {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  padding-bottom: 4px;
  -webkit-overflow-scrolling: touch;
}

.couponScroll::-webkit-scrollbar {
  display: none;
}

.couponCard {
  flex-shrink: 0;
  width: 160px;
  padding: 12px;
  background: linear-gradient(135deg, #FFF8F0, #FFFFFF);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.couponValue {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-accent);
}

.couponName {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.couponCondition {
  font-size: 11px;
  color: var(--color-text-secondary);
  margin-top: 2px;
}
```

**Step 3: Implement Home page**

Replace `frontend/src/pages/home/index.tsx`:

```tsx
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { getShop, type Shop } from '@/api/shop'
import {
  listSeckillActivities,
  listAvailableCoupons,
  type SeckillActivity,
  type Coupon,
} from '@/api/marketing'
import styles from './home.module.css'

export default function HomePage() {
  const navigate = useNavigate()
  const [shop, setShop] = useState<Shop | null>(null)
  const [seckills, setSeckills] = useState<SeckillActivity[]>([])
  const [coupons, setCoupons] = useState<Coupon[]>([])

  useEffect(() => {
    getShop().then(setShop).catch(() => {})
    listSeckillActivities().then(setSeckills).catch(() => {})
    listAvailableCoupons().then(setCoupons).catch(() => {})
  }, [])

  const allSeckillItems = seckills.flatMap((s) => s.items || [])

  return (
    <div className={styles.page}>
      {shop && (
        <div className={styles.header}>
          {shop.logo && (
            <img className={styles.shopLogo} src={shop.logo} alt={shop.name} />
          )}
          <div className={styles.shopName}>{shop.name}</div>
        </div>
      )}

      {allSeckillItems.length > 0 && (
        <div className={styles.section}>
          <div className={styles.sectionTitle}>限时秒杀</div>
          <div className={styles.seckillScroll}>
            {allSeckillItems.map((item) => (
              <div key={item.id} className={styles.seckillCard}>
                <img
                  className={styles.seckillImage}
                  src={item.product_image || 'https://via.placeholder.com/120'}
                  alt={item.product_name}
                />
                <div className={styles.seckillInfo}>
                  <div className={styles.seckillName}>{item.product_name}</div>
                  <div className={styles.seckillPrice}>
                    ¥{(item.seckill_price / 100).toFixed(2)}
                  </div>
                  <div className={styles.seckillOriginal}>
                    ¥{(item.original_price / 100).toFixed(2)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {coupons.length > 0 && (
        <div className={styles.section}>
          <div className={styles.sectionTitle}>优惠券</div>
          <div className={styles.couponScroll}>
            {coupons.map((c) => (
              <div key={c.id} className={styles.couponCard}>
                <div className={styles.couponValue}>
                  {c.type === 1 ? `¥${(c.value / 100).toFixed(0)}` : `${c.value / 10}折`}
                </div>
                <div className={styles.couponName}>{c.name}</div>
                <div className={styles.couponCondition}>
                  满¥{(c.min_spend / 100).toFixed(0)}可用
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className={styles.section}>
        <div className={styles.sectionTitle}>为你推荐</div>
        <div style={{ textAlign: 'center', padding: 40, color: 'var(--color-text-secondary)' }}>
          <span onClick={() => navigate('/search')} style={{ cursor: 'pointer', color: 'var(--color-accent)' }}>
            去搜索发现更多好物 →
          </span>
        </div>
      </div>
    </div>
  )
}
```

**Step 4: Verify home page renders**

```bash
cd frontend && npm run dev
```

**Step 5: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): implement home page with seckill and coupon sections"
```

---

### Task 8: Cart API + Store + Page

**Files:**
- Create: `frontend/src/api/cart.ts`
- Create: `frontend/src/stores/cart.ts`
- Create: `frontend/src/pages/cart/cart.module.css`
- Modify: `frontend/src/pages/cart/index.tsx`

**Step 1: Create cart API**

Create `frontend/src/api/cart.ts`:

```ts
import { request, client } from './client'

export interface CartItem {
  sku_id: number
  product_id: number
  quantity: number
  selected: boolean
  product_name: string
  product_image: string
  sku_spec: string
  price: number
  stock: number
}

export function getCart() {
  return request<CartItem[]>({ method: 'GET', url: '/cart' })
}

export function addCartItem(params: { sku_id: number; product_id: number; quantity: number }) {
  return request<void>({ method: 'POST', url: '/cart/items', data: params })
}

export function updateCartItem(skuId: number, params: { quantity?: number; selected?: boolean; update_selected?: boolean }) {
  return request<void>({ method: 'PUT', url: `/cart/items/${skuId}`, data: params })
}

export function removeCartItem(skuId: number) {
  return client.delete(`/cart/items/${skuId}`)
}

export function clearCart() {
  return client.delete('/cart')
}

export function batchRemove(skuIds: number[]) {
  return request<void>({ method: 'POST', url: '/cart/batch-remove', data: { sku_ids: skuIds } })
}
```

**Step 2: Create cart Zustand store**

Create `frontend/src/stores/cart.ts`:

```ts
import { create } from 'zustand'
import { getCart, updateCartItem, removeCartItem, type CartItem } from '@/api/cart'

interface CartState {
  items: CartItem[]
  loading: boolean
  fetchCart: () => Promise<void>
  toggleSelect: (skuId: number, selected: boolean) => Promise<void>
  updateQuantity: (skuId: number, quantity: number) => Promise<void>
  remove: (skuId: number) => Promise<void>
  selectedItems: () => CartItem[]
  totalAmount: () => number
  totalCount: () => number
}

export const useCartStore = create<CartState>((set, get) => ({
  items: [],
  loading: false,

  fetchCart: async () => {
    set({ loading: true })
    try {
      const items = await getCart()
      set({ items: items || [] })
    } finally {
      set({ loading: false })
    }
  },

  toggleSelect: async (skuId, selected) => {
    await updateCartItem(skuId, { selected, update_selected: true })
    set((s) => ({
      items: s.items.map((i) => (i.sku_id === skuId ? { ...i, selected } : i)),
    }))
  },

  updateQuantity: async (skuId, quantity) => {
    await updateCartItem(skuId, { quantity })
    set((s) => ({
      items: s.items.map((i) => (i.sku_id === skuId ? { ...i, quantity } : i)),
    }))
  },

  remove: async (skuId) => {
    await removeCartItem(skuId)
    set((s) => ({ items: s.items.filter((i) => i.sku_id !== skuId) }))
  },

  selectedItems: () => get().items.filter((i) => i.selected),
  totalAmount: () => get().selectedItems().reduce((sum, i) => sum + i.price * i.quantity, 0),
  totalCount: () => get().items.reduce((sum, i) => sum + i.quantity, 0),
}))
```

**Step 3: Create cart page styles**

Create `frontend/src/pages/cart/cart.module.css`:

```css
.page {
  padding: 12px 12px 80px;
}

.title {
  font-size: 20px;
  font-weight: 600;
  padding: 8px 0 16px;
}

.item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--color-card);
  border-radius: var(--radius);
  margin-bottom: 10px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.itemImage {
  width: 80px;
  height: 80px;
  border-radius: 6px;
  object-fit: cover;
  flex-shrink: 0;
}

.itemInfo {
  flex: 1;
  min-width: 0;
}

.itemName {
  font-size: 14px;
  font-weight: 500;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.itemSpec {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.itemBottom {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
}

.quantityControl {
  display: flex;
  align-items: center;
  gap: 8px;
}

.qtyBtn {
  width: 28px;
  height: 28px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-card);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  font-size: 16px;
}

.qtyValue {
  font-size: 14px;
  min-width: 24px;
  text-align: center;
}

.footer {
  position: fixed;
  bottom: 50px;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  padding: 8px 16px;
  background: var(--color-card);
  border-top: 1px solid var(--color-border);
  z-index: 99;
}

.footerTotal {
  flex: 1;
  font-size: 16px;
}

.checkoutBtn {
  --background-color: var(--color-primary);
  --border-color: var(--color-primary);
  border-radius: var(--radius);
  min-width: 120px;
}
```

**Step 4: Implement Cart page**

Replace `frontend/src/pages/cart/index.tsx`:

```tsx
import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button, Checkbox, SwipeAction, Toast } from 'antd-mobile'
import { useCartStore } from '@/stores/cart'
import Price from '@/components/Price'
import styles from './cart.module.css'

export default function CartPage() {
  const navigate = useNavigate()
  const { items, loading, fetchCart, toggleSelect, updateQuantity, remove } = useCartStore()
  const selectedItems = useCartStore((s) => s.selectedItems())
  const totalAmount = useCartStore((s) => s.totalAmount())

  useEffect(() => {
    fetchCart()
  }, [fetchCart])

  const handleCheckout = () => {
    if (selectedItems.length === 0) {
      Toast.show('请选择商品')
      return
    }
    navigate('/order/confirm')
  }

  if (loading && items.length === 0) {
    return <div style={{ textAlign: 'center', padding: '40vh 0', color: 'var(--color-text-secondary)' }}>加载中...</div>
  }

  return (
    <div className={styles.page}>
      <div className={styles.title}>购物车</div>

      {items.length === 0 ? (
        <div style={{ textAlign: 'center', padding: 60, color: 'var(--color-text-secondary)' }}>
          购物车是空的
        </div>
      ) : (
        items.map((item) => (
          <SwipeAction
            key={item.sku_id}
            rightActions={[{
              key: 'delete',
              text: '删除',
              color: 'danger',
              onClick: () => remove(item.sku_id),
            }]}
          >
            <div className={styles.item}>
              <Checkbox
                checked={item.selected}
                onChange={(v) => toggleSelect(item.sku_id, v)}
              />
              <img className={styles.itemImage} src={item.product_image || 'https://via.placeholder.com/80'} alt='' />
              <div className={styles.itemInfo}>
                <div className={styles.itemName}>{item.product_name}</div>
                {item.sku_spec && <div className={styles.itemSpec}>{item.sku_spec}</div>}
                <div className={styles.itemBottom}>
                  <Price value={item.price} size='sm' />
                  <div className={styles.quantityControl}>
                    <span
                      className={styles.qtyBtn}
                      onClick={() => item.quantity > 1 && updateQuantity(item.sku_id, item.quantity - 1)}
                    >-</span>
                    <span className={styles.qtyValue}>{item.quantity}</span>
                    <span
                      className={styles.qtyBtn}
                      onClick={() => updateQuantity(item.sku_id, item.quantity + 1)}
                    >+</span>
                  </div>
                </div>
              </div>
            </div>
          </SwipeAction>
        ))
      )}

      {items.length > 0 && (
        <div className={styles.footer}>
          <div className={styles.footerTotal}>
            合计: <Price value={totalAmount} size='md' />
          </div>
          <Button
            color='primary'
            className={styles.checkoutBtn}
            onClick={handleCheckout}
          >
            结算({selectedItems.length})
          </Button>
        </div>
      )}
    </div>
  )
}
```

**Step 5: Verify cart page renders**

```bash
cd frontend && npm run dev
```

**Step 6: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): implement cart page with Zustand store, swipe delete, checkout"
```

---

### Task 9: Order Confirm + Order API

**Files:**
- Create: `frontend/src/api/order.ts`
- Create: `frontend/src/api/user.ts`
- Create: `frontend/src/pages/order/Confirm.tsx`
- Create: `frontend/src/pages/order/confirm.module.css`
- Modify: `frontend/src/router/index.tsx` (add routes)

**Step 1: Create user API (for addresses)**

Create `frontend/src/api/user.ts`:

```ts
import { request } from './client'

export interface Address {
  id: number
  name: string
  phone: string
  province: string
  city: string
  district: string
  detail: string
  is_default: boolean
}

export interface UserProfile {
  id: number
  phone: string
  email: string
  nickname: string
  avatar: string
}

export function getProfile() {
  return request<UserProfile>({ method: 'GET', url: '/profile' })
}

export function listAddresses() {
  return request<Address[]>({ method: 'GET', url: '/addresses' })
}
```

**Step 2: Create order API**

Create `frontend/src/api/order.ts`:

```ts
import { request } from './client'

export interface CreateOrderParams {
  items: Array<{ sku_id: number; quantity: number }>
  address_id: number
  coupon_id?: number
  remark?: string
}

export interface CreateOrderResult {
  order_no: string
  pay_amount: number
}

export function createOrder(params: CreateOrderParams) {
  return request<CreateOrderResult>({ method: 'POST', url: '/orders', data: params })
}
```

**Step 3: Create confirm page styles**

Create `frontend/src/pages/order/confirm.module.css`:

```css
.page {
  padding: 12px 12px 80px;
  min-height: 100vh;
  background: var(--color-bg);
}

.navBar {
  margin: -12px -12px 12px;
}

.addressCard {
  padding: 16px;
  background: var(--color-card);
  border-radius: var(--radius);
  margin-bottom: 12px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.addressName {
  font-size: 16px;
  font-weight: 600;
}

.addressPhone {
  color: var(--color-text-secondary);
  margin-left: 12px;
}

.addressDetail {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-top: 6px;
}

.noAddress {
  color: var(--color-accent);
  cursor: pointer;
}

.itemsCard {
  padding: 12px;
  background: var(--color-card);
  border-radius: var(--radius);
  margin-bottom: 12px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.orderItem {
  display: flex;
  gap: 10px;
  padding: 8px 0;
}

.orderItem + .orderItem {
  border-top: 1px solid var(--color-border);
}

.orderItemImage {
  width: 60px;
  height: 60px;
  border-radius: 6px;
  object-fit: cover;
}

.orderItemInfo {
  flex: 1;
}

.orderItemName {
  font-size: 14px;
}

.orderItemMeta {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.remarkInput {
  padding: 12px;
  background: var(--color-card);
  border-radius: var(--radius);
  margin-bottom: 12px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: var(--color-card);
  border-top: 1px solid var(--color-border);
  z-index: 99;
}

.footerTotal {
  flex: 1;
  font-size: 16px;
}

.submitBtn {
  --background-color: var(--color-primary);
  --border-color: var(--color-primary);
  border-radius: var(--radius);
  min-width: 120px;
}
```

**Step 4: Implement Order Confirm page**

Create `frontend/src/pages/order/Confirm.tsx`:

```tsx
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Button, TextArea, Toast } from 'antd-mobile'
import { useCartStore } from '@/stores/cart'
import { listAddresses, type Address } from '@/api/user'
import { createOrder } from '@/api/order'
import Price from '@/components/Price'
import styles from './confirm.module.css'

export default function OrderConfirm() {
  const navigate = useNavigate()
  const selectedItems = useCartStore((s) => s.selectedItems())
  const totalAmount = useCartStore((s) => s.totalAmount())
  const fetchCart = useCartStore((s) => s.fetchCart)

  const [addresses, setAddresses] = useState<Address[]>([])
  const [selectedAddress, setSelectedAddress] = useState<Address | null>(null)
  const [remark, setRemark] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (selectedItems.length === 0) {
      navigate('/cart', { replace: true })
      return
    }
    listAddresses().then((list) => {
      setAddresses(list || [])
      const defaultAddr = (list || []).find((a) => a.is_default) || (list || [])[0]
      if (defaultAddr) setSelectedAddress(defaultAddr)
    }).catch(() => {})
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleSubmit = async () => {
    if (!selectedAddress) {
      Toast.show('请选择收货地址')
      return
    }
    setLoading(true)
    try {
      const result = await createOrder({
        items: selectedItems.map((i) => ({ sku_id: i.sku_id, quantity: i.quantity })),
        address_id: selectedAddress.id,
        remark,
      })
      await fetchCart()
      navigate(`/payment/${result.order_no}`, { replace: true, state: { payAmount: result.pay_amount } })
    } catch (e: unknown) {
      Toast.show((e as Error).message || '下单失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>确认订单</NavBar>
      </div>

      <div className={styles.addressCard}>
        {selectedAddress ? (
          <>
            <div>
              <span className={styles.addressName}>{selectedAddress.name}</span>
              <span className={styles.addressPhone}>{selectedAddress.phone}</span>
            </div>
            <div className={styles.addressDetail}>
              {selectedAddress.province}{selectedAddress.city}{selectedAddress.district}{selectedAddress.detail}
            </div>
          </>
        ) : (
          <div className={styles.noAddress}>
            {addresses.length === 0 ? '请先添加收货地址' : '请选择收货地址'}
          </div>
        )}
      </div>

      <div className={styles.itemsCard}>
        {selectedItems.map((item) => (
          <div key={item.sku_id} className={styles.orderItem}>
            <img className={styles.orderItemImage} src={item.product_image || 'https://via.placeholder.com/60'} alt='' />
            <div className={styles.orderItemInfo}>
              <div className={styles.orderItemName}>{item.product_name}</div>
              <div className={styles.orderItemMeta}>
                <Price value={item.price} size='sm' />
                <span>x{item.quantity}</span>
              </div>
            </div>
          </div>
        ))}
      </div>

      <div className={styles.remarkInput}>
        <TextArea placeholder='订单备注（选填）' value={remark} onChange={setRemark} maxLength={200} rows={2} />
      </div>

      <div className={styles.footer}>
        <div className={styles.footerTotal}>
          合计: <Price value={totalAmount} size='md' />
        </div>
        <Button color='primary' className={styles.submitBtn} loading={loading} onClick={handleSubmit}>
          提交订单
        </Button>
      </div>
    </div>
  )
}
```

**Step 5: Add route to router**

Add to `frontend/src/router/index.tsx` — import `OrderConfirm` and add route:

```tsx
const OrderConfirm = lazy(() => import('@/pages/order/Confirm'))

// Add inside router array (after ProductDetail):
{
  path: '/order/confirm',
  element: <AuthGuard><Lazy><OrderConfirm /></Lazy></AuthGuard>,
},
```

**Step 6: Verify**

```bash
cd frontend && npx tsc --noEmit && npm run dev
```

**Step 7: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): implement order confirm page with address selection"
```

---

### Task 10: Payment Page

**Files:**
- Create: `frontend/src/api/payment.ts`
- Create: `frontend/src/pages/payment/index.tsx`
- Create: `frontend/src/pages/payment/payment.module.css`
- Modify: `frontend/src/router/index.tsx` (add route)

**Step 1: Create payment API**

Create `frontend/src/api/payment.ts`:

```ts
import { request } from './client'

export interface CreatePaymentParams {
  order_id: number
  order_no: string
  channel: 'mock' | 'wechat' | 'alipay'
  amount: number
}

export interface CreatePaymentResult {
  payment_no: string
  pay_url: string
}

export interface Payment {
  id: number
  payment_no: string
  order_no: string
  channel: string
  amount: number
  status: number
}

export function createPayment(params: CreatePaymentParams) {
  return request<CreatePaymentResult>({ method: 'POST', url: '/payments', data: params })
}

export function getPayment(paymentNo: string) {
  return request<Payment>({ method: 'GET', url: `/payments/${paymentNo}` })
}
```

**Step 2: Create payment page styles**

Create `frontend/src/pages/payment/payment.module.css`:

```css
.page {
  min-height: 100vh;
  background: var(--color-bg);
}

.content {
  padding: 24px;
  text-align: center;
}

.amount {
  font-size: 36px;
  font-weight: 700;
  color: var(--color-primary);
  margin: 24px 0 8px;
}

.amountLabel {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: 32px;
}

.channels {
  padding: 0 24px;
}

.channelItem {
  display: flex;
  align-items: center;
  padding: 16px;
  background: var(--color-card);
  border-radius: var(--radius);
  margin-bottom: 10px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  transition: transform 0.2s;
}

.channelItem:active {
  transform: scale(0.98);
}

.channelItemSelected {
  border: 2px solid var(--color-accent);
}

.channelName {
  flex: 1;
  font-size: 16px;
  margin-left: 12px;
}

.channelIcon {
  font-size: 24px;
}

.payBtn {
  margin: 32px 24px 0;
  --background-color: var(--color-primary);
  --border-color: var(--color-primary);
  height: 48px;
  font-size: 16px;
  border-radius: var(--radius);
}

.successPage {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.successIcon {
  font-size: 64px;
  color: var(--color-success);
}

.successText {
  font-size: 20px;
  font-weight: 600;
}
```

**Step 3: Implement Payment page**

Create `frontend/src/pages/payment/index.tsx`:

```tsx
import { useState } from 'react'
import { useParams, useLocation, useNavigate } from 'react-router-dom'
import { NavBar, Button, Toast } from 'antd-mobile'
import { CheckCircleFill } from 'antd-mobile-icons'
import { createPayment } from '@/api/payment'
import styles from './payment.module.css'

const channels = [
  { key: 'mock', name: '模拟支付', icon: '💳' },
  { key: 'wechat', name: '微信支付', icon: '💚' },
  { key: 'alipay', name: '支付宝', icon: '🔵' },
] as const

export default function PaymentPage() {
  const { orderNo } = useParams<{ orderNo: string }>()
  const location = useLocation()
  const navigate = useNavigate()
  const payAmount = (location.state as { payAmount?: number })?.payAmount || 0

  const [selectedChannel, setSelectedChannel] = useState<string>('mock')
  const [loading, setLoading] = useState(false)
  const [paid, setPaid] = useState(false)

  const handlePay = async () => {
    if (!orderNo) return
    setLoading(true)
    try {
      await createPayment({
        order_id: 0,
        order_no: orderNo,
        channel: selectedChannel as 'mock' | 'wechat' | 'alipay',
        amount: payAmount,
      })
      setPaid(true)
      Toast.show('支付成功')
    } catch (e: unknown) {
      Toast.show((e as Error).message || '支付失败')
    } finally {
      setLoading(false)
    }
  }

  if (paid) {
    return (
      <div className={styles.successPage}>
        <CheckCircleFill className={styles.successIcon} />
        <div className={styles.successText}>支付成功</div>
        <Button fill='outline' onClick={() => navigate('/', { replace: true })}>
          返回首页
        </Button>
      </div>
    )
  }

  return (
    <div className={styles.page}>
      <NavBar onBack={() => navigate(-1)}>收银台</NavBar>
      <div className={styles.content}>
        <div className={styles.amountLabel}>支付金额</div>
        <div className={styles.amount}>¥{(payAmount / 100).toFixed(2)}</div>
      </div>

      <div className={styles.channels}>
        {channels.map((ch) => (
          <div
            key={ch.key}
            className={`${styles.channelItem} ${selectedChannel === ch.key ? styles.channelItemSelected : ''}`}
            onClick={() => setSelectedChannel(ch.key)}
          >
            <span className={styles.channelIcon}>{ch.icon}</span>
            <span className={styles.channelName}>{ch.name}</span>
          </div>
        ))}
      </div>

      <Button
        block
        color='primary'
        className={styles.payBtn}
        loading={loading}
        onClick={handlePay}
      >
        立即支付
      </Button>
    </div>
  )
}
```

**Step 4: Add route to router**

Add to `frontend/src/router/index.tsx`:

```tsx
const PaymentPage = lazy(() => import('@/pages/payment'))

// Add inside router array:
{
  path: '/payment/:orderNo',
  element: <AuthGuard><Lazy><PaymentPage /></Lazy></AuthGuard>,
},
```

**Step 5: Verify full flow compiles**

```bash
cd frontend && npx tsc --noEmit && npm run dev
```

**Step 6: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): implement payment page with channel selection and success state"
```

---

### Task 11: Product Detail Page

**Files:**
- Create: `frontend/src/api/inventory.ts`
- Create: `frontend/src/pages/product/detail.module.css`
- Modify: `frontend/src/pages/product/Detail.tsx`

**Step 1: Create inventory API**

Create `frontend/src/api/inventory.ts`:

```ts
import { request } from './client'

export interface StockInfo {
  sku_id: number
  available: number
}

export function getStock(skuId: number) {
  return request<StockInfo>({ method: 'GET', url: `/inventory/stock/${skuId}` })
}

export function batchGetStock(skuIds: number[]) {
  return request<StockInfo[]>({ method: 'POST', url: '/inventory/stock/batch', data: { sku_ids: skuIds } })
}
```

**Step 2: Create product detail styles**

Create `frontend/src/pages/product/detail.module.css`:

```css
.page {
  min-height: 100vh;
  background: var(--color-bg);
  padding-bottom: 70px;
}

.image {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  display: block;
  background: #f5f5f5;
}

.info {
  padding: 16px;
  background: var(--color-card);
  margin-bottom: 12px;
}

.name {
  font-size: 18px;
  font-weight: 600;
  line-height: 1.4;
  margin-bottom: 8px;
}

.priceRow {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.description {
  padding: 16px;
  background: var(--color-card);
  margin-bottom: 12px;
}

.descTitle {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
}

.descText {
  font-size: 14px;
  color: var(--color-text-secondary);
  line-height: 1.8;
}

.stock {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-top: 8px;
}

.stockLow {
  color: var(--color-danger);
}

.footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  gap: 10px;
  padding: 10px 16px;
  background: var(--color-card);
  border-top: 1px solid var(--color-border);
  z-index: 99;
}

.cartBtn {
  flex: 1;
  --border-color: var(--color-primary);
  border-radius: var(--radius);
  height: 44px;
}

.buyBtn {
  flex: 1;
  --background-color: var(--color-primary);
  --border-color: var(--color-primary);
  border-radius: var(--radius);
  height: 44px;
}
```

**Step 3: Implement Product Detail page**

Replace `frontend/src/pages/product/Detail.tsx`:

```tsx
import { useState, useEffect } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { NavBar, Button, Toast } from 'antd-mobile'
import { addCartItem } from '@/api/cart'
import { getStock } from '@/api/inventory'
import { useAuthStore } from '@/stores/auth'
import Price from '@/components/Price'
import type { Product } from '@/types/product'
import styles from './detail.module.css'

export default function ProductDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)

  // Product data passed from search/list via route state
  const product = (location.state as { product?: Product })?.product
  const [stock, setStock] = useState<number | null>(null)
  const [adding, setAdding] = useState(false)

  useEffect(() => {
    if (id) {
      getStock(Number(id)).then((s) => setStock(s.available)).catch(() => {})
    }
  }, [id])

  if (!product) {
    return (
      <div className={styles.page}>
        <NavBar onBack={() => navigate(-1)}>商品详情</NavBar>
        <div style={{ textAlign: 'center', padding: 60, color: 'var(--color-text-secondary)' }}>
          商品信息加载失败
        </div>
      </div>
    )
  }

  const handleAddCart = async () => {
    if (!isLoggedIn) {
      navigate(`/login?redirect=${encodeURIComponent(location.pathname)}`, { state: location.state })
      return
    }
    setAdding(true)
    try {
      await addCartItem({ sku_id: Number(id), product_id: product.id, quantity: 1 })
      Toast.show('已加入购物车')
    } catch (e: unknown) {
      Toast.show((e as Error).message || '添加失败')
    } finally {
      setAdding(false)
    }
  }

  return (
    <div className={styles.page}>
      <NavBar onBack={() => navigate(-1)} style={{ background: 'transparent', position: 'absolute', zIndex: 10, width: '100%' }} />
      <img
        className={styles.image}
        src={product.main_image || 'https://via.placeholder.com/400'}
        alt={product.name}
      />
      <div className={styles.info}>
        <div className={styles.name}>{product.name}</div>
        <div className={styles.priceRow}>
          <Price value={product.price} original={product.original_price} size='lg' />
        </div>
        {stock !== null && (
          <div className={`${styles.stock} ${stock < 10 ? styles.stockLow : ''}`}>
            {stock > 0 ? `库存 ${stock} 件` : '已售罄'}
          </div>
        )}
      </div>

      {product.description && (
        <div className={styles.description}>
          <div className={styles.descTitle}>商品详情</div>
          <div className={styles.descText}>{product.description}</div>
        </div>
      )}

      <div className={styles.footer}>
        <Button fill='outline' className={styles.cartBtn} loading={adding} onClick={handleAddCart}>
          加入购物车
        </Button>
        <Button color='primary' className={styles.buyBtn} onClick={handleAddCart}>
          立即购买
        </Button>
      </div>
    </div>
  )
}
```

**Step 4: Update ProductCard to pass product data via route state**

In `frontend/src/components/ProductCard/index.tsx`, change the navigate call:

```tsx
onClick={() => navigate(`/product/${product.id}`, { state: { product } })}
```

**Step 5: Verify**

```bash
cd frontend && npx tsc --noEmit && npm run dev
```

**Step 6: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): implement product detail page with stock check and add-to-cart"
```

---

### Task 12: Final Verification

**Step 1: Full TypeScript check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: 0 errors

**Step 2: Build for production**

```bash
cd frontend && npm run build
```

Expected: successful build, output in `frontend/dist/`

**Step 3: Verify all routes work in dev**

```bash
cd frontend && npm run dev
```

Manually verify:
- `/` — Home page renders (shop header, sections)
- `/search` — SearchBar + hot words
- `/login` — Login form renders
- `/signup` — Signup form renders
- `/cart` — Redirects to login (not authenticated)
- Tab navigation works

**Step 4: Final commit**

```bash
git add frontend/
git commit -m "feat(frontend): Phase 1 MVP complete — core shopping flow"
```
