import { request } from './client'
import type { Role } from '@/types/user'

export async function listRoles(tenantId?: number) {
  return request<Role[]>({
    method: 'GET',
    url: '/roles',
    params: { tenant_id: tenantId },
  })
}

export async function createRole(data: { tenantId?: number; name: string; code: string; description: string }) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/roles',
    data: { tenant_id: data.tenantId, name: data.name, code: data.code, description: data.description },
  })
}

export async function updateRole(id: number, data: { name: string; code: string; description: string }) {
  return request<null>({
    method: 'PUT',
    url: `/roles/${id}`,
    data,
  })
}
