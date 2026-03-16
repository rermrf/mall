import { request } from './client'

export interface Notification {
  id: number
  channel: number
  title: string
  content: string
  isRead: boolean
  status: number
  ctime: string
}

export interface NotificationPageResult {
  notifications: Notification[]
  total: number
}

export function listNotifications(params: {
  channel?: number
  unreadOnly?: boolean
  page: number
  pageSize: number
}) {
  return request<NotificationPageResult>({
    method: 'GET',
    url: '/notifications',
    params,
  })
}

export function getUnreadCount() {
  return request<number>({ method: 'GET', url: '/notifications/unread-count' })
}

export function markRead(id: number) {
  return request<void>({ method: 'PUT', url: `/notifications/${id}/read` })
}

export function markAllRead() {
  return request<void>({ method: 'PUT', url: '/notifications/read-all' })
}

export function deleteNotification(id: number) {
  return request<void>({ method: 'DELETE', url: `/notifications/${id}` })
}
