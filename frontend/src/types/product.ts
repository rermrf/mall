export interface Product {
  id: number
  name: string
  description: string
  mainImage: string
  images: string[]
  price: number      // lowest SKU price in cents
  originalPrice: number
  sales: number
  categoryId: number
  brandId: number
  status: number
}
