import { request } from './client'
import type { Tenant } from '@/types/tenant'

export interface ListTenantsParams {
  page?: number
  pageSize?: number
  status?: number
}

export async function listTenants(params: ListTenantsParams) {
  return request<{ tenants: Tenant[]; total: number }>({
    method: 'GET',
    url: '/tenants',
    params: { page: params.page, page_size: params.pageSize, status: params.status },
  })
}

export async function getTenant(id: number) {
  return request<Tenant>({
    method: 'GET',
    url: `/tenants/${id}`,
  })
}

export async function createTenant(data: { name: string; contactName: string; contactPhone: string; businessLicense: string; planId: number }) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/tenants',
    data: { name: data.name, contact_name: data.contactName, contact_phone: data.contactPhone, business_license: data.businessLicense, plan_id: data.planId },
  })
}

export async function approveTenant(id: number, approved: boolean, reason?: string) {
  return request<null>({
    method: 'POST',
    url: `/tenants/${id}/approve`,
    data: { approved, reason },
  })
}

export async function freezeTenant(id: number, freeze: boolean) {
  return request<null>({
    method: 'POST',
    url: `/tenants/${id}/freeze`,
    data: { freeze },
  })
}
