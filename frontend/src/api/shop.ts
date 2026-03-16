import { request } from './client'

export interface Shop {
  id: number
  tenantId: number
  name: string
  logo: string
  description: string
  status: number
}

export function getShop() {
  return request<Shop>({ method: 'GET', url: '/shop' })
}
