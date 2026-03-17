import { request } from './client'
import type { NotificationTemplate } from '@/types/notification'

export interface ListTemplatesParams {
  tenantId?: number
  channel?: number
}

export async function listTemplates(params: ListTemplatesParams) {
  return request<NotificationTemplate[]>({
    method: 'GET',
    url: '/notification-templates',
    params: { tenant_id: params.tenantId, channel: params.channel },
  })
}

export interface TemplateData {
  tenantId?: number
  code: string
  channel: number
  title: string
  content: string
  status?: number
}

export async function createTemplate(data: TemplateData) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/notification-templates',
    data: { tenant_id: data.tenantId, code: data.code, channel: data.channel, title: data.title, content: data.content, status: data.status },
  })
}

export async function updateTemplate(id: number, data: TemplateData) {
  return request<null>({
    method: 'PUT',
    url: `/notification-templates/${id}`,
    data: { tenant_id: data.tenantId, code: data.code, channel: data.channel, title: data.title, content: data.content, status: data.status },
  })
}

export async function deleteTemplate(id: number) {
  return request<null>({
    method: 'DELETE',
    url: `/notification-templates/${id}`,
  })
}

export interface SendNotificationData {
  userId: number
  tenantId?: number
  templateCode: string
  channel: number
  params?: Record<string, string>
}

export async function sendNotification(data: SendNotificationData) {
  return request<{ id: number }>({
    method: 'POST',
    url: '/notifications/send',
    data: { user_id: data.userId, tenant_id: data.tenantId, template_code: data.templateCode, channel: data.channel, params: data.params },
  })
}
