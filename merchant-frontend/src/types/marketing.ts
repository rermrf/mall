export interface Coupon {
  id: number
  name: string
  type: number
  threshold: number
  discount_value: number
  total_count: number
  used_count: number
  per_limit: number
  start_time: string
  end_time: string
  scope_type: number
  scope_ids: number[]
  status: number
  created_at: string
}

export interface CreateCouponReq {
  name: string
  type: number
  threshold: number
  discount_value: number
  total_count: number
  per_limit: number
  start_time: string
  end_time: string
  scope_type: number
  scope_ids: number[]
  status: number
}

export interface SeckillActivity {
  id: number
  name: string
  start_time: string
  end_time: string
  status: number
  items: SeckillItem[]
  created_at: string
}

export interface SeckillItem {
  sku_id: number
  seckill_price: number
  seckill_stock: number
  per_limit: number
}

export interface CreateSeckillReq {
  name: string
  start_time: string
  end_time: string
  status: number
  items: SeckillItem[]
}

export interface PromotionRule {
  id: number
  name: string
  type: number
  threshold: number
  discount_value: number
  start_time: string
  end_time: string
  status: number
  created_at: string
}

export interface CreatePromotionReq {
  name: string
  type: number
  threshold: number
  discount_value: number
  start_time: string
  end_time: string
  status: number
}
