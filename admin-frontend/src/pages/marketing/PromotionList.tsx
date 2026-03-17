import { useRef } from 'react'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Tag } from 'antd'
import { listPromotions } from '@/api/marketing'
import type { PromotionRule } from '@/types/marketing'
import { formatPrice } from '@/constants'

export default function PromotionList() {
  const actionRef = useRef<ActionType>()

  const columns: ProColumns<PromotionRule>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '名称', dataIndex: 'name', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '门槛', dataIndex: 'threshold', search: false, render: (_, r) => formatPrice(r.threshold) },
    { title: '优惠', dataIndex: 'discount', search: false, render: (_, r) => formatPrice(r.discount) },
    {
      title: '状态',
      dataIndex: 'status',
      render: (_, r) => {
        const map: Record<number, { text: string; color: string }> = { 0: { text: '未开始', color: 'default' }, 1: { text: '进行中', color: 'green' }, 2: { text: '已结束', color: 'red' } }
        const s = map[r.status] || { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
      valueType: 'select',
      valueEnum: { 0: { text: '未开始' }, 1: { text: '进行中' }, 2: { text: '已结束' } },
    },
    { title: '开始时间', dataIndex: 'startTime', search: false, valueType: 'dateTime' },
    { title: '结束时间', dataIndex: 'endTime', search: false, valueType: 'dateTime' },
  ]

  return (
    <ProTable<PromotionRule>
      headerTitle="促销规则监管"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listPromotions({
          tenantId: params.tenantId,
          status: params.status !== undefined ? Number(params.status) : undefined,
        })
        return { data: res ?? [], total: res?.length ?? 0, success: true }
      }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
