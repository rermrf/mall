# Category Browsing & Product Listing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add category browsing and product listing to the consumer frontend, replacing the search tab with a classic category page.

**Architecture:** New BFF endpoints expose existing gRPC `ListCategories` and `ListProducts` RPCs. Frontend adds a category page with left sidebar navigation + right product grid. TabBar second tab changes from "搜索" to "分类", search becomes accessible from a top search bar.

**Tech Stack:** Go/Gin (BFF), React/TypeScript, antd-mobile, CSS Modules

---

### Task 1: BFF — Add ListCategories and ListProducts endpoints

**Files:**
- Modify: `consumer-bff/handler/product.go`
- Modify: `consumer-bff/ioc/gin.go`

**Step 1: Add ListCategories and ListProducts handlers to product.go**

Append after the existing `GetProduct` handler in `consumer-bff/handler/product.go`:

```go
func (h *ProductHandler) ListCategories(ctx *gin.Context) {
	tenantId := ginx.GetTenantID(ctx)
	resp, err := h.productClient.ListCategories(ctx.Request.Context(), &productv1.ListCategoriesRequest{
		TenantId: tenantId,
	})
	if err != nil {
		h.l.Error("查询分类列表失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetCategories()})
}

type ListProductsReq struct {
	CategoryId int64 `form:"categoryId"`
	Page       int32 `form:"page" binding:"required,min=1"`
	PageSize   int32 `form:"pageSize" binding:"required,min=1,max=100"`
}

func (h *ProductHandler) ListProducts(ctx *gin.Context, req ListProductsReq) (ginx.Result, error) {
	tenantId := ginx.GetTenantID(ctx)
	resp, err := h.productClient.ListProducts(ctx.Request.Context(), &productv1.ListProductsRequest{
		TenantId:   tenantId,
		CategoryId: req.CategoryId,
		Status:     2, // 只返回上架商品
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询商品列表失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"products": resp.GetProducts(),
		"total":    resp.GetTotal(),
	}}, nil
}
```

**Step 2: Register routes in gin.go**

Add to the `pub` group in `consumer-bff/ioc/gin.go` (after the existing `pub.GET("/products/:id", ...)` line):

```go
pub.GET("/categories", productHandler.ListCategories)
pub.GET("/products", ginx.WrapQuery[handler.ListProductsReq](l, productHandler.ListProducts))
```

**Step 3: Verify build**

Run: `cd consumer-bff && go build ./...`
Expected: Clean build, no errors

**Step 4: Commit**

```bash
git add consumer-bff/handler/product.go consumer-bff/ioc/gin.go
git commit -m "feat(consumer-bff): add ListCategories and ListProducts endpoints"
```

---

### Task 2: Frontend — Add types and API functions

**Files:**
- Modify: `frontend/src/types/product.ts`
- Modify: `frontend/src/api/product.ts`

**Step 1: Add Category type to types/product.ts**

Append at end of file:

```typescript
export interface Category {
  id: number
  parentId: number
  name: string
  icon: string
  level: number
  sort: number
  status: number
  children?: Category[]
}
```

**Step 2: Add API functions to api/product.ts**

Add imports and functions:

```typescript
import { request } from './client'
import type { Product, Category } from '@/types/product'

export function getProductDetail(id: number) {
  return request<Product>({
    method: 'GET',
    url: `/products/${id}`,
  })
}

export function listCategories() {
  return request<Category[]>({
    method: 'GET',
    url: '/categories',
  })
}

export function listProducts(params: { categoryId?: number; page?: number; pageSize?: number }) {
  return request<{ products: Product[]; total: number }>({
    method: 'GET',
    url: '/products',
    params,
  })
}
```

