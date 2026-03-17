import { client } from './client'

export interface LoginParams {
  phone: string
  password: string
}

function extractTokens(headers: Record<string, string>) {
  const accessToken = headers['x-jwt-token']
  const refreshToken = headers['x-refresh-token']
  if (accessToken) localStorage.setItem('access_token', accessToken)
  if (refreshToken) localStorage.setItem('refresh_token', refreshToken)
}

export async function login(params: LoginParams) {
  const res = await client.post('/login', params)
  const body = res.data
  if (body.code !== 0) {
    throw new Error(body.msg || '登录失败')
  }
  extractTokens(res.headers as Record<string, string>)
  return body.data
}

export async function logout() {
  await client.post('/logout', {})
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}
