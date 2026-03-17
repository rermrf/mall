import { request } from './client'
import type { Product, Category } from '@/types/product'

export function getProductDetail(id: number) {
  return request<Product>({
    method: 'GET',
    url: `/products/${id}`,
  })
}

export function listCategories() {
  return request<Category[]>({
    method: 'GET',
    url: '/categories',
  })
}

export function listProducts(params: { categoryId?: number; page?: number; pageSize?: number }) {
  return request<{ products: Product[]; total: number }>({
    method: 'GET',
    url: '/products',
    params,
  })
}
