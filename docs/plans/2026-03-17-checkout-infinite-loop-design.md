# Fix: Checkout Infinite Re-render Loop

**Date:** 2026-03-17
**Status:** Approved

## Problem

Clicking checkout navigates to `/order/confirm` which crashes with:

```
Maximum update depth exceeded. This can happen when a component repeatedly
calls setState inside componentWillUpdate or componentDidUpdate.
```

Stack trace points to `forceStoreRerender` → `updateStoreInstance` in React's `useSyncExternalStore`.

## Root Cause

`Confirm.tsx:33` uses a Zustand selector that calls a derived function:

```typescript
const selectedItems = useCartStore((s) => s.selectedItems())
```

`selectedItems()` calls `.filter()`, returning a new array reference every invocation. Zustand's default `Object.is` equality check sees each new reference as a state change. During React's commit phase, `useSyncExternalStore` re-runs the selector to verify consistency, gets a new reference, and triggers `forceStoreRerender` — creating an infinite loop.

## Fix

Replace the selector with an inlined filter and `useShallow` comparator:

```typescript
import { useShallow } from 'zustand/react/shallow'

const selectedItems = useCartStore(
  useShallow((s) => s.items.filter((i) => i.selected))
)
```

`useShallow` performs shallow equality on the array elements, correctly detecting that the cart items haven't changed.

## Scope

- **1 file changed:** `frontend/src/pages/order/Confirm.tsx`
- **2 lines modified:** import + selector
- No other consumers of `selectedItems()` use it inside a React selector
