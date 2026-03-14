import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Descriptions, Table, Tag, Button, Space, message, Modal, Input, Spin } from 'antd'
import { getOrder } from '@/api/order'
import { shipOrder, getOrderLogistics } from '@/api/logistics'
import type { Order } from '@/types/order'
import type { Shipment, ShipOrderReq } from '@/types/logistics'

export default function OrderDetail() {
  const { orderNo } = useParams<{ orderNo: string }>()
  const navigate = useNavigate()
  const [order, setOrder] = useState<Order | null>(null)
  const [logistics, setLogistics] = useState<Shipment | null>(null)
  const [shipModal, setShipModal] = useState(false)
  const [shipForm, setShipForm] = useState<ShipOrderReq>({ carrier_code: '', carrier_name: '', tracking_no: '' })
  const [loading, setLoading] = useState(false)

  const statusMap: Record<number, { text: string; color: string }> = {
    0: { text: '已取消', color: 'default' },
    1: { text: '待付款', color: 'orange' },
    2: { text: '待发货', color: 'blue' },
    3: { text: '已发货', color: 'cyan' },
    4: { text: '已完成', color: 'green' },
    5: { text: '退款中', color: 'red' },
  }

  useEffect(() => {
    if (orderNo) {
      getOrder(orderNo).then(setOrder).catch(() => {})
      getOrderLogistics(orderNo).then(setLogistics).catch(() => {})
    }
  }, [orderNo])

  const handleShip = async () => {
    if (!orderNo || !shipForm.tracking_no) {
      message.warning('请填写运单号')
      return
    }
    setLoading(true)
    try {
      await shipOrder(orderNo, shipForm)
      message.success('发货成功')
      setShipModal(false)
      getOrder(orderNo).then(setOrder).catch(() => {})
    } catch (e: unknown) {
      message.error((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  if (!order) return <Spin spinning style={{ display: 'flex', justifyContent: 'center', padding: 100 }} />

  return (
    <div>
      <Card title="订单信息" extra={<Button onClick={() => navigate(-1)}>返回</Button>}>
        <Descriptions column={2}>
          <Descriptions.Item label="订单号">{order.order_no}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={statusMap[order.status]?.color}>{statusMap[order.status]?.text ?? '未知'}</Tag></Descriptions.Item>
          <Descriptions.Item label="支付金额">¥{((order.pay_amount ?? 0) / 100).toFixed(2)}</Descriptions.Item>
          <Descriptions.Item label="运费">¥{((order.freight_amount ?? 0) / 100).toFixed(2)}</Descriptions.Item>
          <Descriptions.Item label="收货人">{order.receiver_name}</Descriptions.Item>
          <Descriptions.Item label="联系电话">{order.receiver_phone}</Descriptions.Item>
          <Descriptions.Item label="收货地址" span={2}>{order.receiver_address}</Descriptions.Item>
          <Descriptions.Item label="备注">{order.remark || '-'}</Descriptions.Item>
          <Descriptions.Item label="下单时间">{order.created_at}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="商品明细" style={{ marginTop: 16 }}>
        <Table
          dataSource={order.items ?? []}
          rowKey="id"
          pagination={false}
          columns={[
            { title: '商品', dataIndex: 'product_name' },
            { title: '规格', dataIndex: 'spec_values' },
            { title: '单价', dataIndex: 'price', render: (v: number) => `¥${((v ?? 0) / 100).toFixed(2)}` },
            { title: '数量', dataIndex: 'quantity' },
          ]}
        />
      </Card>

      {logistics && (
        <Card title="物流信息" style={{ marginTop: 16 }}>
          <Descriptions>
            <Descriptions.Item label="物流公司">{logistics.carrier_name}</Descriptions.Item>
            <Descriptions.Item label="运单号">{logistics.tracking_no}</Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {order.status === 2 && (
        <Card style={{ marginTop: 16 }}>
          <Space>
            <Button type="primary" onClick={() => setShipModal(true)}>发货</Button>
          </Space>
        </Card>
      )}

      <Modal title="发货" open={shipModal} onOk={handleShip} confirmLoading={loading} onCancel={() => setShipModal(false)}>
        <Input placeholder="物流公司编码" value={shipForm.carrier_code} onChange={(e) => setShipForm({ ...shipForm, carrier_code: e.target.value })} style={{ marginBottom: 12 }} />
        <Input placeholder="物流公司名称" value={shipForm.carrier_name} onChange={(e) => setShipForm({ ...shipForm, carrier_name: e.target.value })} style={{ marginBottom: 12 }} />
        <Input placeholder="运单号" value={shipForm.tracking_no} onChange={(e) => setShipForm({ ...shipForm, tracking_no: e.target.value })} />
      </Modal>
    </div>
  )
}
