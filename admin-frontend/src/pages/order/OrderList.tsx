import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { Tag } from 'antd'
import { listOrders } from '@/api/order'
import type { Order } from '@/types/order'
import { ORDER_STATUS_MAP, formatPrice } from '@/constants'

export default function OrderList() {
  const actionRef = useRef<ActionType>()
  const navigate = useNavigate()

  const columns: ProColumns<Order>[] = [
    { title: '订单号', dataIndex: 'orderNo', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '金额', dataIndex: 'totalAmount', search: false, render: (_, r) => formatPrice(r.totalAmount) },
    { title: '收货人', dataIndex: 'receiverName', search: false },
    {
      title: '状态',
      dataIndex: 'status',
      render: (_, record) => {
        const s = ORDER_STATUS_MAP[record.status] || { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(ORDER_STATUS_MAP).map(([k, v]) => [k, { text: v.text }])
      ),
    },
    { title: '创建时间', dataIndex: 'createdAt', search: false, valueType: 'dateTime' },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="detail" onClick={() => navigate(`/orders/${record.orderNo}`)}>详情</a>,
      ],
    },
  ]

  return (
    <ProTable<Order>
      headerTitle="订单监管"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listOrders({
          tenantId: params.tenantId,
          status: params.status !== undefined ? Number(params.status) : undefined,
          page: params.current ?? 1,
          pageSize: params.pageSize ?? 20,
        })
        return { data: res?.orders ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
