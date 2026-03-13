import { ProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
import { listInventoryLogs } from '@/api/inventory'
import type { InventoryLog } from '@/types/inventory'

export default function StockLog() {
  const columns: ProColumns<InventoryLog>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: 'SKU ID', dataIndex: 'sku_id' },
    { title: '变更类型', dataIndex: 'change_type' },
    { title: '变更数量', dataIndex: 'change_amount', search: false },
    { title: '变更前', dataIndex: 'before_total', search: false },
    { title: '变更后', dataIndex: 'after_total', search: false },
    { title: '关联订单', dataIndex: 'order_no', search: false },
    { title: '时间', dataIndex: 'created_at', valueType: 'dateTime', search: false },
  ]

  return (
    <ProTable<InventoryLog>
      headerTitle="库存变更日志"
      rowKey="id"
      columns={columns}
      request={async (params) => {
        const res = await listInventoryLogs({ sku_id: params.sku_id, page: params.current, pageSize: params.pageSize })
        return { data: res?.logs ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
