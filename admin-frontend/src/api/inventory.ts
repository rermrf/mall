import { request } from './client'
import type { Inventory, InventoryLog } from '@/types/inventory'

export async function getStock(skuId: number) {
  return request<Inventory>({
    method: 'GET',
    url: `/inventory/${skuId}`,
  })
}

export async function batchGetStock(skuIds: number[]) {
  return request<Inventory[]>({
    method: 'POST',
    url: '/inventory/batch',
    data: { sku_ids: skuIds },
  })
}

export interface ListLogsParams {
  tenantId?: number
  skuId?: number
  page: number
  pageSize: number
}

export async function listLogs(params: ListLogsParams) {
  return request<{ logs: InventoryLog[]; total: number }>({
    method: 'GET',
    url: '/inventory/logs',
    params: { tenant_id: params.tenantId, sku_id: params.skuId, page: params.page, page_size: params.pageSize },
  })
}
