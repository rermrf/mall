import { request } from './client'

export interface Shop {
  id: number
  tenant_id: number
  name: string
  logo: string
  description: string
  status: number
}

export function getShop() {
  return request<Shop>({ method: 'GET', url: '/shop' })
}
