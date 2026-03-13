# Consumer Frontend Phase 3 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Consume all remaining 14 consumer-bff endpoints by adding search enhancements, notification center, coupon center, seckill page, and coupon selection at checkout.

**Architecture:** Follows established pattern — API layer in `src/api/`, pages in `src/pages/`, CSS Modules, lazy-loaded routes with AuthGuard. No new state stores; data fetched via useEffect. Debounced search suggestions, setInterval-based countdown for seckill.

**Tech Stack:** React 19, TypeScript, antd-mobile 5, axios, react-router-dom 7, zustand 5, CSS Modules, Vite 7

---

## Task 1: Extend API Modules

**Files:**
- Modify: `frontend/src/api/search.ts`
- Modify: `frontend/src/api/marketing.ts`
- Create: `frontend/src/api/notification.ts`

**Step 1: Add search history functions to `src/api/search.ts`**

Append after the existing `getHotWords` function at line 41:

```ts
export function getSearchHistory(limit: number = 20) {
  return request<string[]>({
    method: 'GET',
    url: '/search/history',
    params: { limit },
  })
}

export function clearSearchHistory() {
  return request<void>({ method: 'DELETE', url: '/search/history' })
}
```

**Step 2: Add marketing functions to `src/api/marketing.ts`**

Append after `listAvailableCoupons` at line 46. Also add `UserCoupon` and `SeckillResult` types:

```ts
export interface UserCoupon {
  id: number
  coupon_id: number
  name: string
  type: number        // 1=满减, 2=折扣
  value: number
  min_spend: number
  start_time: string
  end_time: string
  status: number      // 1=可用, 2=已用, 3=已过期
}

export interface SeckillResult {
  success: boolean
  message: string
  order_no: string
}

export function receiveCoupon(id: number) {
  return request<void>({ method: 'POST', url: `/coupons/${id}/receive` })
}

export function listMyCoupons(status?: number) {
  return request<UserCoupon[]>({
    method: 'GET',
    url: '/coupons/mine',
    params: status ? { status } : undefined,
  })
}

export function seckill(itemId: number) {
  return request<SeckillResult>({ method: 'POST', url: `/seckill/${itemId}` })
}
```

**Step 3: Create `src/api/notification.ts`**

```ts
import { request } from './client'

export interface Notification {
  id: number
  channel: number    // 1=SMS, 2=Email, 3=In-app
  title: string
  content: string
  is_read: boolean
  status: number     // 1=pending, 2=sent, 3=failed
  ctime: string
}

export interface NotificationPageResult {
  notifications: Notification[]
  total: number
}

export function listNotifications(params: {
  channel?: number
  unreadOnly?: boolean
  page: number
  pageSize: number
}) {
  return request<NotificationPageResult>({
    method: 'GET',
    url: '/notifications',
    params,
  })
}

export function getUnreadCount() {
  return request<number>({ method: 'GET', url: '/notifications/unread-count' })
}

export function markRead(id: number) {
  return request<void>({ method: 'PUT', url: `/notifications/${id}/read` })
}

export function markAllRead() {
  return request<void>({ method: 'PUT', url: '/notifications/read-all' })
}
```

**Step 4: Verify types compile**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

---

## Task 2: Search Page Enhancement (History + Autocomplete)

**Files:**
- Modify: `frontend/src/pages/search/index.tsx` (full rewrite)
- Modify: `frontend/src/pages/search/search.module.css` (add new classes)

**Step 1: Add CSS classes for history section and suggestion dropdown to `search.module.css`**

Append after the existing `.loadMore` rule (line 52):

```css
.historySection {
  padding: 12px 0;
}

.historyHeader {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.historyTitle {
  font-size: 14px;
  font-weight: 600;
}

.historyClear {
  font-size: 13px;
  color: var(--color-text-secondary);
  cursor: pointer;
}

.historyTags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.historyTag {
  padding: 6px 14px;
  background: var(--color-card);
  border-radius: 16px;
  font-size: 13px;
  color: var(--color-primary);
  border: 1px solid var(--color-border);
  cursor: pointer;
}

.suggestionsWrap {
  position: relative;
}

.suggestions {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  background: var(--color-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  z-index: 100;
  max-height: 240px;
  overflow-y: auto;
}

.suggestionItem {
  padding: 10px 14px;
  font-size: 14px;
  cursor: pointer;
}

.suggestionItem + .suggestionItem {
  border-top: 1px solid var(--color-border);
}
```

**Step 2: Rewrite `src/pages/search/index.tsx` with history + autocomplete**

Replace the entire file with:

