import { request } from './client'
import type { Product, CreateProductReq, Category, CreateCategoryReq, Brand, CreateBrandReq } from '@/types/product'

export async function createProduct(data: CreateProductReq) {
  return request<{ id: number }>({ method: 'POST', url: '/products', data })
}

export async function updateProduct(id: number, data: CreateProductReq) {
  return request<null>({ method: 'PUT', url: `/products/${id}`, data })
}

export async function getProduct(id: number) {
  return request<Product>({ method: 'GET', url: `/products/${id}` })
}

export async function listProducts(params: { category_id?: number; status?: number; page?: number; pageSize?: number }) {
  return request<{ products: Product[]; total: number }>({ method: 'GET', url: '/products', params })
}

export async function updateProductStatus(id: number, status: number) {
  return request<null>({ method: 'PUT', url: `/products/${id}/status`, data: { status } })
}

export async function createCategory(data: CreateCategoryReq) {
  return request<{ id: number }>({ method: 'POST', url: '/categories', data })
}

export async function updateCategory(id: number, data: CreateCategoryReq) {
  return request<null>({ method: 'PUT', url: `/categories/${id}`, data })
}

export async function listCategories() {
  return request<Category[]>({ method: 'GET', url: '/categories' })
}

export async function createBrand(data: CreateBrandReq) {
  return request<{ id: number }>({ method: 'POST', url: '/brands', data })
}

export async function updateBrand(id: number, data: CreateBrandReq) {
  return request<null>({ method: 'PUT', url: `/brands/${id}`, data })
}

export async function listBrands(params?: { page?: number; pageSize?: number }) {
  return request<{ brands: Brand[]; total: number }>({ method: 'GET', url: '/brands', params })
}
