export interface Payment {
  id: number
  payment_no: string
  order_no: string
  amount: number
  status: number
  channel: string
  created_at: string
}

export interface Refund {
  refund_no: string
  payment_no: string
  amount: number
  reason: string
  status: number
  created_at: string
}

export interface RefundReq {
  amount: number
  reason: string
}
