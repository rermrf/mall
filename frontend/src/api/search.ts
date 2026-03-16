import { request } from './client'
import type { Product } from '@/types/product'

export interface SearchParams {
  keyword?: string
  categoryId?: number
  brandId?: number
  priceMin?: number
  priceMax?: number
  sortBy?: string
  page?: number
  pageSize?: number
}

export interface SearchResult {
  products: Product[]
  total: number
}

export function searchProducts(params: SearchParams) {
  return request<SearchResult>({
    method: 'GET',
    url: '/search',
    params,
  })
}

export function getSuggestions(keyword: string) {
  return request<string[]>({
    method: 'GET',
    url: '/search/suggestions',
    params: { keyword },
  })
}

export function getHotWords() {
  return request<string[]>({
    method: 'GET',
    url: '/search/hot',
  })
}

export function getSearchHistory(limit: number = 20) {
  return request<string[]>({
    method: 'GET',
    url: '/search/history',
    params: { limit },
  })
}

export function clearSearchHistory() {
  return request<void>({ method: 'DELETE', url: '/search/history' })
}
