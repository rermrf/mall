# Fix: Seckill & Promotion Time Parameter Mismatch

**Date:** 2026-03-17
**Status:** Approved & Implemented

## Problem

Creating or updating seckill (flash sale) and promotion (满减) fails because the frontend sends Dayjs date strings while the BFF expects int64 millisecond timestamps.

## Audit Results

Full audit of all merchant frontend API endpoints:

| Module | Endpoint | Frontend Sends | Backend Expects | Status |
|--------|----------|---------------|-----------------|--------|
| Seckill Create/Update | POST/PUT /seckill | Dayjs string | int64 ms | **BUG** |
| Promotion Create/Update | POST/PUT /promotions | Dayjs string | int64 ms | **BUG** |
| Coupon Create/Update | POST/PUT /coupons | ms timestamp | int64 ms | OK |
| Product, Order, Inventory, etc. | — | No time fields | — | N/A |

## Root Cause

`CouponForm.tsx` correctly converts with `new Date(value).getTime()`. `SeckillForm.tsx` and `PromotionList.tsx` were missing this conversion, sending raw ProFormDateTimePicker values (Dayjs strings) directly.

## Fix

3 files changed:

1. **`types/marketing.ts`**: Changed `CreateSeckillReq` and `CreatePromotionReq` time fields from `string` to `number`
2. **`pages/marketing/SeckillForm.tsx`**: Added `new Date(...).getTime()` conversion for startTime/endTime
3. **`pages/marketing/PromotionList.tsx`**: Added same conversion in both create and update handlers
