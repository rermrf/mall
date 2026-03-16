# Consumer Frontend 全面修复 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all identified issues in the consumer frontend (3 critical bugs, core UX improvements, UX polish) plus 3 missing backend APIs.

**Architecture:** The consumer frontend is a React 19 + TypeScript + antd-mobile mobile app. The backend is a Gin-based BFF (`consumer-bff/`) that proxies to gRPC microservices. Changes span both layers.

**Tech Stack:** React 19, TypeScript 5.9, antd-mobile 5, Zustand, Axios, Vite | Go, Gin, gRPC, Wire, protobuf

---

## Round 1: Critical Bug Fixes

### Task 1: Backend — Add Product Detail API Endpoint

The `GetProduct` gRPC RPC already exists in `product.proto:75`. The `ProductServiceClient` is already initialized in `consumer-bff/ioc/grpc.go:88-91`. We just need a new BFF handler and route.

**Files:**
- Create: `consumer-bff/handler/product.go`
- Modify: `consumer-bff/ioc/gin.go:16-31` (add handler param) and line 42-57 (add public route)
- Modify: `consumer-bff/wire.go:28-41` (add to handlerSet)

**Step 1: Create `consumer-bff/handler/product.go`**

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rermrf/emo/logger"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type ProductHandler struct {
	productClient productv1.ProductServiceClient
	l             logger.Logger
}

func NewProductHandler(productClient productv1.ProductServiceClient, l logger.Logger) *ProductHandler {
	return &ProductHandler{productClient: productClient, l: l}
}

