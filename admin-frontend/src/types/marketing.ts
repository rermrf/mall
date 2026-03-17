export interface Coupon {
  id: number
  tenantId: number
  name: string
  type: number
  value: number
  minAmount: number
  scope: number
  status: number
  totalCount: number
  usedCount: number
  startTime: string
  endTime: string
}

export interface SeckillActivity {
  id: number
  tenantId: number
  title: string
  startTime: string
  endTime: string
  status: number
  items: SeckillItem[]
}

export interface SeckillItem {
  id: number
  productId: number
  skuId: number
  seckillPrice: number
  stock: number
  limit: number
}

export interface PromotionRule {
  id: number
  tenantId: number
  name: string
  type: number
  threshold: number
  discount: number
  status: number
  startTime: string
  endTime: string
}
