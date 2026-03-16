import { request } from './client'

export interface StockInfo {
  skuId: number
  available: number
}

export function getStock(skuId: number) {
  return request<StockInfo>({ method: 'GET', url: `/inventory/stock/${skuId}` })
}

export function batchGetStock(skuIds: number[]) {
  return request<StockInfo[]>({ method: 'POST', url: '/inventory/stock/batch', data: { skuIds: skuIds } })
}