func (h *ProductHandler) GetProduct(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的商品ID"})
		return
	}
	resp, err := h.productClient.GetProduct(ctx.Request.Context(), &productv1.GetProductRequest{
		Id: id,
	})
	if err != nil {
		h.l.Error("查询商品详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetProduct()})
}
```

**Step 2: Register route in `consumer-bff/ioc/gin.go`**

Add `productHandler *handler.ProductHandler` to `InitGinServer()` params (after line 27).

Add route in the `pub` group (after line 53, the marketing public routes):
```go
// 商品详情（公开）
pub.GET("/products/:id", productHandler.GetProduct)
```

**Step 3: Add to Wire in `consumer-bff/wire.go`**

Add `handler.NewProductHandler` to `handlerSet` (after line 39).

**Step 4: Regenerate Wire**

Run: `cd consumer-bff && wire`

**Step 5: Commit**

```bash
git add consumer-bff/handler/product.go consumer-bff/ioc/gin.go consumer-bff/wire.go consumer-bff/wire_gen.go
git commit -m "feat(consumer-bff): add GET /api/v1/products/:id endpoint"
```

---

### Task 2: Frontend — Add Product Detail API + Fix Direct Access

**Files:**
- Create: `frontend/src/api/product.ts`
- Modify: `frontend/src/pages/product/Detail.tsx`
- Modify: `frontend/src/types/product.ts` (add SKU/spec types)

**Step 1: Add product detail types to `frontend/src/types/product.ts`**

```typescript
export interface Product {
  id: number
  name: string
  description: string
  mainImage: string
  images: string  // JSON array string from backend
  price: number
  originalPrice: number
  sales: number
  categoryId: number
  brandId: number
  status: number
  skus?: ProductSKU[]
  specs?: ProductSpec[]
}

export interface ProductSKU {
  id: number
  productId: number
  specValues: string  // JSON: {"颜色":"红","尺码":"XL"}
  price: number
  originalPrice: number
  skuCode: string
  status: number
}

export interface ProductSpec {
  id: number
  productId: number
  name: string      // 规格名：颜色/尺码
  values: string    // JSON array: ["红","蓝","绿"]
}
```

**Step 2: Create `frontend/src/api/product.ts`**

```typescript
import { request } from './client'
import type { Product } from '@/types/product'

export function getProductDetail(id: number) {
  return request<Product>({
    method: 'GET',
    url: `/products/${id}`,
  })
}
```

**Step 3: Rewrite `frontend/src/pages/product/Detail.tsx`**

Key changes:
- Add `loading` state, fetch product by ID if `location.state` has no product
- Add image `Swiper` for multiple images
- Add `Stepper` for quantity selection
- Add "立即购买" flow (distinct from add-to-cart)
- Disable buttons when out of stock
- Show product description

```tsx
import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { NavBar, Button, Toast, Swiper, Stepper, Skeleton, ErrorBlock } from 'antd-mobile'
import { addCartItem } from '@/api/cart'
import { getProductDetail } from '@/api/product'
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

  const stateProduct = (location.state as { product?: Product })?.product
  const [product, setProduct] = useState<Product | null>(stateProduct || null)
  const [stock, setStock] = useState<number | null>(null)
  const [quantity, setQuantity] = useState(1)
  const [adding, setAdding] = useState(false)
  const [loading, setLoading] = useState(!stateProduct)
  const [error, setError] = useState(false)

  // Parse images JSON string to array
  const imageList = useMemo(() => {
    if (!product) return []
    if (product.mainImage) {
      const extras: string[] = []
      if (product.images) {
        try { extras.push(...JSON.parse(product.images)) } catch {}
      }
      return [product.mainImage, ...extras.filter((img) => img !== product.mainImage)]
    }
    return []
  }, [product])

  useEffect(() => {
    if (!id) return
    // Fetch product if not from route state
    if (!stateProduct) {
      setLoading(true)
      getProductDetail(Number(id))
        .then(setProduct)
        .catch(() => setError(true))
        .finally(() => setLoading(false))
    }
    // Always fetch stock
    getStock(Number(id)).then((s) => setStock(s.available)).catch(() => {})
  }, [id, stateProduct])

  if (loading) {
    return (
      <div className={styles.page}>
        <NavBar onBack={() => navigate(-1)}>商品详情</NavBar>
        <Skeleton.Paragraph lineCount={5} animated />
      </div>
    )
  }

  if (error || !product) {
    return (
      <div className={styles.page}>
        <NavBar onBack={() => navigate(-1)}>商品详情</NavBar>
        <ErrorBlock
          status="empty"
          title="商品不存在"
          description="该商品可能已下架或链接有误"
        />
      </div>
    )
  }

  const outOfStock = stock !== null && stock <= 0

  const requireLogin = () => {
    if (!isLoggedIn) {
      navigate(`/login?redirect=${encodeURIComponent(location.pathname)}`, { state: location.state })
      return true
    }
    return false
  }

  const handleAddCart = async () => {
    if (requireLogin()) return
    setAdding(true)
    try {
      await addCartItem({ skuId: Number(id), productId: product.id, quantity })
      Toast.show('已加入购物车')
    } catch (e: unknown) {
      Toast.show((e as Error).message || '添加失败')
    } finally {
      setAdding(false)
    }
  }

  const handleBuyNow = () => {
    if (requireLogin()) return
    navigate('/order/confirm', {
      state: {
        directBuy: true,
        product: { ...product, skuId: Number(id) },
        quantity,
      },
    })
  }

  return (
    <div className={styles.page}>
      <NavBar onBack={() => navigate(-1)} style={{ background: 'transparent', position: 'absolute', zIndex: 10, width: '100%' }} />

      {imageList.length > 1 ? (
        <Swiper className={styles.swiper}>
          {imageList.map((img, i) => (
            <Swiper.Item key={i}>
              <img className={styles.image} src={img} alt={product.name} />
            </Swiper.Item>
          ))}
        </Swiper>
      ) : (
        <img
          className={styles.image}
          src={product.mainImage || 'https://via.placeholder.com/400'}
          alt={product.name}
        />
      )}

      <div className={styles.info}>
        <div className={styles.name}>{product.name}</div>
        {product.subtitle && <div className={styles.subtitle}>{product.subtitle}</div>}
        <div className={styles.priceRow}>
          <Price value={product.price} original={product.originalPrice} size='lg' />
          {product.sales > 0 && <span className={styles.sales}>已售{product.sales}</span>}
        </div>
        {stock !== null && (
          <div className={`${styles.stock} ${stock < 10 ? styles.stockLow : ''}`}>
            {stock > 0 ? `库存 ${stock} 件` : '已售罄'}
          </div>
        )}
      </div>

      <div className={styles.quantitySection}>
        <span className={styles.quantityLabel}>数量</span>
        <Stepper
          min={1}
          max={stock ?? 999}
          value={quantity}
          onChange={(v) => setQuantity(v as number)}
          disabled={outOfStock}
        />
      </div>

      {product.description && (
        <div className={styles.description}>
          <div className={styles.descTitle}>商品详情</div>
          <div className={styles.descText}>{product.description}</div>
        </div>
      )}

      <div className={styles.footer}>
        <Button
          fill='outline'
          className={styles.cartBtn}
          loading={adding}
          onClick={handleAddCart}
          disabled={outOfStock}
        >
          加入购物车
        </Button>
        <Button
          color='primary'
          className={styles.buyBtn}
          onClick={handleBuyNow}
          disabled={outOfStock}
        >
          立即购买
        </Button>
      </div>
    </div>
  )
}
```

**Step 4: Commit**

```bash
git add frontend/src/api/product.ts frontend/src/types/product.ts frontend/src/pages/product/Detail.tsx
git commit -m "feat(frontend): fix product detail page - support direct URL access, add image swiper, quantity selector, buy-now flow"
```

---

### Task 3: Frontend — Fix Order Confirm (No Address + Direct Buy)

**Files:**
- Modify: `frontend/src/pages/order/Confirm.tsx`

**Step 1: Update `OrderConfirm` component**

Key changes at specific locations in the current file:
- At line 26-29: Don't redirect to cart if in `directBuy` mode
- At line 73-88: Add "新增地址" button when `addresses.length === 0`
- At line 12-14: Handle `directBuy` state for items/total

Changes to the `useEffect` (line 25-36):
```tsx
// Extract directBuy data from route state
const locationState = location.state as {
  directBuy?: boolean
  product?: { skuId: number; id: number; name: string; mainImage: string; price: number }
  quantity?: number
} | null

const directBuy = locationState?.directBuy
const directItems = directBuy && locationState?.product
  ? [{
      skuId: locationState.product.skuId,
      productId: locationState.product.id,
      productName: locationState.product.name,
      productImage: locationState.product.mainImage,
      price: locationState.product.price,
      quantity: locationState.quantity || 1,
      selected: true,
    }]
  : null

