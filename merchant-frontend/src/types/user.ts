export interface User {
  id: number
  phone: string
  nickname: string
  avatar: string
  email: string
  role: string
  createdAt: string
}

export interface Role {
  id: number
  name: string
  code: string
  description: string
}

export interface CreateRoleReq {
  name: string
  code: string
  description: string
}