```tsx
import { useState, useEffect, useCallback, useRef } from 'react'
import { SearchBar, InfiniteScroll } from 'antd-mobile'
import { searchProducts, getHotWords, getSuggestions, getSearchHistory, clearSearchHistory } from '@/api/search'
import { useAuthStore } from '@/stores/auth'
import ProductCard from '@/components/ProductCard'
import type { Product } from '@/types/product'
import styles from './search.module.css'

export default function SearchPage() {
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const [keyword, setKeyword] = useState('')
  const [products, setProducts] = useState<Product[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [hotWords, setHotWords] = useState<string[]>([])
  const [history, setHistory] = useState<string[]>([])
  const [suggestions, setSuggestions] = useState<string[]>([])
  const [showSuggestions, setShowSuggestions] = useState(false)
  const [searched, setSearched] = useState(false)
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()

  useEffect(() => {
    getHotWords().then(setHotWords).catch(() => {})
    if (isLoggedIn) {
      getSearchHistory(20).then(setHistory).catch(() => {})
    }
  }, [isLoggedIn])

  const doSearch = useCallback(async (kw: string, pageNum: number = 1) => {
    setShowSuggestions(false)
    setSuggestions([])
    const res = await searchProducts({ keyword: kw, page: pageNum, page_size: 20 })
    if (pageNum === 1) {
      setProducts(res.products || [])
    } else {
      setProducts((prev) => [...prev, ...(res.products || [])])
    }
    setTotal(res.total || 0)
    setPage(pageNum)
    setSearched(true)
    // refresh history after search
    if (isLoggedIn) {
      getSearchHistory(20).then(setHistory).catch(() => {})
    }
  }, [isLoggedIn])

  const handleSearch = (val: string) => {
    setKeyword(val)
    if (val.trim()) {
      doSearch(val.trim())
    }
  }

  const handleInputChange = (val: string) => {
    setKeyword(val)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    if (!val.trim()) {
      setSuggestions([])
      setShowSuggestions(false)
      return
    }
    debounceRef.current = setTimeout(() => {
      getSuggestions(val.trim()).then((list) => {
        setSuggestions(list || [])
        setShowSuggestions((list || []).length > 0)
      }).catch(() => {})
    }, 300)
  }

  const handleClearHistory = async () => {
    try {
      await clearSearchHistory()
      setHistory([])
    } catch {
      // ignore
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
          onChange={handleInputChange}
          onSearch={handleSearch}
          onFocus={() => {
            if (suggestions.length > 0) setShowSuggestions(true)
          }}
          onClear={() => { setSearched(false); setProducts([]); setSuggestions([]); setShowSuggestions(false) }}
        />
      </div>

      <div className={styles.suggestionsWrap}>
        {showSuggestions && suggestions.length > 0 && (
          <div className={styles.suggestions}>
            {suggestions.map((s) => (
              <div key={s} className={styles.suggestionItem} onClick={() => handleSearch(s)}>
                {s}
              </div>
            ))}
          </div>
        )}
      </div>

      {!searched && (
        <>
          {hotWords.length > 0 && (
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

          {history.length > 0 && (
            <div className={styles.historySection}>
              <div className={styles.historyHeader}>
                <span className={styles.historyTitle}>搜索历史</span>
                <span className={styles.historyClear} onClick={handleClearHistory}>清空</span>
              </div>
              <div className={styles.historyTags}>
                {history.map((w) => (
                  <span key={w} className={styles.historyTag} onClick={() => handleSearch(w)}>
                    {w}
                  </span>
                ))}
              </div>
            </div>
          )}
        </>
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

**Step 3: Verify compile**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

---

## Task 3: Notification List Page

**Files:**
- Create: `frontend/src/pages/notification/index.tsx`
- Create: `frontend/src/pages/notification/notification.module.css`

**Step 1: Create `src/pages/notification/notification.module.css`**

```css
.page {
  padding: 12px;
  min-height: 100vh;
  background: var(--color-bg);
}

.navBar {
  margin: -12px -12px 12px;
}

.actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}

.markAllBtn {
  font-size: 13px;
  color: var(--color-accent);
  cursor: pointer;
}

.card {
  padding: 14px 16px;
  background: var(--color-card);
  border-radius: var(--radius);
  margin-bottom: 10px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  position: relative;
}

.unread {
  border-left: 3px solid var(--color-accent);
}

.cardTitle {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}

.cardContent {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.4;
  margin-bottom: 6px;
}

.cardTime {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.empty {
  text-align: center;
  padding: 60px 0;
  color: var(--color-text-secondary);
}

.loadMore {
  padding: 12px 0;
}
```

**Step 2: Create `src/pages/notification/index.tsx`**

```tsx
import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, InfiniteScroll, Toast } from 'antd-mobile'
import { listNotifications, markRead, markAllRead, type Notification } from '@/api/notification'
import styles from './notification.module.css'

