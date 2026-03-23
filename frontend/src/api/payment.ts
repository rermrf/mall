import { request } from './client'

export interface CreatePaymentParams {
  orderId: number
  orderNo: string
  channel: 'mock' | 'wechat' | 'alipay'
}

export interface CreatePaymentResult {
  paymentNo: string
  payUrl: string
}

export interface Payment {
  id: number
  paymentNo: string
  orderNo: string
  channel: string
  amount: number
  status: number
}

export function createPayment(params: CreatePaymentParams) {
  return request<CreatePaymentResult>({ method: 'POST', url: '/payments', data: params })
}

export function getPayment(paymentNo: string) {
  return request<Payment>({ method: 'GET', url: `/payments/${paymentNo}` })
}