// Use direct items or cart items
const orderItems = directItems || selectedItems
const orderTotal = orderItems.reduce((sum, i) => sum + i.price * i.quantity, 0)
```

In `useEffect`, change the empty check:
```tsx
useEffect(() => {
  if (orderItems.length === 0) {
    navigate('/cart', { replace: true })
    return
  }
  // ... rest unchanged
}, [])
```

In the address card section (line 73-88), replace the `noAddress` div:
```tsx
<div className={styles.noAddress}>
  {addresses.length === 0 ? (
    <>
      <span>请先添加收货地址</span>
      <Button
        size="mini"
        color="primary"
        style={{ marginLeft: 8 }}
        onClick={() => navigate('/me/addresses/edit', { state: { from: 'order-confirm' } })}
      >
        新增地址
      </Button>
    </>
  ) : '请选择收货地址'}
</div>
```

In `handleSubmit`, use `orderItems` instead of `selectedItems`:
```tsx
items: orderItems.map((i) => ({ skuId: i.skuId, quantity: i.quantity })),
```

Use `orderTotal` instead of `totalAmount` for discount calculation:
```tsx
const usableCoupons = coupons.filter((c) => c.minSpend <= orderTotal)
const discountAmount = selectedCoupon
  ? (selectedCoupon.type === 1 ? selectedCoupon.value : Math.round(orderTotal * (1 - selectedCoupon.value / 100)))
  : 0
const payAmount = Math.max(orderTotal - discountAmount, 0)
```

After successful order, if direct buy don't refetch cart:
```tsx
if (!directBuy) await fetchCart()
```

**Step 2: Commit**

```bash
git add frontend/src/pages/order/Confirm.tsx
git commit -m "fix(frontend): support direct-buy flow and add-address from order confirm page"
```

---

### Task 4: Frontend — Cart Stock Validation

**Files:**
- Modify: `frontend/src/stores/cart.ts`
- Modify: `frontend/src/pages/cart/index.tsx`

**Step 1: Update `stores/cart.ts` — add stock map and optimistic updates**

Add to the state interface:
```typescript
stockMap: Record<number, number>  // skuId -> available stock
fetchStock: () => Promise<void>
```

Add the `fetchStock` method in the store:
```typescript
fetchStock: async () => {
  const items = get().items
  if (items.length === 0) return
  try {
    const { batchGetStock } = await import('@/api/inventory')
    const skuIds = items.map((i) => i.skuId)
    const stocks = await batchGetStock(skuIds)
    const map: Record<number, number> = {}
    for (const s of (stocks || [])) {
      map[s.skuId] = s.available
    }
    set({ stockMap: map })
  } catch {
    // non-critical, ignore
  }
},
```

Initialize `stockMap: {}` in initial state.

**Step 2: Update cart page to use stock data**

In `CartPage`, after `fetchCart()` resolves, call `fetchStock()`:

```tsx
const { items, loading, fetchCart, fetchStock, stockMap, toggleSelect, updateQuantity, remove, clearAll, batchRemoveSelected } = useCartStore()

useEffect(() => {
  fetchCart().then(() => {
    useCartStore.getState().fetchStock()
  })
}, [fetchCart])
```

In the quantity controls (line 102-111), add stock-based disabled state:

```tsx
<div className={styles.quantityControl}>
  <span
    className={`${styles.qtyBtn} ${item.quantity <= 1 ? styles.qtyBtnDisabled : ''}`}
    onClick={() => item.quantity > 1 && updateQuantity(item.skuId, item.quantity - 1)}
  >-</span>
  <span className={styles.qtyValue}>{item.quantity}</span>
  <span
    className={`${styles.qtyBtn} ${stockMap[item.skuId] !== undefined && item.quantity >= stockMap[item.skuId] ? styles.qtyBtnDisabled : ''}`}
    onClick={() => {
      const max = stockMap[item.skuId]
      if (max !== undefined && item.quantity >= max) {
        Toast.show('库存不足')
        return
      }
      updateQuantity(item.skuId, item.quantity + 1)
    }}
  >+</span>
  {stockMap[item.skuId] !== undefined && item.quantity > stockMap[item.skuId] && (
    <span className={styles.stockWarning}>库存不足</span>
  )}
</div>
```

Add `.qtyBtnDisabled` style to `cart.module.css`:
```css
.qtyBtnDisabled {
  opacity: 0.3;
  pointer-events: none;
}
.stockWarning {
  color: var(--adm-color-danger);
  font-size: 11px;
  margin-left: 4px;
}
```

Add checkout stock validation in `handleCheckout`:
```tsx
const handleCheckout = () => {
  if (selectedItems.length === 0) {
    Toast.show('请选择商品')
    return
  }
  const outOfStock = selectedItems.filter((i) => {
    const max = stockMap[i.skuId]
    return max !== undefined && i.quantity > max
  })
  if (outOfStock.length > 0) {
    Toast.show(`${outOfStock[0].productName} 库存不足，请调整数量`)
    return
  }
  navigate('/order/confirm')
}
```

In the empty cart section (line 76-79), add "去逛逛" button:
```tsx
<div style={{ textAlign: 'center', padding: 60, color: 'var(--color-text-secondary)' }}>
  <div>购物车是空的</div>
  <Button
    color="primary"
    fill="outline"
    size="small"
    style={{ marginTop: 16 }}
    onClick={() => navigate('/')}
  >
    去逛逛
  </Button>
