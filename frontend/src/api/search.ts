import { request } from './client'
import type { Product } from '@/types/product'

export interface SearchParams {
  keyword?: string
  category_id?: number
  brand_id?: number
  price_min?: number
  price_max?: number
  sort_by?: string
  page?: number
  page_size?: number
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
