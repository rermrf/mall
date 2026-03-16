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
    { title: '订单号', dataIndex: 'order_no', copyable: true },
    {
      title: '金额',
      dataIndex: 'pay_amount',
      search: false,
      render: (_, r) => formatPrice(r.pay_amount),
    },
    { title: '收货人', dataIndex: 'receiver_name', search: false },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: Object.fromEntries(Object.entries(ORDER_STATUS_MAP).map(([k, v]) => [k, { text: v.text }])),
      render: (_, r) => {
        const s = ORDER_STATUS_MAP[r.status] ?? { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
    },
    { title: '下单时间', dataIndex: 'created_at', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => (
        <a onClick={() => navigate(`/order/${r.order_no}`)}>详情</a>
      ),
    },
  ]

  return (
    <ProTable<Order>
      headerTitle="订单列表"
      actionRef={actionRef}
      rowKey="order_no"
      columns={columns}
      request={async (params) => {
        const res = await listOrders({ status: params.status, page: params.current, pageSize: params.pageSize })
        return { data: res?.orders ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
