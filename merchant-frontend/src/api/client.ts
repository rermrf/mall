import axios from 'axios'
import { message } from 'antd'
import type { ApiResult } from '@/types/api'
import { createRefreshFlow } from './refresh_flow'

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

const refreshFlow = createRefreshFlow()

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (refreshFlow.isRefreshing()) {
        return refreshFlow.waitForToken().then((token) => {
          originalRequest.headers.Authorization = `Bearer ${token}`
          return client(originalRequest)
        })
      }
      originalRequest._retry = true
      refreshFlow.begin()
      try {
        const refreshToken = localStorage.getItem('refresh_token')
        if (!refreshToken) throw new Error('No refresh token')
        const res = await axios.post('/api/v1/refresh-token', {}, {
          headers: refreshFlow.buildRefreshHeaders(refreshToken),
        })
        const newAccessToken = res.headers['x-jwt-token']
        const newRefreshToken = res.headers['x-refresh-token']
        if (newAccessToken) {
          localStorage.setItem('access_token', newAccessToken)
          if (newRefreshToken) localStorage.setItem('refresh_token', newRefreshToken)
          refreshFlow.succeed(newAccessToken)
          originalRequest.headers.Authorization = `Bearer ${newAccessToken}`
          return client(originalRequest)
        }
        throw new Error('No token in refresh response')
      } catch (refreshError) {
        refreshFlow.fail(refreshError)
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        window.location.href = '/login'
        return Promise.reject(refreshError)
      }
    }
    const msg = error.response?.data?.msg || error.message || '网络错误'
    if (error.response?.status !== 401) {
      message.error(msg)
    }
    return Promise.reject(error)
  },
)

export async function request<T>(config: Parameters<typeof client.request>[0]): Promise<T> {
  const res = await client.request(config)
  const body = res.data as ApiResult<T>
  if (body.code !== 0) {
    message.error(body.msg || '请求失败')
    throw new Error(body.msg || '请求失败')
  }
  return body.data
}

export { client }
export default client
