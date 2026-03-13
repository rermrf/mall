export interface FreightTemplate {
  id: number
  name: string
  charge_type: number
  free_threshold: number
  rules: FreightRule[]
  created_at: string
}

export interface FreightRule {
  regions: string[]
  first_unit: number
  first_price: number
  additional_unit: number
  additional_price: number
}

export interface CreateFreightTemplateReq {
  name: string
  charge_type: number
  free_threshold: number
  rules: FreightRule[]
}

export interface Shipment {
  order_no: string
  carrier_code: string
  carrier_name: string
  tracking_no: string
  status: number
  created_at: string
}

export interface ShipOrderReq {
  carrier_code: string
  carrier_name: string
  tracking_no: string
}
