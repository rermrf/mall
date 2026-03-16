import { request } from './client'
import type { Product } from '@/types/product'

export function getProductDetail(id: number) {
  return request<Product>({
    method: 'GET',
    url: `/products/${id}`,
  })
}
