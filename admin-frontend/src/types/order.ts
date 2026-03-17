export interface Order {
  id: number
  orderNo: string
  tenantId: number
  userId: number
  totalAmount: number
  payAmount: number
  status: number
  receiverName: string
  receiverPhone: string
  receiverAddress: string
  createdAt: string
  updatedAt: string
  items: OrderItem[]
}

export interface OrderItem {
  id: number
  productId: number
  skuId: number
  title: string
  image: string
  price: number
  quantity: number
}
