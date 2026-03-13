import { request } from './client'
import type { Order, RefundOrder, HandleRefundReq } from '@/types/order'

export async function listOrders(params: { status?: number; page?: number; pageSize?: number }) {
  return request<{ orders: Order[]; total: number }>({ method: 'GET', url: '/orders', params })
}

export async function getOrder(orderNo: string) {
  return request<Order>({ method: 'GET', url: `/orders/${orderNo}` })
}

export async function handleRefund(orderNo: string, data: HandleRefundReq) {
  return request<null>({ method: 'POST', url: `/orders/${orderNo}/refund/handle`, data })
}

export async function listRefunds(params: { status?: number; page?: number; pageSize?: number }) {
  return request<{ refund_orders: RefundOrder[]; total: number }>({ method: 'GET', url: '/refunds', params })
}
