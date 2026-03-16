import { useRef, useState } from 'react'
import { message } from 'antd'
import { ProTable, ModalForm, ProFormDigit } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listProducts } from '@/api/product'
import { batchGetStock, setStock } from '@/api/inventory'
import type { Product } from '@/types/product'
import type { Inventory } from '@/types/inventory'

interface StockRow {
  skuId: number
  skuCode: string
  productName: string
  specValues: string
  total: number
  locked: number
  available: number
  alertThreshold: number
}

export default function StockList() {
  const actionRef = useRef<ActionType>(null)
  const [editSku, setEditSku] = useState<StockRow | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const columns: ProColumns<StockRow>[] = [
    { title: 'SKU ID', dataIndex: 'skuId', width: 80 },
    { title: '商品', dataIndex: 'productName', ellipsis: true },
    { title: 'SKU编码', dataIndex: 'skuCode' },
    { title: '规格', dataIndex: 'specValues' },
    { title: '总库存', dataIndex: 'total', search: false },
    { title: '锁定', dataIndex: 'locked', search: false },
    { title: '可用', dataIndex: 'available', search: false },
    { title: '预警值', dataIndex: 'alertThreshold', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => (
        <a onClick={() => { setEditSku(r); setModalOpen(true) }}>设置库存</a>
      ),
    },
  ]

  return (
    <>
      <ProTable<StockRow>
        headerTitle="库存管理"
        actionRef={actionRef}
        rowKey="skuId"
        columns={columns}
        search={false}
        request={async (params) => {
          const productRes = await listProducts({ page: params.current, pageSize: params.pageSize })
          const products = productRes?.products ?? []
          const allSkus = products.flatMap((p: Product) =>
            (p.skus ?? []).map((s) => ({ ...s, productName: p.name }))
          )
          if (allSkus.length === 0) return { data: [], total: 0, success: true }
          const stocks = await batchGetStock(allSkus.map((s) => s.id)).catch(() => [] as Inventory[])
          const stockMap = new Map((stocks ?? []).map((s) => [s.skuId, s]))
          const rows: StockRow[] = allSkus.map((s) => {
            const inv = stockMap.get(s.id)
            return {
              skuId: s.id,
              skuCode: s.skuCode,
              productName: s.productName,
              specValues: s.specValues,
              total: inv?.total ?? 0,
              locked: inv?.locked ?? 0,
              available: inv?.available ?? 0,
              alertThreshold: inv?.alertThreshold ?? 0,
            }
          })
          return { data: rows, total: productRes?.total ?? 0, success: true }
        }}
        pagination={{ defaultPageSize: 20 }}
      />
      <ModalForm
        title="设置库存"
        open={modalOpen}
        initialValues={editSku ? { total: editSku.total, alertThreshold: editSku.alertThreshold } : {}}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editSku) {
            await setStock({ skuId: editSku.skuId, total: values.total, alertThreshold: values.alertThreshold })
            message.success('设置成功')
            actionRef.current?.reload()
          }
          return true
        }}
      >
        <ProFormDigit name="total" label="总库存" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="alertThreshold" label="预警阈值" min={0} />
      </ModalForm>
    </>
  )
}
