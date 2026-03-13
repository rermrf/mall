import { request } from './client'

export interface CreatePaymentParams {
  order_id: number
  order_no: string
  channel: 'mock' | 'wechat' | 'alipay'
  amount: number
}

export interface CreatePaymentResult {
  payment_no: string
  pay_url: string
}

export interface Payment {
  id: number
  payment_no: string
  order_no: string
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
