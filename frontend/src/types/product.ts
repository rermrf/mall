export interface Product {
  id: number
  name: string
  subtitle?: string
  description: string
  mainImage: string
  images: string  // JSON array string from backend
  price: number
  originalPrice: number
  sales: number
  categoryId: number
  brandId: number
  status: number
  skus?: ProductSKU[]
  specs?: ProductSpec[]
}

export interface ProductSKU {
  id: number
  productId: number
  specValues: string
  price: number
  originalPrice: number
  skuCode: string
  status: number
}

export interface ProductSpec {
  id: number
  productId: number
  name: string
  values: string
}
