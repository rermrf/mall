import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Tag } from 'antd'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listOrders } from '@/api/order'
import { ORDER_STATUS_MAP, formatPrice } from '@/constants'
import type { Order } from '@/types/order'

export default function OrderList() {
  const navigate = useNavigate()
  const actionRef = useRef<ActionType>(null)

  const columns: ProColumns<Order>[] = [
    { title: '订单号', dataIndex: 'orderNo', copyable: true },
    {
      title: '金额',
      dataIndex: 'payAmount',
      search: false,
      render: (_, r) => formatPrice(r.payAmount),
    },
    { title: '收货人', dataIndex: 'receiverName', search: false },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: Object.fromEntries(Object.entries(ORDER_STATUS_MAP).map(([k, v]) => [k, { text: v.text }])),
      render: (_, r) => {
        const s = ORDER_STATUS_MAP[r.status] ?? { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
    },
    { title: '下单时间', dataIndex: 'createdAt', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => (
        <a onClick={() => navigate(`/order/${r.orderNo}`)}>详情</a>
      ),
    },
  ]

  return (
    <ProTable<Order>
      headerTitle="订单列表"
      actionRef={actionRef}
      rowKey="orderNo"
      columns={columns}
      request={async (params) => {
        const res = await listOrders({ status: params.status, page: params.current, pageSize: params.pageSize })
        return { data: res?.orders ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
