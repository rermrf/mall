# Consumer Frontend Phase 3: Feature Completion Design

**Date:** 2026-03-12
**Status:** Approved
**Scope:** Consume remaining 14 consumer-bff endpoints, add 4 new pages, refactor 3 existing pages + 1 component

---

## Context

Phase 1 MVP (login/signup, home, search, product detail, cart, order confirm, payment) and Phase 2 (profile, addresses, orders, refunds) are complete — 15 pages, 41+ source files, tsc + vite build passing.

Phase 3 closes the consumer experience gap by integrating the remaining unused consumer-bff endpoints: search enhancements, notification center, coupon center, seckill activities, and coupon selection at checkout.

---

## Unused Consumer-BFF Endpoints (14)

| Endpoint | Purpose | Module |
|----------|---------|--------|
| `GET /search/suggestions` | Search autocomplete | Search |
| `GET /search/history` | Search history | Search |
| `DELETE /search/history` | Clear search history | Search |
| `POST /coupons/:id/receive` | Claim coupon | Coupon Center |
| `GET /coupons/mine` | My coupons list | Coupon Center |
| `POST /seckill/:itemId` | Seckill purchase | Seckill |
| `GET /notifications` | Notification list | Notifications |
| `GET /notifications/unread-count` | Unread count | Notifications |
| `PUT /notifications/:id/read` | Mark read | Notifications |
| `PUT /notifications/read-all` | Mark all read | Notifications |

---

## New Pages (4)

| Page | Route | Description |
|------|-------|-------------|
| Notification List | `/notifications` | Notification cards + unread indicator + mark-all-read button |
| Coupon Center | `/coupons` | Two tabs: Available / My Coupons with claim button |
| Seckill Activities | `/seckill` | Activity list with countdown timer + purchase button |
| My Coupons | `/me/coupons` | Tab switch: Available / Used / Expired |

## Existing Page Refactors (3)

| Page | Changes |
|------|---------|
| Search `/search` | Add search history section + autocomplete dropdown (debounce 300ms) + clear history |
| Order Confirm `/order/confirm` | Add coupon selector Popup (list usable coupons, recalculate pay_amount, pass coupon_id) |
| User Center `/me` | Add "Notifications" and "My Coupons" menu entries + unread badge on notifications |

## Component Refactor (1)

| Component | Changes |
|-----------|---------|
| TabBarLayout | Add Badge on "Me" tab showing unread notification count |

---

## Technical Design

### API Layer

- `api/search.ts` — add `getSearchHistory(limit?)`, `clearSearchHistory()` (getSuggestions already exists)
- `api/marketing.ts` — add `receiveCoupon(id)`, `listMyCoupons(status?)`, `seckill(itemId)` + `UserCoupon` type + `SeckillResult` type
- New `api/notification.ts` — `Notification` type + `listNotifications(params)`, `getUnreadCount()`, `markRead(id)`, `markAllRead()`

### Notification Type (from proto)

```ts
interface Notification {
  id: number
  channel: number    // 1=SMS, 2=Email, 3=In-app
  title: string
  content: string
  is_read: boolean
  status: number     // 1=pending, 2=sent, 3=failed
  ctime: string
}
```

### Search Autocomplete

- On SearchBar `onChange`, debounce 300ms, call `getSuggestions(keyword)`
- Render absolutely-positioned dropdown list below search bar
- Click suggestion → trigger search
- Click outside → dismiss

### Seckill Countdown

- `setInterval` every second, compute remaining from `end_time - now`
- States: "Not Started" / countdown HH:MM:SS / "Ended"
- Cleanup interval on unmount

### Coupon Selection at Checkout

- Add clickable coupon area in OrderConfirm
- On click, open Popup listing `listMyCoupons(status=1)` filtered by `min_spend <= totalAmount`
- Selected coupon recalculates displayed pay_amount
- Pass `coupon_id` in createOrder params

### State Management

No new Zustand store needed. Unread count fetched via useEffect in TabBarLayout and UserPage. No global state required.

---

## File Manifest (18 files)

| # | File | Operation |
|---|------|-----------|
| 1 | `src/api/search.ts` | Modify (+2 functions) |
| 2 | `src/api/marketing.ts` | Modify (+3 functions, +2 types) |
| 3 | `src/api/notification.ts` | New |
| 4 | `src/pages/search/index.tsx` | Refactor (history + autocomplete) |
| 5 | `src/pages/search/search.module.css` | Modify |
| 6 | `src/pages/notification/index.tsx` | New |
| 7 | `src/pages/notification/notification.module.css` | New |
| 8 | `src/pages/marketing/Coupons.tsx` | New |
| 9 | `src/pages/marketing/coupons.module.css` | New |
| 10 | `src/pages/marketing/Seckill.tsx` | New |
| 11 | `src/pages/marketing/seckill.module.css` | New |
| 12 | `src/pages/marketing/MyCoupons.tsx` | New |
| 13 | `src/pages/marketing/myCoupons.module.css` | New |
| 14 | `src/pages/order/Confirm.tsx` | Refactor (coupon selector) |
| 15 | `src/pages/order/confirm.module.css` | Modify |
| 16 | `src/pages/user/index.tsx` | Refactor (add entries) |
| 17 | `src/components/Layout/TabBarLayout.tsx` | Refactor (Badge) |
| 18 | `src/router/index.tsx` | Modify (+4 routes) |

**Total: 18 files (5 modify + 5 refactor + 8 new)**

---

## Validation

1. `npx tsc --noEmit` — 0 errors
2. `npm run build` — successful production build
3. Manual route verification: `/notifications`, `/coupons`, `/seckill`, `/me/coupons` all accessible
