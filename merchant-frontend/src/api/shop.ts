import { request } from './client'
import type { Shop, UpdateShopReq, QuotaInfo } from '@/types/shop'

export async function getShop() {
  return request<Shop>({ method: 'GET', url: '/shop' })
}

export async function updateShop(data: UpdateShopReq) {
  return request<null>({ method: 'PUT', url: '/shop', data })
}

export async function checkQuota(type: string) {
  return request<QuotaInfo>({ method: 'GET', url: `/quotas/${type}` })
}