</div>
```

**Step 3: Commit**

```bash
git add frontend/src/stores/cart.ts frontend/src/pages/cart/index.tsx frontend/src/pages/cart/cart.module.css
git commit -m "fix(frontend): add stock validation to cart, disable qty buttons at bounds, empty cart CTA"
```

---

## Round 2: Core UX Improvements

### Task 5: Frontend — Search Filters and Sorting

**Files:**
- Modify: `frontend/src/pages/search/index.tsx`
- Modify: `frontend/src/pages/search/search.module.css`

**Step 1: Add sort bar and filter state**

Add new state variables after line 19:
```tsx
const [sortBy, setSortBy] = useState('default')
const [showFilter, setShowFilter] = useState(false)
const [filterCategoryId, setFilterCategoryId] = useState<number | undefined>()
const [filterBrandId, setFilterBrandId] = useState<number | undefined>()
const [filterPriceMin, setFilterPriceMin] = useState('')
const [filterPriceMax, setFilterPriceMax] = useState('')
```

Update `doSearch` to include filter params (modify line 32):
```tsx
const res = await searchProducts({
  keyword: kw,
  page: pageNum,
  pageSize: 20,
  sortBy,
  categoryId: filterCategoryId,
  brandId: filterBrandId,
  priceMin: filterPriceMin ? Number(filterPriceMin) * 100 : undefined,
  priceMax: filterPriceMax ? Number(filterPriceMax) * 100 : undefined,
})
```

Add `sortBy` to `doSearch` deps. When sortBy changes, re-search:
```tsx
useEffect(() => {
  if (keyword.trim() && searched) {
    setProducts([])
    setPage(1)
    doSearch(keyword.trim(), 1)
  }
}, [sortBy]) // eslint-disable-line react-hooks/exhaustive-deps
```

Add sort bar between search bar and results (after line 98):
```tsx
{searched && (
  <div className={styles.sortBar}>
    {[
      { key: 'default', label: '综合' },
      { key: 'sales_desc', label: '销量' },
      { key: 'price_asc', label: '价格↑' },
      { key: 'price_desc', label: '价格↓' },
    ].map((s) => (
      <span
        key={s.key}
        className={`${styles.sortItem} ${sortBy === s.key ? styles.sortItemActive : ''}`}
        onClick={() => setSortBy(s.key)}
      >
        {s.label}
      </span>
    ))}
    <span className={styles.sortItem} onClick={() => setShowFilter(true)}>筛选</span>
  </div>
)}
```

Add filter Popup at the end:
```tsx
<Popup visible={showFilter} onMaskClick={() => setShowFilter(false)} position="right" bodyStyle={{ width: '75vw' }}>
  <div className={styles.filterPanel}>
    <div className={styles.filterTitle}>筛选</div>
    <div className={styles.filterSection}>
      <div className={styles.filterLabel}>价格区间 (元)</div>
      <div className={styles.filterRow}>
        <Input placeholder="最低价" type="number" value={filterPriceMin} onChange={setFilterPriceMin} />
        <span style={{ margin: '0 8px' }}>-</span>
        <Input placeholder="最高价" type="number" value={filterPriceMax} onChange={setFilterPriceMax} />
      </div>
    </div>
    <div className={styles.filterActions}>
      <Button onClick={() => { setFilterPriceMin(''); setFilterPriceMax(''); setFilterCategoryId(undefined); setFilterBrandId(undefined) }}>重置</Button>
      <Button color="primary" onClick={() => { setShowFilter(false); setProducts([]); setPage(1); doSearch(keyword.trim(), 1) }}>确定</Button>
    </div>
  </div>
</Popup>
```

Add `Input, Popup` to antd-mobile imports.

**Step 2: Add CSS for sort bar and filter panel to `search.module.css`**

```css
.sortBar {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  gap: 16px;
  border-bottom: 1px solid var(--adm-border-color);
  background: var(--adm-color-background);
}
.sortItem {
  font-size: 13px;
  color: var(--color-text-secondary);
  cursor: pointer;
}
.sortItemActive {
  color: var(--color-accent);
  font-weight: 600;
}
.filterPanel {
  padding: 16px;
}
.filterTitle {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 16px;
}
.filterSection {
  margin-bottom: 16px;
}
.filterLabel {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}
.filterRow {
  display: flex;
  align-items: center;
}
.filterActions {
  display: flex;
  gap: 12px;
  margin-top: 24px;
}
.filterActions button {
  flex: 1;
}
```

**Step 3: Commit**

```bash
git add frontend/src/pages/search/index.tsx frontend/src/pages/search/search.module.css
git commit -m "feat(frontend): add search sorting and price filter"
```

---

### Task 6: Frontend — Address Cascader Picker + Phone Validation

**Files:**
- Create: `frontend/src/data/regions.ts` (simplified region data)
- Modify: `frontend/src/pages/user/AddressEdit.tsx`

**Step 1: Create `frontend/src/data/regions.ts`**

Create a simplified region data file with common provinces/cities/districts. Use antd-mobile `CascadePicker` format:

```typescript
export interface CascadeOption {
  label: string
  value: string
  children?: CascadeOption[]
}

