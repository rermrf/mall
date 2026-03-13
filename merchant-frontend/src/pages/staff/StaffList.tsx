import { useRef, useState, useEffect } from 'react'
import { message, Select } from 'antd'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listStaff, assignRole, listRoles } from '@/api/staff'
import type { User, Role } from '@/types/user'

export default function StaffList() {
  const actionRef = useRef<ActionType>(null)
  const [roles, setRoles] = useState<Role[]>([])

  useEffect(() => {
    listRoles().then((r) => setRoles(r ?? [])).catch(() => {})
  }, [])

  const handleAssignRole = async (userId: number, roleId: number) => {
    try {
      await assignRole(userId, roleId)
      message.success('角色已分配')
      actionRef.current?.reload()
    } catch (e: unknown) {
      message.error((e as Error).message)
    }
  }

  const columns: ProColumns<User>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '昵称', dataIndex: 'nickname' },
    { title: '手机号', dataIndex: 'phone' },
    { title: '角色', dataIndex: 'role', search: false },
    {
      title: '分配角色',
      search: false,
      render: (_, record) => (
        <Select
          style={{ width: 140 }}
          placeholder="选择角色"
          onChange={(v) => handleAssignRole(record.id, v)}
          options={roles.map((r) => ({ label: r.name, value: r.id }))}
        />
      ),
    },
    { title: '加入时间', dataIndex: 'created_at', valueType: 'dateTime', search: false },
  ]

  return (
    <ProTable<User>
      headerTitle="员工管理"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listStaff({ page: params.current, pageSize: params.pageSize })
        return { data: res?.users ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