export default function NotificationPage() {
  const navigate = useNavigate()
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [hasMore, setHasMore] = useState(true)
  const [page, setPage] = useState(1)

  const loadMore = useCallback(async () => {
    const res = await listNotifications({ page, pageSize: 20 })
    const list = res.notifications || []
    setNotifications((prev) => (page === 1 ? list : [...prev, ...list]))
    setHasMore(list.length >= 20)
    setPage((p) => p + 1)
  }, [page])

  const handleMarkRead = async (n: Notification) => {
    if (n.is_read) return
    try {
      await markRead(n.id)
      setNotifications((prev) =>
        prev.map((item) => (item.id === n.id ? { ...item, is_read: true } : item))
      )
    } catch {
      // ignore
    }
  }

  const handleMarkAllRead = async () => {
    try {
      await markAllRead()
      setNotifications((prev) => prev.map((item) => ({ ...item, is_read: true })))
      Toast.show('已全部标记为已读')
    } catch (e: unknown) {
      Toast.show((e as Error).message || '操作失败')
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>消息通知</NavBar>
      </div>

      {notifications.length > 0 && (
        <div className={styles.actions}>
          <span className={styles.markAllBtn} onClick={handleMarkAllRead}>全部已读</span>
        </div>
      )}

      {notifications.map((n) => (
        <div
          key={n.id}
          className={`${styles.card} ${!n.is_read ? styles.unread : ''}`}
          onClick={() => handleMarkRead(n)}
        >
          <div className={styles.cardTitle}>{n.title}</div>
          <div className={styles.cardContent}>{n.content}</div>
          <div className={styles.cardTime}>{new Date(n.ctime).toLocaleString('zh-CN')}</div>
        </div>
      ))}

      {notifications.length === 0 && !hasMore && (
        <div className={styles.empty}>暂无消息</div>
      )}

      <div className={styles.loadMore}>
        <InfiniteScroll loadMore={loadMore} hasMore={hasMore} />
      </div>
    </div>
  )
}
```

**Step 3: Verify compile**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

---

## Task 4: Coupon Center Page

**Files:**
- Create: `frontend/src/pages/marketing/Coupons.tsx`
- Create: `frontend/src/pages/marketing/coupons.module.css`

**Step 1: Create `src/pages/marketing/coupons.module.css`**

```css
.page {
  min-height: 100vh;
  background: var(--color-bg);
}

.navBar {
  background: var(--color-card);
}

.tabs {
  --active-line-color: var(--color-accent);
  --active-title-color: var(--color-primary);
  background: var(--color-card);
  position: sticky;
  top: 0;
  z-index: 10;
}

.content {
  padding: 12px;
}

.couponCard {
  display: flex;
  align-items: center;
  background: var(--color-card);
  border-radius: var(--radius);
  margin-bottom: 10px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.couponLeft {
  width: 100px;
  padding: 16px 12px;
  text-align: center;
  background: linear-gradient(135deg, #FFF8F0, #FFFFFF);
  border-right: 1px dashed var(--color-border);
}

.couponValue {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-accent);
}

.couponCondition {
  font-size: 11px;
  color: var(--color-text-secondary);
  margin-top: 2px;
}

.couponRight {
  flex: 1;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.couponInfo {
  flex: 1;
}

.couponName {
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 4px;
}

.couponTime {
  font-size: 11px;
  color: var(--color-text-secondary);
}

.couponRemaining {
  font-size: 11px;
  color: var(--color-text-secondary);
  margin-top: 2px;
}

.receiveBtn {
  border-radius: var(--radius);
  --background-color: var(--color-accent);
  --border-color: var(--color-accent);
  font-size: 13px;
  min-width: 60px;
}

.empty {
  text-align: center;
  padding: 60px 0;
  color: var(--color-text-secondary);
}
```

**Step 2: Create `src/pages/marketing/Coupons.tsx`**

```tsx
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Tabs, Button, Toast } from 'antd-mobile'
import { listAvailableCoupons, receiveCoupon, listMyCoupons, type Coupon, type UserCoupon } from '@/api/marketing'
import { useAuthStore } from '@/stores/auth'
import styles from './coupons.module.css'

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('zh-CN')
}

export default function CouponsPage() {
  const navigate = useNavigate()
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const [activeTab, setActiveTab] = useState('available')
  const [available, setAvailable] = useState<Coupon[]>([])
  const [mine, setMine] = useState<UserCoupon[]>([])

  useEffect(() => {
    listAvailableCoupons().then(setAvailable).catch(() => {})
    if (isLoggedIn) {
      listMyCoupons().then(setMine).catch(() => {})
    }
  }, [isLoggedIn])

  const handleReceive = async (id: number) => {
    if (!isLoggedIn) {
      navigate('/login')
      return
    }
    try {
      await receiveCoupon(id)
      Toast.show('领取成功')
      setAvailable((prev) =>
        prev.map((c) => (c.id === id ? { ...c, remaining: c.remaining - 1 } : c))
      )
      listMyCoupons().then(setMine).catch(() => {})
    } catch (e: unknown) {
      Toast.show((e as Error).message || '领取失败')
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>优惠券</NavBar>
      </div>

      <Tabs className={styles.tabs} activeKey={activeTab} onChange={setActiveTab}>
        <Tabs.Tab key="available" title="领券中心" />
        <Tabs.Tab key="mine" title="我的优惠券" />
      </Tabs>

      <div className={styles.content}>
        {activeTab === 'available' && (
          <>
            {available.map((c) => (
              <div key={c.id} className={styles.couponCard}>
                <div className={styles.couponLeft}>
                  <div className={styles.couponValue}>
                    {c.type === 1 ? `¥${(c.value / 100).toFixed(0)}` : `${c.value / 10}折`}
                  </div>
                  <div className={styles.couponCondition}>满{(c.min_spend / 100).toFixed(0)}可用</div>
                </div>
                <div className={styles.couponRight}>
                  <div className={styles.couponInfo}>
                    <div className={styles.couponName}>{c.name}</div>
                    <div className={styles.couponTime}>{formatDate(c.start_time)} - {formatDate(c.end_time)}</div>
                    <div className={styles.couponRemaining}>剩余 {c.remaining} 张</div>
                  </div>
                  <Button
                    size="mini"
                    color="primary"
                    className={styles.receiveBtn}
                    disabled={c.remaining <= 0}
                    onClick={() => handleReceive(c.id)}
                  >
                    {c.remaining > 0 ? '领取' : '已抢光'}
                  </Button>
                </div>
              </div>
            ))}
            {available.length === 0 && <div className={styles.empty}>暂无可领优惠券</div>}
          </>
        )}

        {activeTab === 'mine' && (
          <>
            {mine.map((c) => (
              <div key={c.id} className={styles.couponCard}>
                <div className={styles.couponLeft}>
                  <div className={styles.couponValue}>
                    {c.type === 1 ? `¥${(c.value / 100).toFixed(0)}` : `${c.value / 10}折`}
                  </div>
                  <div className={styles.couponCondition}>满{(c.min_spend / 100).toFixed(0)}可用</div>
                </div>
                <div className={styles.couponRight}>
                  <div className={styles.couponInfo}>
                    <div className={styles.couponName}>{c.name}</div>
                    <div className={styles.couponTime}>{formatDate(c.start_time)} - {formatDate(c.end_time)}</div>
                  </div>
                </div>
              </div>
            ))}
            {mine.length === 0 && <div className={styles.empty}>暂无优惠券</div>}
          </>
        )}
      </div>
    </div>
  )
}
```

**Step 3: Verify compile**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

---

## Task 5: My Coupons Page

**Files:**
- Create: `frontend/src/pages/marketing/MyCoupons.tsx`
- Create: `frontend/src/pages/marketing/myCoupons.module.css`

**Step 1: Create `src/pages/marketing/myCoupons.module.css`**

```css
.page {
  min-height: 100vh;
  background: var(--color-bg);
}

.navBar {
  background: var(--color-card);
}

.tabs {
  --active-line-color: var(--color-accent);
  --active-title-color: var(--color-primary);
  background: var(--color-card);
  position: sticky;
  top: 0;
  z-index: 10;
}

.content {
  padding: 12px;
}

.couponCard {
  display: flex;
  align-items: center;
  background: var(--color-card);
  border-radius: var(--radius);
  margin-bottom: 10px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.couponDisabled {
  opacity: 0.5;
}

.couponLeft {
  width: 100px;
  padding: 16px 12px;
  text-align: center;
  background: linear-gradient(135deg, #FFF8F0, #FFFFFF);
  border-right: 1px dashed var(--color-border);
}

.couponValue {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-accent);
}

.couponCondition {
  font-size: 11px;
  color: var(--color-text-secondary);
  margin-top: 2px;
}

.couponRight {
  flex: 1;
  padding: 12px 16px;
}

.couponName {
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 4px;
}

.couponTime {
  font-size: 11px;
  color: var(--color-text-secondary);
}

.statusTag {
  display: inline-block;
  font-size: 11px;
  margin-top: 4px;
  padding: 1px 6px;
  border-radius: 3px;
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
}

.empty {
  text-align: center;
  padding: 60px 0;
  color: var(--color-text-secondary);
}
```

**Step 2: Create `src/pages/marketing/MyCoupons.tsx`**

```tsx
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Tabs } from 'antd-mobile'
import { listMyCoupons, type UserCoupon } from '@/api/marketing'
import styles from './myCoupons.module.css'

const STATUS_TABS = [
  { key: '0', title: '全部' },
  { key: '1', title: '可用' },
  { key: '2', title: '已使用' },
  { key: '3', title: '已过期' },
]

const STATUS_LABEL: Record<number, string> = { 1: '可用', 2: '已使用', 3: '已过期' }

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('zh-CN')
}

export default function MyCouponsPage() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('0')
  const [coupons, setCoupons] = useState<UserCoupon[]>([])

  useEffect(() => {
    const status = activeTab === '0' ? undefined : Number(activeTab)
    listMyCoupons(status).then(setCoupons).catch(() => setCoupons([]))
  }, [activeTab])

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>我的优惠券</NavBar>
      </div>

      <Tabs className={styles.tabs} activeKey={activeTab} onChange={setActiveTab}>
        {STATUS_TABS.map((t) => (
          <Tabs.Tab key={t.key} title={t.title} />
        ))}
      </Tabs>

      <div className={styles.content}>
        {coupons.map((c) => (
          <div
            key={c.id}
            className={`${styles.couponCard} ${c.status !== 1 ? styles.couponDisabled : ''}`}
          >
            <div className={styles.couponLeft}>
              <div className={styles.couponValue}>
                {c.type === 1 ? `¥${(c.value / 100).toFixed(0)}` : `${c.value / 10}折`}
              </div>
              <div className={styles.couponCondition}>满{(c.min_spend / 100).toFixed(0)}可用</div>
            </div>
            <div className={styles.couponRight}>
              <div className={styles.couponName}>{c.name}</div>
              <div className={styles.couponTime}>{formatDate(c.start_time)} - {formatDate(c.end_time)}</div>
              <span className={styles.statusTag}>{STATUS_LABEL[c.status] || '未知'}</span>
            </div>
          </div>
        ))}
        {coupons.length === 0 && <div className={styles.empty}>暂无优惠券</div>}
      </div>
    </div>
  )
}
```

**Step 3: Verify compile**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

---

## Task 6: Seckill Activities Page

**Files:**
- Create: `frontend/src/pages/marketing/Seckill.tsx`
- Create: `frontend/src/pages/marketing/seckill.module.css`

**Step 1: Create `src/pages/marketing/seckill.module.css`**

```css
.page {
  padding: 12px;
  min-height: 100vh;
  background: var(--color-bg);
}

