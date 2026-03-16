import { request } from './client'
import type { Coupon, CreateCouponReq, SeckillActivity, CreateSeckillReq, PromotionRule, CreatePromotionReq } from '@/types/marketing'

export async function createCoupon(data: CreateCouponReq) {
  return request<{ id: number }>({ method: 'POST', url: '/coupons', data })
}

export async function updateCoupon(id: number, data: CreateCouponReq) {
  return request<null>({ method: 'PUT', url: `/coupons/${id}`, data })
}

export async function listCoupons(params: { status?: number; page?: number; pageSize?: number }) {
  return request<{ coupons: Coupon[]; total: number }>({ method: 'GET', url: '/coupons', params })
}

export async function getCoupon(id: number) {
  return request<Coupon>({ method: 'GET', url: `/coupons/${id}` })
}

export async function createSeckill(data: CreateSeckillReq) {
  return request<{ id: number }>({ method: 'POST', url: '/seckill', data })
}

export async function updateSeckill(id: number, data: CreateSeckillReq) {
  return request<null>({ method: 'PUT', url: `/seckill/${id}`, data })
}

export async function listSeckill(params: { status?: number; page?: number; pageSize?: number }) {
  return request<{ activities: SeckillActivity[]; total: number }>({ method: 'GET', url: '/seckill', params })
}

export async function getSeckill(id: number) {
  return request<SeckillActivity>({ method: 'GET', url: `/seckill/${id}` })
}

export async function createPromotion(data: CreatePromotionReq) {
  return request<{ id: number }>({ method: 'POST', url: '/promotions', data })
}

export async function updatePromotion(id: number, data: CreatePromotionReq) {
  return request<null>({ method: 'PUT', url: `/promotions/${id}`, data })
}

export async function listPromotions(params?: { status?: number }) {
  return request<PromotionRule[]>({ method: 'GET', url: '/promotions', params })
}
