import { request } from './client'
import type { TenantPlan } from '@/types/tenant'

export async function listPlans() {
  return request<{ plans: TenantPlan[] }>({
    method: 'GET',
    url: '/plans',
  })
}

export interface PlanData {
  name: string
  price: number
  durationDays: number
  maxProducts: number
  maxStaff: number
  features: string
}

export async function createPlan(data: PlanData) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/plans',
    data: { name: data.name, price: data.price, duration_days: data.durationDays, max_products: data.maxProducts, max_staff: data.maxStaff, features: data.features },
  })
}

export async function updatePlan(id: number, data: PlanData) {
  return request<null>({
    method: 'PUT',
    url: `/plans/${id}`,
    data: { name: data.name, price: data.price, duration_days: data.durationDays, max_products: data.maxProducts, max_staff: data.maxStaff, features: data.features },
  })
}
