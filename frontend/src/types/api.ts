export interface ApiResult<T = unknown> {
  code: number
  msg: string
  data: T
}
