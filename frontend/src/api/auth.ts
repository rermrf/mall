import { client } from './client'

export interface LoginParams {
  phone: string
  password: string
}

export interface LoginByPhoneParams {
  phone: string
  code: string
}

export interface SignupParams {
  phone: string
  email: string
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

export async function loginByPhone(params: LoginByPhoneParams) {
  const res = await client.post('/login/phone', params)
  const body = res.data
  if (body.code !== 0) {
    throw new Error(body.msg || '登录失败')
  }
  extractTokens(res.headers as Record<string, string>)
  return body.data
}

export async function signup(params: SignupParams) {
  const res = await client.post('/signup', params)
  return res.data
}

export async function sendSmsCode(phone: string, scene: number = 1) {
  const res = await client.post('/sms/send', { phone, scene })
  return res.data
}

export async function logout() {
  await client.post('/logout', {})
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}

export interface OAuthLoginParams {
  provider: string
  code: string
}

export async function oauthLogin(params: OAuthLoginParams) {
  const res = await client.post('/login/oauth', params)
  const body = res.data
  if (body.code !== 0) {
    throw new Error(body.msg || '登录失败')
  }
  extractTokens(res.headers as Record<string, string>)
  return body.data
}
