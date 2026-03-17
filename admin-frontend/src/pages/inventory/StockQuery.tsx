import { useState } from 'react'
import { Card, Input, Button, Descriptions, Space, message } from 'antd'
import { getStock } from '@/api/inventory'
import type { Inventory } from '@/types/inventory'

export default function StockQuery() {
  const [skuId, setSkuId] = useState('')
  const [loading, setLoading] = useState(false)
  const [stock, setStock] = useState<Inventory | null>(null)

  const handleQuery = async () => {
    if (!skuId) { message.warning('请输入 SKU ID'); return }
    setLoading(true)
    try {
      const res = await getStock(Number(skuId))
      setStock(res)
    } catch { /* handled */ }
    setLoading(false)
  }

  return (
    <div>
      <Card title="库存查询">
        <Space>
          <Input placeholder="输入 SKU ID" value={skuId} onChange={(e) => setSkuId(e.target.value)} style={{ width: 200 }} />
          <Button type="primary" loading={loading} onClick={handleQuery}>查询</Button>
        </Space>
      </Card>
      {stock && (
        <Card title="查询结果" style={{ marginTop: 16 }}>
          <Descriptions column={2} bordered>
            <Descriptions.Item label="SKU ID">{stock.skuId}</Descriptions.Item>
            <Descriptions.Item label="租户ID">{stock.tenantId}</Descriptions.Item>
            <Descriptions.Item label="总库存">{stock.stock}</Descriptions.Item>
            <Descriptions.Item label="锁定">{stock.locked}</Descriptions.Item>
            <Descriptions.Item label="可用">{stock.available}</Descriptions.Item>
          </Descriptions>
        </Card>
      )}
    </div>
  )
}
