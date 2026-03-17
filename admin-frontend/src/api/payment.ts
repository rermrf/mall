import { request } from './client'
import type { Payment, Refund } from '@/types/payment'

export async function getPayment(paymentNo: string) {
  return request<Payment>({
    method: 'GET',
    url: `/payments/${paymentNo}`,
  })
}

export async function getRefund(refundNo: string) {
  return request<Refund>({
    method: 'GET',
    url: `/refunds/${refundNo}`,
  })
}
