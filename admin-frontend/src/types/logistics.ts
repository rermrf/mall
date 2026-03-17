export interface FreightTemplate {
  id: number
  tenantId: number
  name: string
  chargeType: number
  regions: FreightRegion[]
}

export interface FreightRegion {
  region: string
  firstWeight: number
  firstFee: number
  continueWeight: number
  continueFee: number
}

export interface Shipment {
  id: number
  orderId: number
  orderNo: string
  company: string
  trackingNo: string
  status: number
  createdAt: string
}
