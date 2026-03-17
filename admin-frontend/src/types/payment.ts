export interface Payment {
  id: number
  paymentNo: string
  orderNo: string
  amount: number
  status: number
  channel: string
  paidAt: string
  createdAt: string
}

export interface Refund {
  id: number
  refundNo: string
  orderNo: string
  paymentNo: string
  amount: number
  status: number
  reason: string
  createdAt: string
}