**Step 3: Verify build**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add frontend/src/types/product.ts frontend/src/api/product.ts
git commit -m "feat(frontend): add Category type and product listing API"
```

---

### Task 3: Frontend — Create Category page

**Files:**
- Create: `frontend/src/pages/category/index.tsx`
- Create: `frontend/src/pages/category/category.module.css`

**Step 1: Create category.module.css**

```css
.page {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.searchBar {
  padding: 8px 12px;
  background: var(--color-card);
}

.body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.sidebar {
  width: 90px;
  flex-shrink: 0;
  background: #f5f5f5;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
}

.sidebarItem {
  padding: 14px 8px;
  text-align: center;
  font-size: 13px;
  color: var(--color-text-secondary);
  border-left: 3px solid transparent;
  cursor: pointer;
  word-break: break-all;
}

.sidebarItemActive {
  background: var(--color-card);
  color: var(--color-accent);
  border-left-color: var(--color-accent);
  font-weight: 600;
}

.main {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  -webkit-overflow-scrolling: touch;
}

.grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.empty {
  text-align: center;
  padding: 60px 0;
  color: var(--color-text-secondary);
  font-size: 14px;
}

.loading {
  text-align: center;
  padding: 40px 0;
}
```

**Step 2: Create category/index.tsx**

```tsx
import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { SearchBar, InfiniteScroll, SpinLoading } from 'antd-mobile'
import { listCategories, listProducts } from '@/api/product'
import ProductCard from '@/components/ProductCard'
import type { Category } from '@/types/product'
import type { Product } from '@/types/product'
import styles from './category.module.css'

export default function CategoryPage() {
  const navigate = useNavigate()
  const [categories, setCategories] = useState<Category[]>([])
  const [activeId, setActiveId] = useState<number>(0)
  const [products, setProducts] = useState<Product[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const mainRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    listCategories().then((list) => {
      const cats = list ?? []
      setCategories(cats)
      if (cats.length > 0) setActiveId(cats[0].id)
    }).catch(() => {}).finally(() => setLoading(false))
  }, [])

  const fetchProducts = useCallback(async (categoryId: number, pageNum: number) => {
    const res = await listProducts({ categoryId, page: pageNum, pageSize: 20 })
    if (pageNum === 1) {
      setProducts(res.products || [])
    } else {
      setProducts((prev) => [...prev, ...(res.products || [])])
    }
    setTotal(res.total || 0)
    setPage(pageNum)
  }, [])

  useEffect(() => {
    if (activeId > 0) {
      setProducts([])
      setPage(1)
      setTotal(0)
      fetchProducts(activeId, 1)
      mainRef.current?.scrollTo(0, 0)
    }
  }, [activeId, fetchProducts])

  const handleCategoryClick = (id: number) => {
    if (id !== activeId) setActiveId(id)
  }

  const loadMore = async () => {
    if (activeId > 0) await fetchProducts(activeId, page + 1)
  }

  if (loading) {
    return <div className={styles.loading}><SpinLoading color="default" /></div>
  }

  return (
    <div className={styles.page}>
      <div className={styles.searchBar}>
        <SearchBar placeholder="搜索商品" onFocus={() => navigate('/search')} />
      </div>
      <div className={styles.body}>
        <div className={styles.sidebar}>
          {categories.map((cat) => (
            <div
              key={cat.id}
              className={`${styles.sidebarItem} ${activeId === cat.id ? styles.sidebarItemActive : ''}`}
              onClick={() => handleCategoryClick(cat.id)}
            >
              {cat.name}
            </div>
          ))}
        </div>
        <div className={styles.main} ref={mainRef}>
          {products.length > 0 ? (
            <div className={styles.grid}>
              {products.map((p) => (
                <ProductCard key={p.id} product={p} />
              ))}
            </div>
          ) : (
            <div className={styles.empty}>该分类下暂无商品</div>
          )}
          <InfiniteScroll loadMore={loadMore} hasMore={products.length < total} />
        </div>
      </div>
    </div>
  )
}
```

**Step 3: Verify build**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors (route not yet registered, but the component compiles)

**Step 4: Commit**

```bash
git add frontend/src/pages/category/
git commit -m "feat(frontend): add category browsing page with left-right layout"
```

---

### Task 4: Frontend — Update router, TabBar, and home page

**Files:**
- Modify: `frontend/src/router/index.tsx`
- Modify: `frontend/src/components/Layout/TabBarLayout.tsx`
- Modify: `frontend/src/pages/home/index.tsx`

**Step 1: Update router/index.tsx**

Add lazy import for CategoryPage:

```typescript
const CategoryPage = lazy(() => import('@/pages/category'))
```

In the TabBarLayout children array, replace the `/search` entry with `/category`:

```typescript
{ path: '/category', element: <Lazy><CategoryPage /></Lazy> },
```

Move `/search` out of the TabBarLayout children — add it as a standalone route (before the `/login` route):

```typescript
{ path: '/search', element: <Lazy><SearchPage /></Lazy> },
```

**Step 2: Update TabBarLayout.tsx**

Replace the SearchOutline import and tab:

```typescript
import {
  AppOutline,
  AppstoreOutline,
  ShopbagOutline,
  UserOutline,
} from 'antd-mobile-icons'
```

Change the tabs array second entry:

```typescript
{ key: '/category', title: '分类', icon: <AppstoreOutline /> },
```

**Step 3: Update home/index.tsx**

Add SearchBar import and a search bar at top of the page (after the shop header, before the seckill section):

```tsx
import { Skeleton, SearchBar } from 'antd-mobile'
```

Add search bar JSX after the header `div` and before the seckill section:

```tsx
<div style={{ padding: '0 12px 12px' }}>
  <SearchBar placeholder="搜索商品" onFocus={() => navigate('/search')} />
</div>
```

**Step 4: Verify build**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add frontend/src/router/index.tsx frontend/src/components/Layout/TabBarLayout.tsx frontend/src/pages/home/index.tsx
git commit -m "feat(frontend): replace search tab with category, add search bar to home"
```

---

### Task 5: Verify end-to-end

**Step 1: Build BFF**

Run: `cd consumer-bff && go build ./...`
Expected: Clean build

**Step 2: Build frontend**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

**Step 3: Manual verification checklist**

- [ ] Home page shows search bar at top
- [ ] Clicking search bar navigates to /search
- [ ] Bottom TabBar shows: 首页 / 分类 / 购物车 / 我的
- [ ] Clicking "分类" tab shows category page
- [ ] Left sidebar lists categories
- [ ] Clicking a category loads products on right side
- [ ] Product cards link to product detail
- [ ] Infinite scroll loads more products
- [ ] Search page still works (accessible from search bar)

**Step 4: Final commit**

If any tweaks were needed, commit them.
