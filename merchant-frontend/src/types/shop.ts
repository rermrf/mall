export interface Shop {
  id: number
  name: string
  logo: string
  description: string
  subdomain: string
  customDomain: string
  plan: string
  status: number
}

export interface UpdateShopReq {
  name: string
  logo: string
  description: string
  subdomain: string
  customDomain: string
}

export interface QuotaInfo {
  type: string
  used: number
  limit: number
}
