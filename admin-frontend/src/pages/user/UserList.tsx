import { useRef } from 'react'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Switch, message } from 'antd'
import { listUsers, updateUserStatus } from '@/api/user'
import type { User } from '@/types/user'

export default function UserList() {
  const actionRef = useRef<ActionType>()

  const handleStatusChange = async (id: number, checked: boolean) => {
    try {
      await updateUserStatus(id, checked ? 1 : 0)
      message.success('状态更新成功')
      actionRef.current?.reload()
    } catch { /* handled by interceptor */ }
  }

  const columns: ProColumns<User>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '手机号', dataIndex: 'phone', search: false },
    { title: '昵称', dataIndex: 'nickname', search: false },
    { title: '邮箱', dataIndex: 'email', search: false },
    { title: '角色', dataIndex: 'role', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: true,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '搜索手机号/昵称' },
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (_, record) => (
        <Switch
          checked={record.status === 1}
          onChange={(checked) => handleStatusChange(record.id, checked)}
          checkedChildren="启用"
          unCheckedChildren="禁用"
        />
      ),
      valueType: 'select',
      valueEnum: { 0: { text: '禁用' }, 1: { text: '启用' } },
    },
    { title: '注册时间', dataIndex: 'createdAt', search: false, valueType: 'dateTime' },
  ]

  return (
    <ProTable<User>
      headerTitle="用户列表"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listUsers({
          tenantId: params.tenantId,
          page: params.current,
          pageSize: params.pageSize,
          status: params.status !== undefined ? Number(params.status) : undefined,
          keyword: params.keyword,
        })
        return { data: res?.users ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