.navBar {
  margin: -12px -12px 12px;
}

.activityCard {
  background: var(--color-card);
  border-radius: var(--radius);
  margin-bottom: 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.activityHeader {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--color-primary);
  color: #fff;
}

.activityName {
  font-size: 16px;
  font-weight: 600;
}

.countdown {
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.countdownLabel {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.7);
  margin-right: 4px;
}

.countdownTime {
  background: rgba(255, 255, 255, 0.2);
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 600;
}

.itemsList {
  padding: 12px 16px;
}

.seckillItem {
  display: flex;
  gap: 12px;
  padding: 10px 0;
  align-items: center;
}

.seckillItem + .seckillItem {
  border-top: 1px solid var(--color-border);
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
}

.itemName {
  font-size: 14px;
  margin-bottom: 4px;
}

.itemPrices {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.seckillPrice {
  font-size: 18px;
  font-weight: 700;
  color: var(--color-danger);
}

.originalPrice {
  font-size: 12px;
  color: var(--color-text-secondary);
  text-decoration: line-through;
}

.itemStock {
  font-size: 11px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.buyBtn {
  border-radius: var(--radius);
  --background-color: var(--color-danger);
  --border-color: var(--color-danger);
  min-width: 70px;
  flex-shrink: 0;
}

.empty {
  text-align: center;
  padding: 60px 0;
  color: var(--color-text-secondary);
}
```

**Step 2: Create `src/pages/marketing/Seckill.tsx`**

```tsx
import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Button, Toast } from 'antd-mobile'
import { listSeckillActivities, seckill, type SeckillActivity } from '@/api/marketing'
import { useAuthStore } from '@/stores/auth'
import styles from './seckill.module.css'

function useCountdown(endTime: string): string {
  const [text, setText] = useState('')
  const intervalRef = useRef<ReturnType<typeof setInterval>>()

  const compute = useCallback(() => {
    const diff = new Date(endTime).getTime() - Date.now()
    if (diff <= 0) {
      setText('已结束')
      if (intervalRef.current) clearInterval(intervalRef.current)
      return
    }
    const h = Math.floor(diff / 3600000)
    const m = Math.floor((diff % 3600000) / 60000)
    const s = Math.floor((diff % 60000) / 1000)
    setText(`${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`)
  }, [endTime])

  useEffect(() => {
    compute()
    intervalRef.current = setInterval(compute, 1000)
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [compute])

  return text
}

function ActivityCard({ activity }: { activity: SeckillActivity }) {
  const navigate = useNavigate()
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const countdown = useCountdown(activity.end_time)
  const isStarted = new Date(activity.start_time).getTime() <= Date.now()
  const isEnded = countdown === '已结束'

  const handleBuy = async (itemId: number) => {
    if (!isLoggedIn) {
      navigate('/login')
      return
    }
    try {
      const res = await seckill(itemId)
      if (res.order_no) {
        Toast.show('抢购成功！')
        navigate(`/payment/${res.order_no}`)
      } else {
        Toast.show(res.message || '抢购成功')
      }
    } catch (e: unknown) {
      Toast.show((e as Error).message || '抢购失败')
    }
  }

  return (
    <div className={styles.activityCard}>
      <div className={styles.activityHeader}>
        <span className={styles.activityName}>{activity.name}</span>
        <span className={styles.countdown}>
          {isEnded ? (
            '已结束'
          ) : !isStarted ? (
            '即将开始'
          ) : (
            <>
              <span className={styles.countdownLabel}>距结束</span>
              <span className={styles.countdownTime}>{countdown}</span>
            </>
          )}
        </span>
      </div>
      <div className={styles.itemsList}>
        {(activity.items || []).map((item) => (
          <div key={item.id} className={styles.seckillItem}>
            <img
              className={styles.itemImage}
              src={item.product_image || 'https://via.placeholder.com/80'}
              alt={item.product_name}
            />
            <div className={styles.itemInfo}>
              <div className={styles.itemName}>{item.product_name}</div>
              <div className={styles.itemPrices}>
                <span className={styles.seckillPrice}>¥{(item.seckill_price / 100).toFixed(2)}</span>
                <span className={styles.originalPrice}>¥{(item.original_price / 100).toFixed(2)}</span>
              </div>
              <div className={styles.itemStock}>剩余 {item.available_stock} 件</div>
            </div>
            <Button
              size="mini"
              color="danger"
              className={styles.buyBtn}
              disabled={isEnded || !isStarted || item.available_stock <= 0}
              onClick={() => handleBuy(item.id)}
            >
              {item.available_stock <= 0 ? '已抢光' : !isStarted ? '未开始' : '抢购'}
            </Button>
          </div>
        ))}
      </div>
    </div>
  )
}

export default function SeckillPage() {
  const navigate = useNavigate()
  const [activities, setActivities] = useState<SeckillActivity[]>([])

  useEffect(() => {
    listSeckillActivities().then(setActivities).catch(() => {})
  }, [])

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>限时秒杀</NavBar>
      </div>

      {activities.map((a) => (
        <ActivityCard key={a.id} activity={a} />
      ))}

      {activities.length === 0 && <div className={styles.empty}>暂无秒杀活动</div>}
    </div>
  )
}
```

**Step 3: Verify compile**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

---

## Task 7: Order Confirm — Coupon Selector

**Files:**
- Modify: `frontend/src/pages/order/Confirm.tsx`
- Modify: `frontend/src/pages/order/confirm.module.css`

**Step 1: Add coupon selector CSS to `confirm.module.css`**

Append after `.submitBtn` rule (line 112):

```css
.couponCard {
  padding: 14px 16px;
  background: var(--color-card);
  border-radius: var(--radius);
  margin-bottom: 12px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
}

.couponLabel {
  font-size: 14px;
}

.couponSelected {
  color: var(--color-accent);
  font-weight: 500;
}

.couponArrow {
  color: var(--color-text-secondary);
  font-size: 14px;
}

.popupContent {
  padding: 16px;
  max-height: 60vh;
  overflow-y: auto;
}

.popupTitle {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 16px;
  text-align: center;
}

.popupCouponItem {
  display: flex;
  align-items: center;
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  margin-bottom: 10px;
  cursor: pointer;
}

.popupCouponItemActive {
  border-color: var(--color-accent);
  background: #FFFAF0;
}

.popupCouponValue {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-accent);
  min-width: 60px;
  text-align: center;
}

