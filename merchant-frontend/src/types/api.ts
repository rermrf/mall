export interface ApiResult<T = unknown> {
  code: number
  msg: string
  data: T
}

export interface PageResult<T> {
  list: T[]
  total: number
}

export interface PageParams {
  page?: number
  pageSize?: number
}
