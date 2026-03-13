import { request, client } from './client'

export interface CartItem {
  sku_id: number
  product_id: number
  quantity: number
  selected: boolean
  product_name: string
  product_image: string
  sku_spec: string
  price: number
  stock: number
}

export function getCart() {
  return request<CartItem[]>({ method: 'GET', url: '/cart' })
}

export function addCartItem(params: { sku_id: number; product_id: number; quantity: number }) {
  return request<void>({ method: 'POST', url: '/cart/items', data: params })
}

export function updateCartItem(skuId: number, params: { quantity?: number; selected?: boolean; update_selected?: boolean }) {
  return request<void>({ method: 'PUT', url: `/cart/items/${skuId}`, data: params })
}

export function removeCartItem(skuId: number) {
  return client.delete(`/cart/items/${skuId}`)
}

export function clearCart() {
  return client.delete('/cart')
}

export function batchRemove(skuIds: number[]) {
  return request<void>({ method: 'POST', url: '/cart/batch-remove', data: { sku_ids: skuIds } })
}
