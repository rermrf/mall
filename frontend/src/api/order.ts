import { request } from './client'

export interface CreateOrderParams {
  items: Array<{ sku_id: number; quantity: number }>
  address_id: number
  coupon_id?: number
  remark?: string
}

export interface CreateOrderResult {
  order_no: string
  pay_amount: number
}

export interface OrderItem {
  id: number
  order_id: number
  product_id: number
  sku_id: number
  product_name: string
  sku_spec: string
  product_image: string
  price: number
  quantity: number
  total_amount: number
}

export interface Order {
  id: number
  order_no: string
  status: number
  total_amount: number
  discount_amount: number
  freight_amount: number
  pay_amount: number
  receiver_name: string
  receiver_phone: string
  receiver_address: string
  remark: string
  pay_time: number
  ship_time: number
  receive_time: number
  close_time: number
  items: OrderItem[]
  ctime: string
  utime: string
}

// status: 1=待付款, 2=已付款/待发货, 3=已发货, 4=已收货, 5=已完成, 6=已取消, 7=已退款

export interface RefundOrder {
  id: number
  order_id: number
  refund_no: string
  type: number
  status: number
  refund_amount: number
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

export function listOrders(params: { status?: number; page: number; page_size: number }) {
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
  return request<{ refund_no: string }>({ method: 'POST', url: `/orders/${orderNo}/refund`, data: params })
}

export function listRefunds(params: { page: number; page_size: number }) {
  return request<PageResult<RefundOrder>>({ method: 'GET', url: '/refunds', params })
}

export function getRefund(refundNo: string) {
  return request<RefundOrder>({ method: 'GET', url: `/refunds/${refundNo}` })
}