.popupCouponInfo {
  flex: 1;
  margin-left: 12px;
}

.popupCouponName {
  font-size: 14px;
}

.popupCouponCondition {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.popupNoCoupon {
  padding: 12px;
  text-align: center;
  color: var(--color-text-secondary);
  cursor: pointer;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.popupNoCouponActive {
  border-color: var(--color-accent);
  background: #FFFAF0;
}
```

**Step 2: Rewrite `src/pages/order/Confirm.tsx` with coupon selector**

Replace the entire file:

```tsx
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Button, TextArea, Toast, Popup } from 'antd-mobile'
import { useCartStore } from '@/stores/cart'
import { listAddresses, type Address } from '@/api/user'
import { createOrder } from '@/api/order'
import { listMyCoupons, type UserCoupon } from '@/api/marketing'
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
  const [coupons, setCoupons] = useState<UserCoupon[]>([])
  const [selectedCoupon, setSelectedCoupon] = useState<UserCoupon | null>(null)
  const [showCouponPopup, setShowCouponPopup] = useState(false)

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
    listMyCoupons(1).then((list) => setCoupons(list || [])).catch(() => {})
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const usableCoupons = coupons.filter((c) => c.min_spend <= totalAmount)

  const discountAmount = selectedCoupon
    ? (selectedCoupon.type === 1 ? selectedCoupon.value : Math.round(totalAmount * (1 - selectedCoupon.value / 100)))
    : 0
  const payAmount = Math.max(totalAmount - discountAmount, 0)

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
        coupon_id: selectedCoupon?.coupon_id,
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

      <div className={styles.couponCard} onClick={() => setShowCouponPopup(true)}>
        <span className={styles.couponLabel}>优惠券</span>
        <span>
          {selectedCoupon ? (
            <span className={styles.couponSelected}>
              -{selectedCoupon.type === 1 ? `¥${(selectedCoupon.value / 100).toFixed(0)}` : `${selectedCoupon.value / 10}折`}
            </span>
          ) : (
            <span style={{ color: usableCoupons.length > 0 ? 'var(--color-accent)' : 'var(--color-text-secondary)' }}>
              {usableCoupons.length > 0 ? `${usableCoupons.length}张可用` : '无可用券'}
            </span>
          )}
          <span className={styles.couponArrow}> ›</span>
        </span>
      </div>

      <div className={styles.remarkInput}>
        <TextArea placeholder='订单备注（选填）' value={remark} onChange={setRemark} maxLength={200} rows={2} />
      </div>

      <div className={styles.footer}>
        <div className={styles.footerTotal}>
          合计: <Price value={payAmount} size='md' />
        </div>
        <Button color='primary' className={styles.submitBtn} loading={loading} onClick={handleSubmit}>
          提交订单
        </Button>
      </div>

      <Popup
        visible={showCouponPopup}
        onMaskClick={() => setShowCouponPopup(false)}
        bodyStyle={{ borderTopLeftRadius: 12, borderTopRightRadius: 12 }}
      >
        <div className={styles.popupContent}>
          <div className={styles.popupTitle}>选择优惠券</div>
          <div
            className={`${styles.popupNoCoupon} ${!selectedCoupon ? styles.popupNoCouponActive : ''}`}
            onClick={() => { setSelectedCoupon(null); setShowCouponPopup(false) }}
          >
            不使用优惠券
          </div>
          {usableCoupons.map((c) => (
            <div
              key={c.id}
              className={`${styles.popupCouponItem} ${selectedCoupon?.id === c.id ? styles.popupCouponItemActive : ''}`}
              onClick={() => { setSelectedCoupon(c); setShowCouponPopup(false) }}
            >
              <span className={styles.popupCouponValue}>
                {c.type === 1 ? `¥${(c.value / 100).toFixed(0)}` : `${c.value / 10}折`}
              </span>
              <div className={styles.popupCouponInfo}>
                <div className={styles.popupCouponName}>{c.name}</div>
                <div className={styles.popupCouponCondition}>满{(c.min_spend / 100).toFixed(0)}可用</div>
              </div>
            </div>
          ))}
        </div>
      </Popup>
    </div>
  )
}
```

**Step 3: Verify compile**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

---

## Task 8: User Center + TabBarLayout — Add Entries & Badge

**Files:**
- Modify: `frontend/src/pages/user/index.tsx`
- Modify: `frontend/src/pages/user/user.module.css`
- Modify: `frontend/src/components/Layout/TabBarLayout.tsx`

**Step 1: Add badge CSS to `user.module.css`**

Append after `.logoutBtn` rule (line 95):

```css
.menuBadge {
  display: inline-block;
  min-width: 16px;
  height: 16px;
  line-height: 16px;
  padding: 0 4px;
  background: var(--color-danger);
  color: #fff;
  font-size: 10px;
  border-radius: 8px;
  text-align: center;
  margin-right: 8px;
}
```

**Step 2: Update `src/pages/user/index.tsx` — add notification & coupon entries**

Replace the entire file:

```tsx
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Dialog, Toast, Button } from 'antd-mobile'
import { useAuthStore } from '@/stores/auth'
import { getProfile, type UserProfile } from '@/api/user'
import { getUnreadCount } from '@/api/notification'
import styles from './user.module.css'