// Simplified version - in production, use a full dataset package
export const regionData: CascadeOption[] = [
  {
    label: '北京市', value: '北京市',
    children: [{ label: '北京市', value: '北京市', children: [
      { label: '东城区', value: '东城区' },
      { label: '西城区', value: '西城区' },
      { label: '朝阳区', value: '朝阳区' },
      { label: '丰台区', value: '丰台区' },
      { label: '海淀区', value: '海淀区' },
      { label: '石景山区', value: '石景山区' },
      { label: '通州区', value: '通州区' },
      { label: '昌平区', value: '昌平区' },
      { label: '大兴区', value: '大兴区' },
      { label: '顺义区', value: '顺义区' },
    ]}],
  },
  {
    label: '上海市', value: '上海市',
    children: [{ label: '上海市', value: '上海市', children: [
      { label: '黄浦区', value: '黄浦区' },
      { label: '徐汇区', value: '徐汇区' },
      { label: '长宁区', value: '长宁区' },
      { label: '静安区', value: '静安区' },
      { label: '普陀区', value: '普陀区' },
      { label: '虹口区', value: '虹口区' },
      { label: '杨浦区', value: '杨浦区' },
      { label: '浦东新区', value: '浦东新区' },
      { label: '闵行区', value: '闵行区' },
      { label: '宝山区', value: '宝山区' },
    ]}],
  },
  {
    label: '广东省', value: '广东省',
    children: [
      { label: '广州市', value: '广州市', children: [
        { label: '天河区', value: '天河区' },
        { label: '越秀区', value: '越秀区' },
        { label: '海珠区', value: '海珠区' },
        { label: '荔湾区', value: '荔湾区' },
        { label: '番禺区', value: '番禺区' },
        { label: '白云区', value: '白云区' },
      ]},
      { label: '深圳市', value: '深圳市', children: [
        { label: '南山区', value: '南山区' },
        { label: '福田区', value: '福田区' },
        { label: '罗湖区', value: '罗湖区' },
        { label: '宝安区', value: '宝安区' },
        { label: '龙岗区', value: '龙岗区' },
        { label: '龙华区', value: '龙华区' },
      ]},
    ],
  },
  {
    label: '浙江省', value: '浙江省',
    children: [
      { label: '杭州市', value: '杭州市', children: [
        { label: '上城区', value: '上城区' },
        { label: '拱墅区', value: '拱墅区' },
        { label: '西湖区', value: '西湖区' },
        { label: '滨江区', value: '滨江区' },
        { label: '余杭区', value: '余杭区' },
        { label: '萧山区', value: '萧山区' },
      ]},
    ],
  },
  {
    label: '江苏省', value: '江苏省',
    children: [
      { label: '南京市', value: '南京市', children: [
        { label: '玄武区', value: '玄武区' },
        { label: '秦淮区', value: '秦淮区' },
        { label: '建邺区', value: '建邺区' },
        { label: '鼓楼区', value: '鼓楼区' },
        { label: '江宁区', value: '江宁区' },
      ]},
      { label: '苏州市', value: '苏州市', children: [
        { label: '姑苏区', value: '姑苏区' },
        { label: '吴中区', value: '吴中区' },
        { label: '工业园区', value: '工业园区' },
      ]},
    ],
  },
]
```

> Note: This is a simplified dataset. For production, install `china-division` or `@vant/area-data` npm package for full coverage.

**Step 2: Rewrite `frontend/src/pages/user/AddressEdit.tsx`**

Replace province/city/district text inputs with `CascadePicker`:

```tsx
import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { NavBar, Input, Switch, Button, Toast, CascadePicker } from 'antd-mobile'
import { createAddress, updateAddress, type Address } from '@/api/user'
import { regionData } from '@/data/regions'
import styles from './addressEdit.module.css'

const PHONE_REG = /^1[3-9]\d{9}$/

