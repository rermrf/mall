import { create } from 'zustand'
import { getCart, updateCartItem, removeCartItem, clearCart, batchRemove, type CartItem } from '@/api/cart'

interface CartState {
  items: CartItem[]
  loading: boolean
  stockMap: Record<number, number>  // skuId -> available stock
  fetchCart: () => Promise<void>
  fetchStock: () => Promise<void>
  toggleSelect: (skuId: number, selected: boolean) => Promise<void>
  updateQuantity: (skuId: number, quantity: number) => Promise<void>
  remove: (skuId: number) => Promise<void>
  clearAll: () => Promise<void>
  batchRemoveSelected: () => Promise<void>
  selectedItems: () => CartItem[]
  totalAmount: () => number
  totalCount: () => number
}

export const useCartStore = create<CartState>((set, get) => ({
  items: [],
  loading: false,
  stockMap: {},

  fetchCart: async () => {
    set({ loading: true })
    try {
      const items = await getCart()
      set({ items: items || [] })
    } finally {
      set({ loading: false })
    }
  },

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

  clearAll: async () => {
    await clearCart()
    set({ items: [] })
  },

  batchRemoveSelected: async () => {
    const selected = get().selectedItems()
    if (selected.length === 0) return
    const skuIds = selected.map((i) => i.skuId)
    await batchRemove(skuIds)
    set((s) => ({ items: s.items.filter((i) => !i.selected) }))
  },

  selectedItems: () => get().items.filter((i) => i.selected),
  totalAmount: () => get().selectedItems().reduce((sum, i) => sum + i.price * i.quantity, 0),
  totalCount: () => get().items.reduce((sum, i) => sum + i.quantity, 0),
}))