export default function UserPage() {
  const navigate = useNavigate()
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const clearAuth = useAuthStore((s) => s.clearAuth)
  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [unread, setUnread] = useState(0)

  useEffect(() => {
    if (isLoggedIn) {
      getProfile().then(setProfile).catch(() => {})
      getUnreadCount().then(setUnread).catch(() => {})
    }
  }, [isLoggedIn])

  const handleLogout = async () => {
    const confirmed = await Dialog.confirm({ content: '确定退出登录？' })
    if (confirmed) {
      clearAuth()
      setProfile(null)
      Toast.show('已退出登录')
      navigate('/', { replace: true })
    }
  }

  const menus: Array<{ label: string; path: string; badge?: number }> = [
    { label: '我的订单', path: '/orders' },
    { label: '我的退款', path: '/refunds' },
    { label: '消息通知', path: '/notifications', badge: unread },
    { label: '优惠券', path: '/coupons' },
    { label: '收货地址', path: '/me/addresses' },
    { label: '编辑资料', path: '/me/profile' },
  ]

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        {isLoggedIn && profile ? (
          <>
            {profile.avatar ? (
              <img className={styles.avatar} src={profile.avatar} alt="" />
            ) : (
              <div className={styles.avatarPlaceholder}>👤</div>
            )}
            <div className={styles.userInfo}>
              <div className={styles.nickname}>{profile.nickname || '用户'}</div>
              <div className={styles.phone}>{profile.phone}</div>
            </div>
          </>
        ) : (
          <>
            <div className={styles.avatarPlaceholder}>👤</div>
            <div className={styles.userInfo}>
              <div className={styles.loginBtn} onClick={() => navigate('/login')}>
                登录 / 注册
              </div>
            </div>
          </>
        )}
      </div>

      <div className={styles.section}>
        {menus.map((item) => (
          <div key={item.path} className={styles.menuItem} onClick={() => navigate(item.path)}>
            <span className={styles.menuLabel}>{item.label}</span>
            {item.badge && item.badge > 0 ? (
              <span className={styles.menuBadge}>{item.badge > 99 ? '99+' : item.badge}</span>
            ) : null}
            <span className={styles.menuArrow}>›</span>
          </div>
        ))}
      </div>

      {isLoggedIn && (
        <div className={styles.logoutSection}>
          <Button className={styles.logoutBtn} onClick={handleLogout}>
            退出登录
          </Button>
        </div>
      )}
    </div>
  )
}
```

**Step 3: Update `src/components/Layout/TabBarLayout.tsx` — add unread badge**

Replace the entire file:

```tsx
import { useState, useEffect } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { TabBar, Badge } from 'antd-mobile'
import {
  AppOutline,
  SearchOutline,
  ShopbagOutline,
  UserOutline,
} from 'antd-mobile-icons'
import { useAuthStore } from '@/stores/auth'
import { getUnreadCount } from '@/api/notification'
import styles from './TabBarLayout.module.css'

