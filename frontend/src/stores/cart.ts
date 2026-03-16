import { create } from 'zustand'
import { getCart, updateCartItem, removeCartItem, clearCart, batchRemove, type CartItem } from '@/api/cart'

interface CartState {
  items: CartItem[]
  loading: boolean
  fetchCart: () => Promise<void>
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
    await updateCartItem(skuId, { selected, updateSelected: true })
    set((s) => ({
      items: s.items.map((i) => (i.skuId === skuId ? { ...i, selected } : i)),
    }))
  },

  updateQuantity: async (skuId, quantity) => {
    await updateCartItem(skuId, { quantity })
    set((s) => ({
      items: s.items.map((i) => (i.skuId === skuId ? { ...i, quantity } : i)),
    }))
  },

  remove: async (skuId) => {
    await removeCartItem(skuId)
    set((s) => ({ items: s.items.filter((i) => i.skuId !== skuId) }))
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
