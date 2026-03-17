import { request } from './client'
import type { User } from '@/types/user'

export interface ListUsersParams {
  tenantId?: number
  page?: number
  pageSize?: number
  status?: number
  keyword?: string
}

export async function listUsers(params: ListUsersParams) {
  return request<{ users: User[]; total: number }>({
    method: 'GET',
    url: '/users',
    params: { tenant_id: params.tenantId, page: params.page, page_size: params.pageSize, status: params.status, keyword: params.keyword },
  })
}

export async function updateUserStatus(id: number, status: number) {
  return request<null>({
    method: 'POST',
    url: `/users/${id}/status`,
    data: { status },
  })
}
