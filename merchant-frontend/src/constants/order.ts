// 订单状态 — 对应后端 domain.OrderStatusXxx
export const ORDER_STATUS = {
  CANCELLED: 0,
  PENDING: 1,
  PAID: 2,
  SHIPPED: 3,
  COMPLETED: 4,
  REFUNDING: 5,
} as const

export const ORDER_STATUS_MAP: Record<number, { text: string; color: string }> = {
  [ORDER_STATUS.CANCELLED]: { text: '已取消', color: 'default' },
  [ORDER_STATUS.PENDING]: { text: '待付款', color: 'orange' },
  [ORDER_STATUS.PAID]: { text: '待发货', color: 'blue' },
  [ORDER_STATUS.SHIPPED]: { text: '已发货', color: 'cyan' },
  [ORDER_STATUS.COMPLETED]: { text: '已完成', color: 'green' },
  [ORDER_STATUS.REFUNDING]: { text: '退款中', color: 'red' },
}
