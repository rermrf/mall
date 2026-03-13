import { request } from './client'
import type { User, Role, CreateRoleReq } from '@/types/user'

export async function getProfile() {
  return request<User>({ method: 'GET', url: '/profile' })
}

export async function updateProfile(data: { nickname: string; avatar: string }) {
  return request<null>({ method: 'PUT', url: '/profile', data })
}

export async function listStaff(params: { page?: number; pageSize?: number }) {
  return request<{ users: User[]; total: number }>({ method: 'GET', url: '/staff', params })
}

export async function assignRole(userId: number, roleId: number) {
  return request<null>({ method: 'POST', url: `/staff/${userId}/role`, data: { role_id: roleId } })
}

export async function listRoles() {
  return request<Role[]>({ method: 'GET', url: '/roles' })
}

export async function createRole(data: CreateRoleReq) {
  return request<{ id: number }>({ method: 'POST', url: '/roles', data })
}

export async function updateRole(id: number, data: CreateRoleReq) {
  return request<null>({ method: 'PUT', url: `/roles/${id}`, data })
}
