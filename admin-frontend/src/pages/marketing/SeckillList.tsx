import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Tag } from 'antd'
import { listSeckill } from '@/api/marketing'
import type { SeckillActivity } from '@/types/marketing'

export default function SeckillList() {
  const actionRef = useRef<ActionType>()
  const navigate = useNavigate()

  const columns: ProColumns<SeckillActivity>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '标题', dataIndex: 'title', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '开始时间', dataIndex: 'startTime', search: false, valueType: 'dateTime' },
    { title: '结束时间', dataIndex: 'endTime', search: false, valueType: 'dateTime' },
    {
      title: '状态',
      dataIndex: 'status',
      render: (_, r) => {
        const map: Record<number, { text: string; color: string }> = {
          0: { text: '未开始', color: 'default' },
          1: { text: '进行中', color: 'green' },
          2: { text: '已结束', color: 'red' },
        }
        const s = map[r.status] || { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
      valueType: 'select',
      valueEnum: { 0: { text: '未开始' }, 1: { text: '进行中' }, 2: { text: '已结束' } },
    },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="detail" onClick={() => navigate(`/seckill/${record.id}`)}>详情</a>,
      ],
    },
  ]

  return (
    <ProTable<SeckillActivity>
      headerTitle="秒杀活动监管"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listSeckill({
          tenantId: params.tenantId,
          status: params.status !== undefined ? Number(params.status) : undefined,
          page: params.current ?? 1,
          pageSize: params.pageSize ?? 20,
        })
        return { data: res?.activities ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
