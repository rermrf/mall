export interface Shop {
  id: number
  name: string
  logo: string
  description: string
  subdomain: string
  custom_domain: string
  plan: string
  status: number
}

export interface UpdateShopReq {
  name: string
  logo: string
  description: string
  subdomain: string
  custom_domain: string
}

export interface QuotaInfo {
  type: string
  used: number
  limit: number
}
