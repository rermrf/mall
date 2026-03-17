export interface Category {
  id: number
  parentId: number
  name: string
  level: number
  sort: number
  icon: string
  status: number
}

export interface Brand {
  id: number
  name: string
  logo: string
  status: number
}
