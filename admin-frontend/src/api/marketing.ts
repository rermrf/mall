import { request } from './client'
import type { Coupon, SeckillActivity, PromotionRule } from '@/types/marketing'

export interface ListCouponsParams {
  tenantId?: number
  status?: number
  page: number
  pageSize: number
}

export async function listCoupons(params: ListCouponsParams) {
  return request<{ coupons: Coupon[]; total: number }>({
    method: 'GET',
    url: '/coupons',
    params: { tenant_id: params.tenantId, status: params.status, page: params.page, page_size: params.pageSize },
  })
}

export interface ListSeckillParams {
  tenantId?: number
  status?: number
  page: number
  pageSize: number
}

export async function listSeckill(params: ListSeckillParams) {
  return request<{ activities: SeckillActivity[]; total: number }>({
    method: 'GET',
    url: '/seckill',
    params: { tenant_id: params.tenantId, status: params.status, page: params.page, page_size: params.pageSize },
  })
}

export async function getSeckill(id: number) {
  return request<SeckillActivity>({
    method: 'GET',
    url: `/seckill/${id}`,
  })
}

export async function listPromotions(params: { tenantId?: number; status?: number }) {
  return request<PromotionRule[]>({
    method: 'GET',
    url: '/promotions',
    params: { tenant_id: params.tenantId, status: params.status },
  })
}
