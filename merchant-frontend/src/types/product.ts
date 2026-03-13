export interface Product {
  id: number
  category_id: number
  brand_id: number
  name: string
  subtitle: string
  main_image: string
  images: string[]
  description: string
  status: number
  skus: ProductSKU[]
  specs: ProductSpec[]
  created_at: string
  updated_at: string
}

export interface ProductSKU {
  id: number
  sku_code: string
  price: number
  original_price: number
  cost_price: number
  bar_code: string
  spec_values: string
  status: number
}

export interface ProductSpec {
  name: string
  values: string[]
}

export interface CreateProductReq {
  category_id: number
  brand_id: number
  name: string
  subtitle: string
  main_image: string
  images: string[]
  description: string
  status: number
  skus: CreateSKUReq[]
  specs: ProductSpec[]
}

export interface CreateSKUReq {
  sku_code: string
  price: number
  original_price: number
  cost_price: number
  bar_code: string
  spec_values: string
  status: number
}

export type UpdateProductReq = CreateProductReq

export interface Category {
  id: number
  parent_id: number
  name: string
  level: number
  sort: number
  icon: string
  status: number
  children?: Category[]
}

export interface CreateCategoryReq {
  parent_id: number
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

export interface CreateBrandReq {
  name: string
  logo: string
  status: number
}
