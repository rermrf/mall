import { request } from './client'
import type { Inventory, SetStockReq, InventoryLog } from '@/types/inventory'

export async function setStock(data: SetStockReq) {
  return request<null>({ method: 'POST', url: '/inventory/stock', data })
}

export async function getStock(skuId: number) {
  return request<Inventory>({ method: 'GET', url: `/inventory/stock/${skuId}` })
}

export async function batchGetStock(skuIds: number[]) {
  return request<Inventory[]>({ method: 'POST', url: '/inventory/stock/batch', data: { sku_ids: skuIds } })
}

export async function listInventoryLogs(params: { sku_id?: number; page?: number; pageSize?: number }) {
  return request<{ logs: InventoryLog[]; total: number }>({ method: 'GET', url: '/inventory/logs', params })
}
