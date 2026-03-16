export interface Order {
  id: number
  orderNo: string
  userId: number
  totalAmount: number
  payAmount: number
  freightAmount: number
  status: number
  paymentNo: string
  receiverName: string
  receiverPhone: string
  receiverAddress: string
  remark: string
  items: OrderItem[]
  createdAt: string
  updatedAt: string
}

export interface OrderItem {
  id: number
  productId: number
  productName: string
  productImage: string
  skuId: number
  skuCode: string
  specValues: string
  price: number
  quantity: number
}

export interface RefundOrder {
  id: number
  orderNo: string
  refundNo: string
  userId: number
  amount: number
  reason: string
  status: number
  createdAt: string
  updatedAt: string
}

export interface HandleRefundReq {
  refundNo: string
  approved: boolean
  reason: string
}
