import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { listFreightTemplates } from '@/api/logistics'
import type { FreightTemplate } from '@/types/logistics'

export default function FreightList() {
  const actionRef = useRef<ActionType>()
  const navigate = useNavigate()

  const columns: ProColumns<FreightTemplate>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '模板名称', dataIndex: 'name', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '计费方式', dataIndex: 'chargeType', search: false, render: (_, r) => r.chargeType === 1 ? '按重量' : '按件数' },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="detail" onClick={() => navigate(`/freight-templates/${record.id}`)}>详情</a>,
      ],
    },
  ]

  return (
    <ProTable<FreightTemplate>
      headerTitle="运费模板监管"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listFreightTemplates(params.tenantId)
        return { data: res ?? [], total: res?.length ?? 0, success: true }
      }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
