import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Tag } from 'antd'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listPayments } from '@/api/payment'
import { PAYMENT_STATUS_MAP, formatPrice } from '@/constants'
import type { Payment } from '@/types/payment'

export default function PaymentList() {
  const navigate = useNavigate()
  const actionRef = useRef<ActionType>(null)

  const columns: ProColumns<Payment>[] = [
    { title: '支付单号', dataIndex: 'payment_no', copyable: true },
    { title: '订单号', dataIndex: 'order_no', copyable: true },
    {
      title: '金额',
      dataIndex: 'amount',
      search: false,
      render: (_, r) => formatPrice(r.amount),
    },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: Object.fromEntries(Object.entries(PAYMENT_STATUS_MAP).map(([k, v]) => [k, { text: v.text }])),
      render: (_, r) => {
        const s = PAYMENT_STATUS_MAP[r.status] ?? { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
    },
    { title: '支付渠道', dataIndex: 'channel', search: false },
    { title: '创建时间', dataIndex: 'created_at', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => (
        <a onClick={() => navigate(`/payment/${r.payment_no}`)}>详情</a>
      ),
    },
  ]

  return (
    <ProTable<Payment>
      headerTitle="支付列表"
      actionRef={actionRef}
      rowKey="payment_no"
      columns={columns}
      request={async (params) => {
        const res = await listPayments({ status: params.status, page: params.current, pageSize: params.pageSize })
        return { data: res?.payments ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
