import { ProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
import { listInventoryLogs } from '@/api/inventory'
import type { InventoryLog } from '@/types/inventory'

export default function StockLog() {
  const columns: ProColumns<InventoryLog>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: 'SKU ID', dataIndex: 'skuId' },
    { title: '变更类型', dataIndex: 'changeType' },
    { title: '变更数量', dataIndex: 'changeAmount', search: false },
    { title: '变更前', dataIndex: 'beforeTotal', search: false },
    { title: '变更后', dataIndex: 'afterTotal', search: false },
    { title: '关联订单', dataIndex: 'orderNo', search: false },
    { title: '时间', dataIndex: 'createdAt', valueType: 'dateTime', search: false },
  ]

  return (
    <ProTable<InventoryLog>
      headerTitle="库存变更日志"
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listInventoryLogs({ skuId: params.skuId, page: params.current, pageSize: params.pageSize })
        return { data: res?.logs ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
