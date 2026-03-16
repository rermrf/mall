import { request } from './client'

export interface ShipmentTrack {
  description: string
  location: string
  trackTime: number
}

export interface Shipment {
  carrierCode: string
  carrierName: string
  trackingNo: string
  status: number
  tracks: ShipmentTrack[]
}

export function getOrderLogistics(orderNo: string) {
  return request<Shipment>({ method: 'GET', url: `/orders/${orderNo}/logistics` })
}
