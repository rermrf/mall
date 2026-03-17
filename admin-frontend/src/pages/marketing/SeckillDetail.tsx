import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Descriptions, Card, Spin, Table, Divider, Tag } from 'antd'
import { getSeckill } from '@/api/marketing'
import type { SeckillActivity } from '@/types/marketing'
import { formatPrice } from '@/constants'

export default function SeckillDetail() {
  const { id } = useParams<{ id: string }>()
  const [activity, setActivity] = useState<SeckillActivity | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    getSeckill(Number(id)).then(setActivity).catch(() => {}).finally(() => setLoading(false))
  }, [id])

  if (loading) return <Spin style={{ display: 'block', margin: '20vh auto' }} size="large" />
  if (!activity) return <div>活动不存在</div>

  const statusMap: Record<number, { text: string; color: string }> = {
    0: { text: '未开始', color: 'default' },
    1: { text: '进行中', color: 'green' },
    2: { text: '已结束', color: 'red' },
  }
  const s = statusMap[activity.status] || { text: '未知', color: 'default' }

  return (
    <div>
      <Card title="秒杀活动详情">
        <Descriptions column={2} bordered>
          <Descriptions.Item label="ID">{activity.id}</Descriptions.Item>
          <Descriptions.Item label="标题">{activity.title}</Descriptions.Item>
          <Descriptions.Item label="租户ID">{activity.tenantId}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={s.color}>{s.text}</Tag></Descriptions.Item>
          <Descriptions.Item label="开始时间">{activity.startTime}</Descriptions.Item>
          <Descriptions.Item label="结束时间">{activity.endTime}</Descriptions.Item>
        </Descriptions>
      </Card>

      {activity.items && activity.items.length > 0 && (
        <>
          <Divider />
          <Card title="秒杀商品">
            <Table
              dataSource={activity.items}
              rowKey="id"
              pagination={false}
              columns={[
                { title: '商品ID', dataIndex: 'productId' },
                { title: 'SKU ID', dataIndex: 'skuId' },
                { title: '秒杀价', dataIndex: 'seckillPrice', render: (v: number) => formatPrice(v) },
                { title: '库存', dataIndex: 'stock' },
                { title: '限购', dataIndex: 'limit' },
              ]}
            />
          </Card>
        </>
      )}
    </div>
  )
}
