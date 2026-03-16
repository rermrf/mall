import axios from 'axios'
import type { ApiResult } from '@/types/api'

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let isRefreshing = false
let pendingRequests: Array<(token: string) => void> = []

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        return new Promise((resolve) => {
          pendingRequests.push((token: string) => {
            originalRequest.headers.Authorization = `Bearer ${token}`
            resolve(client(originalRequest))
          })
        })
      }
      originalRequest._retry = true
      isRefreshing = true
      try {
        const refreshToken = localStorage.getItem('refresh_token')
        if (!refreshToken) throw new Error('No refresh token')
        const res = await axios.post('/api/v1/refresh-token', {}, {
          headers: { Authorization: `Bearer ${refreshToken}` },
        })
        const newAccessToken = res.headers['x-jwt-token']
        const newRefreshToken = res.headers['x-refresh-token']
        if (newAccessToken) {
          localStorage.setItem('access_token', newAccessToken)
          if (newRefreshToken) localStorage.setItem('refresh_token', newRefreshToken)
          pendingRequests.forEach((cb) => cb(newAccessToken))
          pendingRequests = []
          originalRequest.headers.Authorization = `Bearer ${newAccessToken}`
          return client(originalRequest)
        }
        throw new Error('No token in refresh response')
      } catch {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        window.location.href = '/login'
        return Promise.reject(error)
      } finally {
        isRefreshing = false
      }
    }
    return Promise.reject(error)
  },
)

export async function request<T>(config: Parameters<typeof client.request>[0]): Promise<T> {
  try {
    const res = await client.request(config)
    const body = res.data as ApiResult<T>
    if (body.code !== 0) {
      throw new Error(body.msg || '请求失败')
    }
    return body.data
  } catch (e: unknown) {
    if (e instanceof Error) throw e
    const axiosError = e as { response?: { status: number } }
    if (axiosError.response?.status === 500) {
      throw new Error('服务器繁忙，请稍后重试')
    }
    if (axiosError.response?.status === 404) {
      throw new Error('请求的资源不存在')
    }
    throw new Error('网络连接失败，请检查网络')
  }
}

export { client }
export default client
