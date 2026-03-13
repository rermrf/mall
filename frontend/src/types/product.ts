export interface Product {
  id: number
  name: string
  description: string
  main_image: string
  images: string[]
  price: number      // lowest SKU price in cents
  original_price: number
  sales: number
  category_id: number
  brand_id: number
  status: number
}
