export interface Tenant {
  id: number
  name: string
  contactName: string
  contactPhone: string
  businessLicense: string
  planId: number
  status: number
  createdAt: string
  updatedAt: string
}

export interface TenantPlan {
  id: number
  name: string
  price: number
  durationDays: number
  maxProducts: number
  maxStaff: number
  features: string
}
