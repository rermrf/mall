import { request } from './client'

export interface Address {
  id: number
  name: string
  phone: string
  province: string
  city: string
  district: string
  detail: string
  is_default: boolean
}

export interface UserProfile {
  id: number
  phone: string
  email: string
  nickname: string
  avatar: string
}

export function getProfile() {
  return request<UserProfile>({ method: 'GET', url: '/profile' })
}

export function updateProfile(params: { nickname: string; avatar: string }) {
  return request<UserProfile>({ method: 'PUT', url: '/profile', data: params })
}

export function listAddresses() {
  return request<Address[]>({ method: 'GET', url: '/addresses' })
}

export function createAddress(params: Omit<Address, 'id'>) {
  return request<{ id: number }>({ method: 'POST', url: '/addresses', data: params })
}

export function updateAddress(id: number, params: Omit<Address, 'id'>) {
  return request<void>({ method: 'PUT', url: `/addresses/${id}`, data: params })
}

export function deleteAddress(id: number) {
  return request<void>({ method: 'DELETE', url: `/addresses/${id}` })
}
