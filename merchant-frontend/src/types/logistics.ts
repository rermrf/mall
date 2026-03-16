export interface FreightTemplate {
  id: number
  name: string
  chargeType: number
  freeThreshold: number
  rules: FreightRule[]
  createdAt: string
}

export interface FreightRule {
  regions: string[]
  firstUnit: number
  firstPrice: number
  additionalUnit: number
  additionalPrice: number
}

export interface CreateFreightTemplateReq {
  name: string
  chargeType: number
  freeThreshold: number
  rules: FreightRule[]
}

export interface Shipment {
  orderNo: string
  carrierCode: string
  carrierName: string
  trackingNo: string
  status: number
  createdAt: string
}

export interface ShipOrderReq {
  carrierCode: string
  carrierName: string
  trackingNo: string
}
