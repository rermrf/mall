// 支付状态 — 对应后端 domain.PaymentStatusXxx
export const PAYMENT_STATUS = {
  PENDING: 0,
  PAID: 1,
  REFUNDED: 2,
  CLOSED: 3,
} as const

export const PAYMENT_STATUS_MAP: Record<number, { text: string; color: string }> = {
  [PAYMENT_STATUS.PENDING]: { text: '待支付', color: 'orange' },
  [PAYMENT_STATUS.PAID]: { text: '已支付', color: 'green' },
  [PAYMENT_STATUS.REFUNDED]: { text: '已退款', color: 'red' },
  [PAYMENT_STATUS.CLOSED]: { text: '已关闭', color: 'default' },
}
