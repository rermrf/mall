import { request } from './client'
import type { Brand } from '@/types/product'

export interface ListBrandsParams {
  page: number
  pageSize: number
}

export async function listBrands(params: ListBrandsParams) {
  return request<{ brands: Brand[]; total: number }>({
    method: 'GET',
    url: '/brands',
    params: { page: params.page, page_size: params.pageSize },
  })
}

export async function createBrand(data: { name: string; logo?: string; status?: number }) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/brands',
    data,
  })
}

export async function updateBrand(id: number, data: { name: string; logo?: string; status?: number }) {
  return request<null>({
    method: 'PUT',
    url: `/brands/${id}`,
    data,
  })
}
