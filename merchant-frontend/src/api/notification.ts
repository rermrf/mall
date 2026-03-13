import { request } from './client'
import type { Notification } from '@/types/notification'

export async function listNotifications(params: { channel?: string; unread_only?: boolean; page?: number; pageSize?: number }) {
  return request<{ notifications: Notification[]; total: number }>({ method: 'GET', url: '/notifications', params })
}

export async function getUnreadCount() {
  return request<number>({ method: 'GET', url: '/notifications/unread-count' })
}

export async function markRead(id: number) {
  return request<null>({ method: 'PUT', url: `/notifications/${id}/read` })
}

export async function markAllRead() {
  return request<null>({ method: 'PUT', url: '/notifications/read-all' })
}
