import { useRef } from 'react'
import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
import { listLogs } from '@/api/inventory'
import type { InventoryLog } from '@/types/inventory'

export default function StockLog() {
  const actionRef = useRef<ActionType>()

  const columns: ProColumns<InventoryLog>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: 'SKU ID', dataIndex: 'skuId', valueType: 'digit', fieldProps: { placeholder: '按SKU筛选' } },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: false,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    { title: '变动量', dataIndex: 'change', search: false },
    { title: '类型', dataIndex: 'type', search: false },
    { title: '关联订单', dataIndex: 'orderNo', search: false },
    { title: '时间', dataIndex: 'createdAt', search: false, valueType: 'dateTime' },
  ]

  return (
    <ProTable<InventoryLog>
      headerTitle="库存日志"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listLogs({
          tenantId: params.tenantId,
          skuId: params.skuId,
          page: params.current ?? 1,
          pageSize: params.pageSize ?? 20,
        })
        return { data: res?.logs ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
      search={{ labelWidth: 'auto' }}
    />
  )
}
