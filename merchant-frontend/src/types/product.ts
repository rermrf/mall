export interface Product {
  id: number
  categoryId: number
  brandId: number
  name: string
  subtitle: string
  mainImage: string
  images: string[]
  description: string
  status: number
  skus: ProductSKU[]
  specs: ProductSpec[]
  createdAt: string
  updatedAt: string
}

export interface ProductSKU {
  id: number
  skuCode: string
  price: number
  originalPrice: number
  costPrice: number
  barCode: string
  specValues: string
  status: number
}

export interface ProductSpec {
  name: string
  values: string[]
}

export interface CreateProductReq {
  categoryId: number
  brandId: number
  name: string
  subtitle: string
  mainImage: string
  images: string[]
  description: string
  status: number
  skus: CreateSKUReq[]
  specs: ProductSpec[]
}

export interface CreateSKUReq {
  skuCode: string
  price: number
  originalPrice: number
  costPrice: number
  barCode: string
  specValues: string
  status: number
}

export type UpdateProductReq = CreateProductReq

export interface Category {
  id: number
  parentId: number
  name: string
  level: number
  sort: number
  icon: string
  status: number
  children?: Category[]
}

export interface CreateCategoryReq {
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

export interface CreateBrandReq {
  name: string
  logo: string
  status: number
}
