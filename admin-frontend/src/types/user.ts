export interface User {
  id: number
  phone: string
  nickname: string
  avatar: string
  email: string
  role: string
  status: number
  tenantId: number
  createdAt: string
}

export interface Role {
  id: number
  tenantId: number
  name: string
  code: string
  description: string
}
