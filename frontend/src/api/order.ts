import { request } from './client'

export interface CreateOrderParams {
  items: Array<{ skuId: number; quantity: number }>
  addressId: number
  couponId?: number
  remark?: string
}

export interface CreateOrderResult {
  orderNo: string
  payAmount: number
}

export interface OrderItem {
  id: number
  orderId: number
  productId: number
  skuId: number
  productName: string
  skuSpec: string
  productImage: string
  price: number
  quantity: number
  totalAmount: number
}

export interface Order {
  id: number
  orderNo: string
  status: number
  totalAmount: number
  discountAmount: number
  freightAmount: number
  payAmount: number
  receiverName: string
  receiverPhone: string
  receiverAddress: string
  remark: string
  payTime: number
  shipTime: number
  receiveTime: number
  closeTime: number
  items: OrderItem[]
  ctime: string
  utime: string
}

// status: 1=待付款, 2=已付款/待发货, 3=已发货, 4=已收货, 5=已完成, 6=已取消, 7=已退款

export interface RefundOrder {
  id: number
  orderId: number
  refundNo: string
  type: number
  status: number
  refundAmount: number
  reason: string
  ctime: string
  utime: string
}

// refund status: 1=待审核, 2=已通过, 3=退款中, 4=已完成, 5=已拒绝

export interface PageResult<T> {
  list: T[]
  total: number
}

export function createOrder(params: CreateOrderParams) {
  return request<CreateOrderResult>({ method: 'POST', url: '/orders', data: params })
}

export function listOrders(params: { status?: number; page: number; pageSize: number }) {
  return request<PageResult<Order>>({ method: 'GET', url: '/orders', params })
}

export function getOrder(orderNo: string) {
  return request<Order>({ method: 'GET', url: `/orders/${orderNo}` })
}

export function cancelOrder(orderNo: string) {
  return request<void>({ method: 'POST', url: `/orders/${orderNo}/cancel` })
}

export function confirmReceive(orderNo: string) {
  return request<void>({ method: 'POST', url: `/orders/${orderNo}/confirm` })
}

export function applyRefund(orderNo: string, params: { reason: string }) {
  return request<{ refundNo: string }>({ method: 'POST', url: `/orders/${orderNo}/refund`, data: params })
}

export function listRefunds(params: { page: number; pageSize: number }) {
  return request<PageResult<RefundOrder>>({ method: 'GET', url: '/refunds', params })
}

export function getRefund(refundNo: string) {
  return request<RefundOrder>({ method: 'GET', url: `/refunds/${refundNo}` })
}
