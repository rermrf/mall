import { request } from './client'

export interface SeckillActivity {
  id: number
  name: string
  start_time: string
  end_time: string
  status: number
  items: SeckillItem[]
}

export interface SeckillItem {
  id: number
  product_name: string
  product_image: string
  original_price: number
  seckill_price: number
  total_stock: number
  available_stock: number
}

export interface Coupon {
  id: number
  name: string
  type: number
  value: number
  min_spend: number
  start_time: string
  end_time: string
  total: number
  remaining: number
}

export function listSeckillActivities() {
  return request<SeckillActivity[]>({
    method: 'GET',
    url: '/seckill',
  })
}

export function listAvailableCoupons() {
  return request<Coupon[]>({
    method: 'GET',
    url: '/coupons',
  })
}

export interface UserCoupon {
  id: number
  coupon_id: number
  name: string
  type: number
  value: number
  min_spend: number
  start_time: string
  end_time: string
  status: number
}

export interface SeckillResult {
  success: boolean
  message: string
  order_no: string
}

export function receiveCoupon(id: number) {
  return request<void>({ method: 'POST', url: `/coupons/${id}/receive` })
}

export function listMyCoupons(status?: number) {
  return request<UserCoupon[]>({
    method: 'GET',
    url: '/coupons/mine',
    params: status ? { status } : undefined,
  })
}

export function seckill(itemId: number) {
  return request<SeckillResult>({ method: 'POST', url: `/seckill/${itemId}` })
}
