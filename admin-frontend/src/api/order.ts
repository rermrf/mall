import { request } from './client'
import type { Order } from '@/types/order'

export interface ListOrdersParams {
  tenantId?: number
  status?: number
  page: number
  pageSize: number
}

export async function listOrders(params: ListOrdersParams) {
  return request<{ orders: Order[]; total: number }>({
    method: 'GET',
    url: '/orders',
    params: { tenant_id: params.tenantId, status: params.status, page: params.page, page_size: params.pageSize },
  })
}

export async function getOrder(orderNo: string) {
  return request<Order>({
    method: 'GET',
    url: `/orders/${orderNo}`,
  })
}
