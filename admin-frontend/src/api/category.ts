import { request } from './client'
import type { Category } from '@/types/product'

export async function listCategories() {
  return request<Category[]>({
    method: 'GET',
    url: '/categories',
  })
}

export interface CategoryData {
  parentId?: number
  name: string
  level?: number
  sort?: number
  icon?: string
  status?: number
}

export async function createCategory(data: CategoryData) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/categories',
    data: { parent_id: data.parentId, name: data.name, level: data.level, sort: data.sort, icon: data.icon, status: data.status },
  })
}

export async function updateCategory(id: number, data: CategoryData) {
  return request<null>({
    method: 'PUT',
    url: `/categories/${id}`,
    data: { parent_id: data.parentId, name: data.name, level: data.level, sort: data.sort, icon: data.icon, status: data.status },
  })
}
