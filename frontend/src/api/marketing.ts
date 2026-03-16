import { request } from './client'

export interface SeckillActivity {
  id: number
  name: string
  startTime: string
  endTime: string
  status: number
  items: SeckillItem[]
}

export interface SeckillItem {
  id: number
  productName: string
  productImage: string
  originalPrice: number
  seckillPrice: number
  totalStock: number
  availableStock: number
}

export interface Coupon {
  id: number
  name: string
  type: number
  value: number
  minSpend: number
  startTime: string
  endTime: string
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
  couponId: number
  name: string
  type: number
  value: number
  minSpend: number
  startTime: string
  endTime: string
  status: number
}

export interface SeckillResult {
  success: boolean
  message: string
  orderNo: string
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
