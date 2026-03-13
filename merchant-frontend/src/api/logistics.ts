import { request } from './client'
import type { FreightTemplate, CreateFreightTemplateReq, Shipment, ShipOrderReq } from '@/types/logistics'

export async function createFreightTemplate(data: CreateFreightTemplateReq) {
  return request<{ id: number }>({ method: 'POST', url: '/freight-templates', data })
}

export async function updateFreightTemplate(id: number, data: CreateFreightTemplateReq) {
  return request<null>({ method: 'PUT', url: `/freight-templates/${id}`, data })
}

export async function getFreightTemplate(id: number) {
  return request<FreightTemplate>({ method: 'GET', url: `/freight-templates/${id}` })
}

export async function listFreightTemplates() {
  return request<FreightTemplate[]>({ method: 'GET', url: '/freight-templates' })
}

export async function deleteFreightTemplate(id: number) {
  return request<null>({ method: 'DELETE', url: `/freight-templates/${id}` })
}

export async function shipOrder(orderNo: string, data: ShipOrderReq) {
  return request<null>({ method: 'POST', url: `/orders/${orderNo}/ship`, data })
}

export async function getOrderLogistics(orderNo: string) {
  return request<Shipment>({ method: 'GET', url: `/orders/${orderNo}/logistics` })
}