export default function AddressEditPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const existing = (location.state as { address?: Address })?.address

  const [name, setName] = useState(existing?.name || '')
  const [phone, setPhone] = useState(existing?.phone || '')
  const [region, setRegion] = useState<string[]>(
    existing ? [existing.province, existing.city, existing.district].filter(Boolean) : []
  )
  const [regionVisible, setRegionVisible] = useState(false)
  const [detail, setDetail] = useState(existing?.detail || '')
  const [isDefault, setIsDefault] = useState(existing?.isDefault || false)
  const [loading, setLoading] = useState(false)

  const regionText = region.length === 3 ? region.join(' ') : ''

  const handleSave = async () => {
    if (!name.trim()) { Toast.show('请输入姓名'); return }
    if (!phone.trim() || !PHONE_REG.test(phone.trim())) { Toast.show('请输入正确的11位手机号'); return }
    if (region.length < 2) { Toast.show('请选择省市区'); return }
    if (!detail.trim()) { Toast.show('请输入详细地址'); return }

    const params = {
      name: name.trim(),
      phone: phone.trim(),
      province: region[0],
      city: region[1],
      district: region[2] || '',
      detail: detail.trim(),
      isDefault,
    }

    setLoading(true)
    try {
      if (existing) {
        await updateAddress(existing.id, params)
      } else {
        await createAddress(params)
      }
      Toast.show('保存成功')
      navigate(-1)
    } catch (e: unknown) {
      Toast.show((e as Error).message || '保存失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>{existing ? '编辑地址' : '新增地址'}</NavBar>
      </div>

      <div className={styles.card}>
        <div className={styles.field}>
          <div className={styles.label}>姓名</div>
          <Input placeholder="收货人姓名" value={name} onChange={setName} clearable />
        </div>

        <div className={styles.field}>
          <div className={styles.label}>手机号</div>
          <Input placeholder="收货人手机号" type="tel" value={phone} onChange={setPhone} clearable maxLength={11} />
        </div>

        <div className={styles.field} onClick={() => setRegionVisible(true)}>
          <div className={styles.label}>所在地区</div>
          <div className={styles.regionValue}>
            {regionText || <span style={{ color: 'var(--color-text-secondary)' }}>请选择省/市/区</span>}
          </div>
        </div>

        <CascadePicker
          title="选择地区"
          options={regionData}
          visible={regionVisible}
          onClose={() => setRegionVisible(false)}
          onConfirm={(val) => {
            setRegion(val as string[])
            setRegionVisible(false)
          }}
          value={region}
        />

        <div className={styles.field}>
          <div className={styles.label}>详细地址</div>
          <Input placeholder="街道、门牌号等" value={detail} onChange={setDetail} clearable />
        </div>

        <div className={styles.defaultRow}>
          <span className={styles.defaultLabel}>设为默认地址</span>
          <Switch checked={isDefault} onChange={setIsDefault} />
        </div>

        <Button color="primary" className={styles.saveBtn} loading={loading} onClick={handleSave}>
          保存
        </Button>
      </div>
    </div>
  )
}
```

**Step 3: Commit**

```bash
git add frontend/src/data/regions.ts frontend/src/pages/user/AddressEdit.tsx
git commit -m "feat(frontend): replace address text inputs with cascade picker, add phone validation"
```

---

### Task 7: Frontend — Loading States for All Pages

**Files:**
- Modify: `frontend/src/pages/home/index.tsx`
- Modify: `frontend/src/pages/marketing/Coupons.tsx`
- Modify: `frontend/src/pages/marketing/Seckill.tsx`
- Modify: `frontend/src/pages/order/RefundList.tsx`

**Step 1: Add loading skeleton to Home page**

In `home/index.tsx`, add:
```tsx
const [loading, setLoading] = useState(true)
```

In `useEffect`, track loading:
```tsx
useEffect(() => {
  Promise.all([
    getShop().then(setShop).catch(() => {}),
    listSeckillActivities().then((v) => setSeckills(v ?? [])).catch(() => {}),
    listAvailableCoupons().then((v) => setCoupons(v ?? [])).catch(() => {}),
  ]).finally(() => setLoading(false))
}, [])
```

Add loading state at top of render:
```tsx
if (loading) {
  return (
    <div className={styles.page}>
      <Skeleton.Title animated />
      <Skeleton.Paragraph lineCount={5} animated />
    </div>
  )
}
```

Import `Skeleton` from `antd-mobile`.

**Step 2:** Apply the same pattern to Coupons, Seckill, and RefundList pages: add `loading` state, set it in the data fetch, show `Skeleton` or `SpinLoading` while loading.

**Step 3: Commit**

```bash
git add frontend/src/pages/home/index.tsx frontend/src/pages/marketing/Coupons.tsx frontend/src/pages/marketing/Seckill.tsx frontend/src/pages/order/RefundList.tsx
git commit -m "feat(frontend): add loading skeletons to home, coupons, seckill, refund list pages"
```

---

## Round 3: Polish & Detail Fixes

### Task 8: Backend — Add Notification Delete + Refund Cancel RPCs

These require proto changes, code regeneration, and service implementation.

**Files:**
- Modify: `api/proto/notification/v1/notification.proto`
- Modify: `api/proto/order/v1/order.proto`
- Modify: `consumer-bff/handler/notification.go`
- Modify: `consumer-bff/handler/order.go`
- Modify: `consumer-bff/ioc/gin.go`

**Step 1: Add `DeleteNotification` to notification proto**

In `api/proto/notification/v1/notification.proto`, after line 49 (GetUnreadCount):
```protobuf
  // 删除通知
  rpc DeleteNotification(DeleteNotificationRequest) returns (DeleteNotificationResponse);
```

Add at end of file:
```protobuf
message DeleteNotificationRequest {
  int64 id = 1;
  int64 user_id = 2;
}
message DeleteNotificationResponse {}
```

**Step 2: Add `CancelRefund` to order proto**

In `api/proto/order/v1/order.proto`, after line 100 (ListRefundOrders):
```protobuf
  // 取消退款申请（买家主动取消，仅待审核状态可取消）
  rpc CancelRefund(CancelRefundRequest) returns (CancelRefundResponse);
```

Add at end of file:
```protobuf
message CancelRefundRequest {
  string refund_no = 1;
  int64 buyer_id = 2;
}
message CancelRefundResponse {}
```

**Step 3: Regenerate protobuf code**

Run: `make proto` or the project's protobuf generation command.

**Step 4: Implement `DeleteNotification` in notification service**

Find the notification gRPC service implementation and add the `DeleteNotification` method. This should:
1. Verify the notification belongs to the user
2. Delete from database
3. Return success

**Step 5: Implement `CancelRefund` in order service**

Find the order gRPC service implementation and add the `CancelRefund` method. This should:
1. Get the refund order by `refund_no`
2. Verify buyer_id matches and status is 1 (待审核)
3. Delete/cancel the refund record (or set status to a "cancelled" value)
4. Return success

**Step 6: Add BFF handlers**

In `consumer-bff/handler/notification.go`, add:
```go
func (h *NotificationHandler) DeleteNotification(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.notificationClient.DeleteNotification(ctx.Request.Context(), &notificationv1.DeleteNotificationRequest{
		Id:     id,
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("删除通知失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}
```

In `consumer-bff/handler/order.go`, add:
```go
func (h *OrderHandler) CancelRefund(ctx *gin.Context) {
	refundNo := ctx.Param("refundNo")
	uid, _ := ctx.Get("uid")
	_, err := h.orderClient.CancelRefund(ctx.Request.Context(), &orderv1.CancelRefundRequest{
		RefundNo: refundNo,
		BuyerId:  uid.(int64),
	})
	if err != nil {
		h.l.Error("取消退款失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.OrderErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}
```

**Step 7: Register routes in `consumer-bff/ioc/gin.go`**

After line 106 (MarkAllRead):
```go
auth.DELETE("/notifications/:id", notificationHandler.DeleteNotification)
```

After line 81 (GetRefundOrder):
```go
auth.POST("/refunds/:refundNo/cancel", orderHandler.CancelRefund)
```

**Step 8: Regenerate Wire and commit**

```bash
cd consumer-bff && wire
git add api/proto/ consumer-bff/ notification/ order/
git commit -m "feat: add DeleteNotification and CancelRefund RPCs with BFF endpoints"
```

---

### Task 9: Frontend — Notification Enhancements

**Files:**
- Modify: `frontend/src/api/notification.ts`
- Modify: `frontend/src/pages/notification/index.tsx`

**Step 1: Add `deleteNotification` to API**

In `frontend/src/api/notification.ts`:
```typescript
export function deleteNotification(id: number) {
  return request<void>({ method: 'DELETE', url: `/notifications/${id}` })
}
```

**Step 2: Rewrite notification page**

Add channel filter tabs, swipe-to-delete, and collapsible content.

Key changes to `notification/index.tsx`:
- Add `Tabs`, `SwipeAction`, `Collapse` imports
- Add channel tabs: 全部(0) | 系统(3) | 订单(3) | 营销(3) — the `channel` param maps to proto channel values
- Filter `listNotifications` with `channel` param
- Wrap each notification card in `SwipeAction` with delete action
- Make content area collapsible (click to expand full content)

**Step 3: Commit**

```bash
git add frontend/src/api/notification.ts frontend/src/pages/notification/index.tsx
git commit -m "feat(frontend): add notification filtering, swipe-to-delete, expand content"
```

---

### Task 10: Frontend — Refund Cancel Button

**Files:**
- Modify: `frontend/src/api/order.ts`
- Modify: `frontend/src/pages/order/RefundDetail.tsx`

**Step 1: Add `cancelRefund` to API**

In `frontend/src/api/order.ts`:
```typescript
export function cancelRefund(refundNo: string) {
  return request<void>({ method: 'POST', url: `/refunds/${refundNo}/cancel` })
}
```

**Step 2: Add cancel button to RefundDetail**

In `RefundDetail.tsx`, after the timeline card (line 119), add:
```tsx
{refund.status === 1 && (
  <div className={styles.footer}>
    <Button
      color="danger"
      fill="outline"
      onClick={() => {
        Dialog.confirm({
          content: '确定取消退款申请？',
          onConfirm: async () => {
            try {
              await cancelRefund(refund.refundNo)
              Toast.show('已取消退款')
              navigate(-1)
            } catch (e: unknown) {
              Toast.show((e as Error).message || '取消失败')
            }
          },
        })
      }}
    >
      取消退款
    </Button>
  </div>
)}
```

Import `Dialog` from antd-mobile and `cancelRefund` from API.

**Step 3: Commit**

```bash
git add frontend/src/api/order.ts frontend/src/pages/order/RefundDetail.tsx
git commit -m "feat(frontend): add cancel refund button for pending refunds"
```

---

### Task 11: Frontend — Multi-Tab Auth Sync

**Files:**
- Modify: `frontend/src/stores/auth.ts`

**Step 1: Add storage event listener**

```typescript
export const useAuthStore = create<AuthState>((set) => {
  // Listen for auth changes from other tabs
  if (typeof window !== 'undefined') {
    window.addEventListener('storage', (e) => {
      if (e.key === 'access_token') {
        if (!e.newValue) {
          set({ isLoggedIn: false })
          window.location.href = '/login'
        } else {
          set({ isLoggedIn: true })
        }
      }
    })
  }

  return {
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
  }
})
```

**Step 2: Commit**

```bash
git add frontend/src/stores/auth.ts
git commit -m "feat(frontend): sync auth state across browser tabs via storage event"
```

---

### Task 12: Frontend — Cart Optimistic Updates

**Files:**
- Modify: `frontend/src/stores/cart.ts`

**Step 1: Refactor cart store for optimistic updates**

Change `updateQuantity` and `toggleSelect` to update UI first, then call API:

```typescript
updateQuantity: async (skuId, quantity) => {
  const prev = get().items
  // Optimistic update
  set((s) => ({
    items: s.items.map((i) => (i.skuId === skuId ? { ...i, quantity } : i)),
  }))
  try {
    await updateCartItem(skuId, { quantity })
  } catch (e) {
    // Rollback
    set({ items: prev })
    throw e
  }
},

toggleSelect: async (skuId, selected) => {
  const prev = get().items
  set((s) => ({
    items: s.items.map((i) => (i.skuId === skuId ? { ...i, selected } : i)),
  }))
  try {
    await updateCartItem(skuId, { selected, updateSelected: true })
  } catch (e) {
    set({ items: prev })
    throw e
  }
},

remove: async (skuId) => {
  const prev = get().items
  set((s) => ({ items: s.items.filter((i) => i.skuId !== skuId) }))
  try {
    await removeCartItem(skuId)
  } catch (e) {
    set({ items: prev })
    throw e
  }
},
```

**Step 2: Commit**

```bash
git add frontend/src/stores/cart.ts
git commit -m "feat(frontend): add optimistic updates to cart operations with rollback"
```

---

### Task 13: Frontend — Coupon Refresh After Claim

**Files:**
- Modify: `frontend/src/pages/marketing/Coupons.tsx`

**Step 1: Update `handleReceive` to refresh lists**

The current code (line 34-37) already partially does this. Improve it:

```tsx
const handleReceive = async (id: number) => {
  if (!isLoggedIn) {
    navigate('/login')
    return
  }
  try {
    await receiveCoupon(id)
    Toast.show('领取成功')
    // Refresh both lists
    const [newAvailable, newMine] = await Promise.all([
      listAvailableCoupons(),
      listMyCoupons(),
    ])
    setAvailable(newAvailable ?? [])
    setMine(newMine ?? [])
  } catch (e: unknown) {
    Toast.show((e as Error).message || '领取失败')
  }
}
```

**Step 2: Commit**

```bash
git add frontend/src/pages/marketing/Coupons.tsx
git commit -m "fix(frontend): refresh coupon lists after claiming"
```

---

### Task 14: Frontend — UX Detail Fixes

**Files:**
- Modify: `frontend/src/pages/order/Detail.tsx` — copy order number, check existing refund
- Modify: `frontend/src/pages/order/List.tsx` — scroll reset on tab change
- Modify: `frontend/src/pages/auth/Login.tsx` — disable OAuth buttons
- Modify: `frontend/src/pages/auth/Signup.tsx` — phone validation
- Modify: `frontend/src/api/client.ts` — better error messages

**Step 1: Order Detail — copy order number**

In `order/Detail.tsx`, find the order number display (line 188):
```tsx
<span className={styles.infoValue}>
  {order.orderNo}
  <span
    className={styles.copyBtn}
    onClick={() => {
      navigator.clipboard.writeText(order.orderNo).then(
        () => Toast.show('已复制'),
        () => Toast.show('复制失败')
      )
    }}
  >
    复制
  </span>
</span>
```

Add `.copyBtn` to CSS:
```css
.copyBtn {
  margin-left: 8px;
  color: var(--color-accent);
  font-size: 12px;
  cursor: pointer;
}
```

**Step 2: Order Detail — check existing refund before showing button**

In the refund button section (line 239-241), add a check:
```tsx
{(order.status === 2 || order.status === 3) && order.status !== 7 && (
  <Button className={styles.footerBtn} onClick={handleRefund}>申请退款</Button>
)}
```

**Step 3: Order List — scroll to top on tab change**

In `handleTabChange` (line 36-41):
```tsx
const handleTabChange = (key: string) => {
  setActiveTab(key)
  setOrders([])
  setPage(1)
  setHasMore(true)
  window.scrollTo(0, 0)
}
```

**Step 4: Login — disable OAuth buttons**

In `Login.tsx`, replace the OAuth buttons (line 170-178):
```tsx
<div className={styles.oauthBtn} style={{ opacity: 0.4 }}>
  <span className={styles.oauthIcon}>💚</span>
  <span className={styles.oauthLabel}>微信(即将开放)</span>
</div>
<div className={styles.oauthBtn} style={{ opacity: 0.4 }}>
  <span className={styles.oauthIcon}>🔵</span>
  <span className={styles.oauthLabel}>支付宝(即将开放)</span>
</div>
```

Remove the `onClick` handlers.

**Step 5: Signup — phone validation**

In `Signup.tsx`, update `handleSignup` validation:
```tsx
const PHONE_REG = /^1[3-9]\d{9}$/

const handleSignup = async () => {
  if (!phone || !PHONE_REG.test(phone)) {
    Toast.show('请输入正确的11位手机号')
    return
  }
  if (!password || password.length < 6) {
    Toast.show('密码至少6位')
    return
  }
  // ... rest unchanged
}
```

**Step 6: API Client — better error messages**

In `api/client.ts`, enhance the error in `request<T>()` (line 65-72):
```typescript
export async function request<T>(config: Parameters<typeof client.request>[0]): Promise<T> {
  try {
    const res = await client.request(config)
    const body = res.data as ApiResult<T>
    if (body.code !== 0) {
      throw new Error(body.msg || '请求失败')
    }
    return body.data
  } catch (e: unknown) {
    if (e instanceof Error) throw e
    const axiosError = e as { response?: { status: number } }
    if (axiosError.response?.status === 500) {
      throw new Error('服务器繁忙，请稍后重试')
    }
    if (axiosError.response?.status === 404) {
      throw new Error('请求的资源不存在')
    }
    throw new Error('网络连接失败，请检查网络')
  }
}
```

**Step 7: Commit**

```bash
git add frontend/src/pages/order/Detail.tsx frontend/src/pages/order/detail.module.css \
  frontend/src/pages/order/List.tsx frontend/src/pages/auth/Login.tsx \
  frontend/src/pages/auth/Signup.tsx frontend/src/api/client.ts
git commit -m "fix(frontend): UX polish - copy order no, scroll reset, OAuth disabled, phone validation, better errors"
```

---

## Summary

| Round | Tasks | Key Changes |
|-------|-------|-------------|
| **1** | 1-4 | Product detail API + direct access fix, order confirm no-address fix, cart stock validation |
| **2** | 5-7 | Search filters/sorting, address cascader, loading states, buy-now flow |
| **3** | 8-14 | Notification delete/filter, refund cancel, auth sync, optimistic updates, UX polish |

**Total: 14 tasks, ~30 files modified/created**
