import { useRef } from 'react'
import { Tag, Button, Popconfirm, message } from 'antd'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listRefunds, handleRefund } from '@/api/order'
import type { RefundOrder } from '@/types/order'

export default function RefundList() {
  const actionRef = useRef<ActionType>(null)

  const handleAction = async (orderNo: string, refundNo: string, approved: boolean) => {
    try {
      await handleRefund(orderNo, { refund_no: refundNo, approved, reason: approved ? '同意退款' : '拒绝退款' })
      message.success(approved ? '已同意退款' : '已拒绝退款')
      actionRef.current?.reload()
    } catch (e: unknown) {
      message.error((e as Error).message)
    }
  }

  const columns: ProColumns<RefundOrder>[] = [
    { title: '退款单号', dataIndex: 'refund_no', copyable: true },
    { title: '订单号', dataIndex: 'order_no', copyable: true },
    { title: '退款金额', dataIndex: 'amount', search: false, render: (_, r) => `¥${((r.amount ?? 0) / 100).toFixed(2)}` },
    { title: '原因', dataIndex: 'reason', search: false, ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: { 1: { text: '待处理', status: 'Warning' }, 2: { text: '已同意', status: 'Success' }, 3: { text: '已拒绝', status: 'Error' } },
      render: (_, r) => {
        const map: Record<number, { text: string; color: string }> = { 1: { text: '待处理', color: 'orange' }, 2: { text: '已同意', color: 'green' }, 3: { text: '已拒绝', color: 'red' } }
        const s = map[r.status] ?? { text: '未知', color: 'default' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
    },
    { title: '申请时间', dataIndex: 'created_at', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) =>
        r.status === 1 ? (
          <>
            <Popconfirm title="确认同意退款？" onConfirm={() => handleAction(r.order_no, r.refund_no, true)}>
              <Button type="link" size="small">同意</Button>
            </Popconfirm>
            <Popconfirm title="确认拒绝退款？" onConfirm={() => handleAction(r.order_no, r.refund_no, false)}>
              <Button type="link" size="small" danger>拒绝</Button>
            </Popconfirm>
          </>
        ) : '-',
    },
  ]

  return (
    <ProTable<RefundOrder>
      headerTitle="退款管理"
      actionRef={actionRef}
      rowKey="refund_no"
      columns={columns}
      request={async (params) => {
        const res = await listRefunds({ status: params.status, page: params.current, pageSize: params.pageSize })
        return { data: res?.refund_orders ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
