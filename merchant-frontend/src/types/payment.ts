export interface Payment {
  id: number
  paymentNo: string
  orderNo: string
  amount: number
  status: number
  channel: string
  createdAt: string
}

export interface Refund {
  refundNo: string
  paymentNo: string
  amount: number
  reason: string
  status: number
  createdAt: string
}

export interface RefundReq {
  amount: number
  reason: string
}
