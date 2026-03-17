export interface Inventory {
  skuId: number
  tenantId: number
  stock: number
  locked: number
  available: number
}

export interface InventoryLog {
  id: number
  skuId: number
  tenantId: number
  change: number
  type: string
  orderNo: string
  createdAt: string
}
