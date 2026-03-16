import { useRef } from 'react'
import { Button, Tag, message } from 'antd'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listNotifications, markRead, markAllRead } from '@/api/notification'
import { useNotificationStore } from '@/stores/notification'
import type { Notification } from '@/types/notification'

export default function NotificationList() {
  const actionRef = useRef<ActionType>(null)
  const fetchUnreadCount = useNotificationStore((s) => s.fetchUnreadCount)

  const handleMarkRead = async (id: number) => {
    await markRead(id)
    actionRef.current?.reload()
    fetchUnreadCount()
  }

  const handleMarkAllRead = async () => {
    await markAllRead()
    message.success('全部已读')
    actionRef.current?.reload()
    fetchUnreadCount()
  }

  const columns: ProColumns<Notification>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '标题', dataIndex: 'title' },
    { title: '内容', dataIndex: 'content', ellipsis: true, search: false },
    { title: '渠道', dataIndex: 'channel', search: false },
    {
      title: '状态',
      dataIndex: 'isRead',
      search: false,
      render: (_, r) => r.isRead ? <Tag>已读</Tag> : <Tag color="blue">未读</Tag>,
    },
    { title: '时间', dataIndex: 'createdAt', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => !r.isRead ? <a onClick={() => handleMarkRead(r.id)}>标记已读</a> : '-',
    },
  ]

  return (
    <ProTable<Notification>
      headerTitle="消息中心"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      toolBarRender={() => [
        <Button key="readAll" onClick={handleMarkAllRead}>全部已读</Button>,
      ]}
      request={async (params) => {
        const res = await listNotifications({ page: params.current, pageSize: params.pageSize })
        return { data: res?.notifications ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
