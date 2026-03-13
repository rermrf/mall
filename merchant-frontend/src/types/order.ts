export interface Order {
  id: number
  order_no: string
  user_id: number
  total_amount: number
  pay_amount: number
  freight_amount: number
  status: number
  payment_no: string
  receiver_name: string
  receiver_phone: string
  receiver_address: string
  remark: string
  items: OrderItem[]
  created_at: string
  updated_at: string
}

export interface OrderItem {
  id: number
  product_id: number
  product_name: string
  product_image: string
  sku_id: number
  sku_code: string
  spec_values: string
  price: number
  quantity: number
}

export interface RefundOrder {
  id: number
  order_no: string
  refund_no: string
  user_id: number
  amount: number
  reason: string
  status: number
  created_at: string
  updated_at: string
}

export interface HandleRefundReq {
  refund_no: string
  approved: boolean
  reason: string
}
