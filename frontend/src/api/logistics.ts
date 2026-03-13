import { request } from './client'

export interface ShipmentTrack {
  description: string
  location: string
  track_time: number
}

export interface Shipment {
  carrier_code: string
  carrier_name: string
  tracking_no: string
  status: number
  tracks: ShipmentTrack[]
}

export function getOrderLogistics(orderNo: string) {
  return request<Shipment>({ method: 'GET', url: `/orders/${orderNo}/logistics` })
}
