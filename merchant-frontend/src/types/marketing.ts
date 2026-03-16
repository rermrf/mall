export interface Coupon {
  id: number
  name: string
  type: number
  threshold: number
  discountValue: number
  totalCount: number
  usedCount: number
  perLimit: number
  startTime: string
  endTime: string
  scopeType: number
  scopeIds: number[]
  status: number
  createdAt: string
}

export interface CreateCouponReq {
  name: string
  type: number
  threshold: number
  discountValue: number
  totalCount: number
  perLimit: number
  startTime: number
  endTime: number
  scopeType: number
  scopeIds: string
  status: number
}

export interface SeckillActivity {
  id: number
  name: string
  startTime: string
  endTime: string
  status: number
  items: SeckillItem[]
  createdAt: string
}

export interface SeckillItem {
  skuId: number
  seckillPrice: number
  seckillStock: number
  perLimit: number
}

export interface CreateSeckillReq {
  name: string
  startTime: string
  endTime: string
  status: number
  items: SeckillItem[]
}

export interface PromotionRule {
  id: number
  name: string
  type: number
  threshold: number
  discountValue: number
  startTime: string
  endTime: string
  status: number
  createdAt: string
}

export interface CreatePromotionReq {
  name: string
  type: number
  threshold: number
  discountValue: number
  startTime: string
  endTime: string
  status: number
}
