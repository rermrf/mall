import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Button, Tag, Modal, Input, message } from 'antd'
import { listTenants, approveTenant, freezeTenant } from '@/api/tenant'
import type { Tenant } from '@/types/tenant'

const TENANT_STATUS_MAP: Record<number, { text: string; color: string }> = {
  0: { text: '待审核', color: 'orange' },
  1: { text: '正常', color: 'green' },
  2: { text: '已冻结', color: 'red' },
  3: { text: '已拒绝', color: 'default' },
}

export default function TenantList() {
  const actionRef = useRef<ActionType>()
  const navigate = useNavigate()

  const handleApprove = (id: number, approved: boolean) => {
    if (approved) {
      Modal.confirm({
        title: '审批通过',
        content: '确认通过该租户申请？',
        onOk: async () => {
          await approveTenant(id, true)
          message.success('已通过')
          actionRef.current?.reload()
        },
      })
    } else {
      let reason = ''
      Modal.confirm({
        title: '拒绝申请',
        content: <Input.TextArea placeholder="拒绝原因" onChange={(e) => { reason = e.target.value }} />,
        onOk: async () => {
          await approveTenant(id, false, reason)
          message.success('已拒绝')
          actionRef.current?.reload()
        },
      })
    }
  }

  const handleFreeze = (id: number, freeze: boolean) => {
    Modal.confirm({
      title: freeze ? '冻结租户' : '解冻租户',
      content: freeze ? '确认冻结该租户？' : '确认解冻该租户？',
      onOk: async () => {
        await freezeTenant(id, freeze)
        message.success(freeze ? '已冻结' : '已解冻')
        actionRef.current?.reload()
      },
    })
  }

  const columns: ProColumns<Tenant>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '租户名称', dataIndex: 'name', search: false },
    { title: '联系人', dataIndex: 'contactName', search: false },
    { title: '联系电话', dataIndex: 'contactPhone', search: false },
    {
      title: '状态',
      dataIndex: 'status',
      render: (_, record) => {
        const s = TENANT_STATUS_MAP[record.status] || { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
      valueType: 'select',
      valueEnum: { 0: { text: '待审核' }, 1: { text: '正常' }, 2: { text: '已冻结' }, 3: { text: '已拒绝' } },
    },
    { title: '创建时间', dataIndex: 'createdAt', search: false, valueType: 'dateTime' },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => {
        const actions: React.ReactNode[] = [
          <a key="detail" onClick={() => navigate(`/tenants/${record.id}`)}>详情</a>,
        ]
        if (record.status === 0) {
          actions.push(<a key="approve" onClick={() => handleApprove(record.id, true)}>通过</a>)
          actions.push(<a key="reject" style={{ color: '#ff4d4f' }} onClick={() => handleApprove(record.id, false)}>拒绝</a>)
        }
        if (record.status === 1) {
          actions.push(<a key="freeze" style={{ color: '#ff4d4f' }} onClick={() => handleFreeze(record.id, true)}>冻结</a>)
        }
        if (record.status === 2) {
          actions.push(<a key="unfreeze" onClick={() => handleFreeze(record.id, false)}>解冻</a>)
        }
        return actions
      },
    },
  ]

  return (
    <ProTable<Tenant>
      headerTitle="租户列表"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listTenants({
          page: params.current,
          pageSize: params.pageSize,
          status: params.status !== undefined ? Number(params.status) : undefined,
        })
        return { data: res?.tenants ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
