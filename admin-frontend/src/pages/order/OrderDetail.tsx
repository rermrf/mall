import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Descriptions, Card, Spin, Tag, Table, Divider } from 'antd'
import { getOrder } from '@/api/order'
import { getOrderLogistics } from '@/api/logistics'
import type { Order } from '@/types/order'
import type { Shipment } from '@/types/logistics'
import { ORDER_STATUS_MAP, formatPrice } from '@/constants'
import { silentApiError } from '@/utils/error'

export default function OrderDetail() {
  const { orderNo } = useParams<{ orderNo: string }>()
  const [order, setOrder] = useState<Order | null>(null)
  const [shipment, setShipment] = useState<Shipment | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!orderNo) return
    getOrder(orderNo).then(setOrder).catch(() => {}).finally(() => setLoading(false))
    getOrderLogistics(orderNo).then(setShipment).catch(silentApiError('orderDetail:logistics'))
  }, [orderNo])

  if (loading) return <Spin style={{ display: 'block', margin: '20vh auto' }} size="large" />
  if (!order) return <div>订单不存在</div>

  const s = ORDER_STATUS_MAP[order.status] || { text: '未知', color: 'default' }

  return (
    <div>
      <Card title="订单详情">
        <Descriptions column={2} bordered>
          <Descriptions.Item label="订单号">{order.orderNo}</Descriptions.Item>
          <Descriptions.Item label="租户ID">{order.tenantId}</Descriptions.Item>
          <Descriptions.Item label="总金额">{formatPrice(order.totalAmount)}</Descriptions.Item>
          <Descriptions.Item label="实付金额">{formatPrice(order.payAmount)}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={s.color}>{s.text}</Tag></Descriptions.Item>
          <Descriptions.Item label="收货人">{order.receiverName}</Descriptions.Item>
          <Descriptions.Item label="收货电话">{order.receiverPhone}</Descriptions.Item>
          <Descriptions.Item label="收货地址">{order.receiverAddress}</Descriptions.Item>
          <Descriptions.Item label="创建时间">{order.createdAt}</Descriptions.Item>
        </Descriptions>
      </Card>

      {order.items && order.items.length > 0 && (
        <>
          <Divider />
          <Card title="商品明细">
            <Table
              dataSource={order.items}
              rowKey="id"
              pagination={false}
              columns={[
                { title: '商品', dataIndex: 'title' },
                { title: '单价', dataIndex: 'price', render: (v: number) => formatPrice(v) },
                { title: '数量', dataIndex: 'quantity' },
                { title: '小计', render: (_, r) => formatPrice(r.price * r.quantity) },
              ]}
            />
          </Card>
        </>
      )}

      {shipment && (
        <>
          <Divider />
          <Card title="物流信息">
            <Descriptions column={2} bordered>
              <Descriptions.Item label="物流公司">{shipment.company}</Descriptions.Item>
              <Descriptions.Item label="运单号">{shipment.trackingNo}</Descriptions.Item>
              <Descriptions.Item label="状态">{shipment.status}</Descriptions.Item>
              <Descriptions.Item label="创建时间">{shipment.createdAt}</Descriptions.Item>
            </Descriptions>
          </Card>
        </>
      )}
    </div>
  )
}
