export * from './order'
export * from './payment'
export * from './marketing'

// 通用价格工具
export function formatPrice(fen: number): string {
  return `¥${((fen ?? 0) / 100).toFixed(2)}`
}

export function parsePriceToFen(yuan: number): number {
  return Math.round(yuan * 100)
}
