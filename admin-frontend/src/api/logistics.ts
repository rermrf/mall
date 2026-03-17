import { request } from './client'
import type { FreightTemplate, Shipment } from '@/types/logistics'

export async function listFreightTemplates(tenantId?: number) {
  return request<FreightTemplate[]>({
    method: 'GET',
    url: '/freight-templates',
    params: { tenant_id: tenantId },
  })
}

export async function getFreightTemplate(id: number) {
  return request<FreightTemplate>({
    method: 'GET',
    url: `/freight-templates/${id}`,
  })
}

export async function getShipment(id: number) {
  return request<Shipment>({
    method: 'GET',
    url: `/shipments/${id}`,
  })
}

export async function getOrderLogistics(orderNo: string) {
  return request<Shipment>({
    method: 'GET',
    url: `/orders/${orderNo}/logistics`,
  })
}
