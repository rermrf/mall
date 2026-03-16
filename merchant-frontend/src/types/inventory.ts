export interface Inventory {
  skuId: number
  total: number
  locked: number
  available: number
  alertThreshold: number
}

export interface SetStockReq {
  skuId: number
  total: number
  alertThreshold: number
}

export interface InventoryLog {
  id: number
  skuId: number
  changeType: string
  changeAmount: number
  beforeTotal: number
  afterTotal: number
  orderNo: string
  createdAt: string
}
