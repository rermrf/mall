import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Button, Popconfirm, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { listTemplates, deleteTemplate } from '@/api/notification'
import type { NotificationTemplate } from '@/types/notification'

const CHANNEL_MAP: Record<number, string> = { 1: '站内信', 2: '短信', 3: '邮件' }

export default function TemplateList() {
  const actionRef = useRef<ActionType>()
  const navigate = useNavigate()

  const handleDelete = async (id: number) => {
    await deleteTemplate(id)
    message.success('已删除')
    actionRef.current?.reload()
  }

  const columns: ProColumns<NotificationTemplate>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '模板编码', dataIndex: 'code', search: false },
    { title: '标题', dataIndex: 'title', search: false },
    {
      title: '渠道',
      dataIndex: 'channel',
      render: (_, r) => CHANNEL_MAP[r.channel] || r.channel,
      valueType: 'select',
      valueEnum: { 1: { text: '站内信' }, 2: { text: '短信' }, 3: { text: '邮件' } },
    },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: true,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="edit" onClick={() => navigate(`/notification-templates/${record.id}/edit`)}>编辑</a>,
        <Popconfirm key="delete" title="确认删除？" onConfirm={() => handleDelete(record.id)}>
          <a style={{ color: '#ff4d4f' }}>删除</a>
        </Popconfirm>,
      ],
    },
  ]

  return (
    <ProTable<NotificationTemplate>
      headerTitle="通知模板"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listTemplates({
          tenantId: params.tenantId,
          channel: params.channel !== undefined ? Number(params.channel) : undefined,
        })
        return { data: res ?? [], total: res?.length ?? 0, success: true }
      }}
      toolBarRender={() => [
        <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => navigate('/notification-templates/create')}>
          新建模板
        </Button>,
      ]}
      search={{ labelWidth: 'auto' }}
    />
  )
}
