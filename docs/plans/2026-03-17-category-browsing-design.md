# Feature: Category Browsing & Product Listing

**Date:** 2026-03-17
**Status:** Approved

## Problem

The consumer frontend has no way to browse categories or products. Users can only find products through keyword search or the seckill section on the home page. The original design specified a "分类" tab, but it was implemented as "搜索" instead.

## Current State

- gRPC service has `ListCategories` and `ListProducts` RPCs — **implemented**
- Consumer BFF has no HTTP endpoints for categories/products listing — **missing**
- Frontend has no category page, no product listing, no category API — **missing**
- TabBar shows "搜索" where "分类" should be

## Design

### BFF Layer (consumer-bff)

New endpoints in `handler/product.go`:

| Method | Path | Handler | gRPC Call |
|--------|------|---------|-----------|
| GET | /api/v1/categories | ListCategories | productClient.ListCategories |
| GET | /api/v1/products | ListProducts | productClient.ListProducts(categoryId, page, pageSize) |

Both registered as public endpoints (no auth required).

### Frontend

**New files:**
- `src/pages/category/index.tsx` — Category browsing page (left-right layout)
- `src/pages/category/category.module.css` — Styles

**Modified files:**
- `src/api/product.ts` — Add `listCategories()`, `listProducts()`
- `src/types/product.ts` — Add `Category` interface
- `src/router/index.tsx` — Add `/category` route in TabBar, move `/search` out of TabBar
- `src/components/Layout/TabBarLayout.tsx` — Change 2nd tab from "搜索" to "分类"
- `src/pages/home/index.tsx` — Add search bar at top

**Category page layout:**
```
┌────────────────────────┐
│  🔍 搜索栏（点击跳搜索页）  │
├─────┬──────────────────┤
│ 一级 │  商品卡片网格      │
│ 分类 │  图片+名称+价格    │
│ 导航 │  点击进详情页      │
│ 列表 │                  │
├─────┴──────────────────┤
│ 首页  分类  购物车  我的  │
└────────────────────────┘
```

- Left sidebar: vertical list of top-level categories, selected state highlighted
- Right area: products grid under selected category, with pagination/infinite scroll
- Top: search bar that navigates to /search on tap
