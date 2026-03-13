export interface Inventory {
  sku_id: number
  total: number
  locked: number
  available: number
  alert_threshold: number
}

export interface SetStockReq {
  sku_id: number
  total: number
  alert_threshold: number
}

export interface InventoryLog {
  id: number
  sku_id: number
  change_type: string
  change_amount: number
  before_total: number
  after_total: number
  order_no: string
  created_at: string
}