export default function TabBarLayout() {
  const location = useLocation()
  const navigate = useNavigate()
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const [unread, setUnread] = useState(0)

  useEffect(() => {
    if (isLoggedIn) {
      getUnreadCount().then(setUnread).catch(() => {})
    } else {
      setUnread(0)
    }
  }, [isLoggedIn, location.pathname])

  const tabs = [
    { key: '/', title: '首页', icon: <AppOutline /> },
    { key: '/search', title: '搜索', icon: <SearchOutline /> },
    { key: '/cart', title: '购物车', icon: <ShopbagOutline /> },
    {
      key: '/me',
      title: '我的',
      icon: unread > 0 ? <Badge content={Badge.dot}><UserOutline /></Badge> : <UserOutline />,
    },
  ]

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

**Step 4: Verify compile**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

---

## Task 9: Router Update + Final Build Verification

**Files:**
- Modify: `frontend/src/router/index.tsx`

**Step 1: Add Phase 3 lazy imports and routes to `src/router/index.tsx`**

After the Phase 2 lazy imports (line 24), add:

```ts
// Phase 3 pages
const NotificationPage = lazy(() => import('@/pages/notification'))
const CouponsPage = lazy(() => import('@/pages/marketing/Coupons'))
const MyCouponsPage = lazy(() => import('@/pages/marketing/MyCoupons'))
const SeckillPage = lazy(() => import('@/pages/marketing/Seckill'))
```

After the last Phase 2 route (the `/refunds/:refundNo` block ending at line 93), add:

```tsx
  // Phase 3 routes
  {
    path: '/notifications',
    element: <AuthGuard><Lazy><NotificationPage /></Lazy></AuthGuard>,
  },
  {
    path: '/coupons',
    element: <Lazy><CouponsPage /></Lazy>,
  },
  {
    path: '/me/coupons',
    element: <AuthGuard><Lazy><MyCouponsPage /></Lazy></AuthGuard>,
  },
  {
    path: '/seckill',
    element: <Lazy><SeckillPage /></Lazy>,
  },
```

Note: `/coupons` and `/seckill` are public (no AuthGuard) — matching the BFF's public endpoints. `/notifications` and `/me/coupons` require auth.

**Step 2: Run full type check**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

**Step 3: Run production build**

Run: `cd frontend && npm run build`
Expected: Successful build with all new chunks visible in output

**Step 4: Verify new routes in build output**

Expect to see these new chunks in the build output:
- `Coupons-*.js`
- `MyCoupons-*.js`
- `Seckill-*.js`
- `notification/index-*.js` (or similar)

---

## Summary

| Task | Files | What It Does |
|------|-------|-------------|
| 1 | 3 files (2 modify, 1 new) | API layer: search history, marketing, notifications |
| 2 | 2 files (2 modify) | Search page: history + autocomplete dropdown |
| 3 | 2 files (2 new) | Notification list page |
| 4 | 2 files (2 new) | Coupon center page (available + my coupons tabs) |
| 5 | 2 files (2 new) | My Coupons standalone page with status tabs |
| 6 | 2 files (2 new) | Seckill activities page with countdown |
| 7 | 2 files (2 modify) | Order confirm coupon selector popup |
| 8 | 3 files (3 modify) | User center entries + TabBar unread badge |
| 9 | 1 file (1 modify) | Router + final build verification |

**Total: 18 files changed, 9 tasks, ~600 lines of TSX + ~350 lines of CSS**
