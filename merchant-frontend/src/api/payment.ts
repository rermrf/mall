import { request } from './client'
import type { Payment, Refund, RefundReq } from '@/types/payment'

export async function listPayments(params: { status?: number; page?: number; pageSize?: number }) {
  return request<{ payments: Payment[]; total: number }>({ method: 'GET', url: '/payments', params })
}

export async function getPayment(paymentNo: string) {
  return request<Payment>({ method: 'GET', url: `/payments/${paymentNo}` })
}

export async function refundPayment(paymentNo: string, data: RefundReq) {
  return request<{ refund_no: string }>({ method: 'POST', url: `/payments/${paymentNo}/refund`, data })
}

export async function getRefund(refundNo: string) {
  return request<Refund>({ method: 'GET', url: `/refunds/${refundNo}/payment` })
}
