import { request, client } from './client'

export interface CartItem {
  skuId: number
  productId: number
  quantity: number
  selected: boolean
  productName: string
  productImage: string
  skuSpec: string
  price: number
  stock: number
}

export function getCart() {
  return request<CartItem[]>({ method: 'GET', url: '/cart' })
}

export function addCartItem(params: { skuId: number; productId: number; quantity: number }) {
  return request<void>({ method: 'POST', url: '/cart/items', data: params })
}

export function updateCartItem(skuId: number, params: { quantity?: number; selected?: boolean; updateSelected?: boolean }) {
  return request<void>({ method: 'PUT', url: `/cart/items/${skuId}`, data: params })
}

export function removeCartItem(skuId: number) {
  return client.delete(`/cart/items/${skuId}`)
}

export function clearCart() {
  return client.delete('/cart')
}

export function batchRemove(skuIds: number[]) {
  return request<void>({ method: 'POST', url: '/cart/batch-remove', data: { skuIds: skuIds } })
}
