import { useRef } from 'react'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { listCoupons } from '@/api/marketing'
import type { Coupon } from '@/types/marketing'
import { formatPrice, COUPON_TYPE_OPTIONS } from '@/constants'

export default function CouponList() {
  const actionRef = useRef<ActionType>()

  const columns: ProColumns<Coupon>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '名称', dataIndex: 'name', search: false },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '类型', dataIndex: 'type', search: false, render: (_, r) => COUPON_TYPE_OPTIONS.find((o) => o.value === r.type)?.label || r.type },
    { title: '面额', dataIndex: 'value', search: false, render: (_, r) => formatPrice(r.value) },
    { title: '门槛', dataIndex: 'minAmount', search: false, render: (_, r) => formatPrice(r.minAmount) },
    { title: '已用/总量', search: false, render: (_, r) => `${r.usedCount}/${r.totalCount}` },
    { title: '开始时间', dataIndex: 'startTime', search: false, valueType: 'dateTime' },
    { title: '结束时间', dataIndex: 'endTime', search: false, valueType: 'dateTime' },
  ]

  return (
    <ProTable<Coupon>
      headerTitle="优惠券监管"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listCoupons({
          tenantId: params.tenantId,
          page: params.current ?? 1,
          pageSize: params.pageSize ?? 20,
        })
        return { data: res?.coupons ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
