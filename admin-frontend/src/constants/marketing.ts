export const COUPON_TYPE = {
  FIXED: 1,
  DISCOUNT: 2,
  GIFT: 3,
} as const

export const COUPON_TYPE_OPTIONS = [
  { label: '满减', value: COUPON_TYPE.FIXED },
  { label: '折扣', value: COUPON_TYPE.DISCOUNT },
  { label: '固定金额', value: COUPON_TYPE.GIFT },
]

export const COUPON_SCOPE = {
  ALL: 0,
  PRODUCT: 1,
  CATEGORY: 2,
} as const

export const COUPON_SCOPE_OPTIONS = [
  { label: '全场', value: COUPON_SCOPE.ALL },
  { label: '指定商品', value: COUPON_SCOPE.PRODUCT },
  { label: '指定分类', value: COUPON_SCOPE.CATEGORY },
]
