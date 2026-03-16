import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Descriptions, Table, Tag, Button, Space, message, Modal, Input, Spin } from 'antd'
import { getOrder } from '@/api/order'
import { shipOrder, getOrderLogistics } from '@/api/logistics'
import { ORDER_STATUS_MAP, ORDER_STATUS, formatPrice } from '@/constants'
import { silentApiError } from '@/utils/error'
import type { Order } from '@/types/order'
import type { Shipment, ShipOrderReq } from '@/types/logistics'

export default function OrderDetail() {
  const { orderNo } = useParams<{ orderNo: string }>()
  const navigate = useNavigate()
  const [order, setOrder] = useState<Order | null>(null)
  const [logistics, setLogistics] = useState<Shipment | null>(null)
  const [shipModal, setShipModal] = useState(false)
  const [shipForm, setShipForm] = useState<ShipOrderReq>({ carrierCode: '', carrierName: '', trackingNo: '' })
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (orderNo) {
      getOrder(orderNo).then(setOrder).catch(silentApiError('orderDetail:getOrder'))
      getOrderLogistics(orderNo).then(setLogistics).catch(silentApiError('orderDetail:getLogistics'))
    }
  }, [orderNo])

  const handleShip = async () => {
    if (!orderNo || !shipForm.trackingNo) {
      message.warning('请填写运单号')
      return
    }
    setLoading(true)
    try {
      await shipOrder(orderNo, shipForm)
      message.success('发货成功')
      setShipModal(false)
      getOrder(orderNo).then(setOrder).catch(silentApiError('orderDetail:refreshOrder'))
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
          <Descriptions.Item label="订单号">{order.orderNo}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={ORDER_STATUS_MAP[order.status]?.color}>{ORDER_STATUS_MAP[order.status]?.text ?? '未知'}</Tag></Descriptions.Item>
          <Descriptions.Item label="支付金额">{formatPrice(order.payAmount)}</Descriptions.Item>
          <Descriptions.Item label="运费">{formatPrice(order.freightAmount)}</Descriptions.Item>
          <Descriptions.Item label="收货人">{order.receiverName}</Descriptions.Item>
          <Descriptions.Item label="联系电话">{order.receiverPhone}</Descriptions.Item>
          <Descriptions.Item label="收货地址" span={2}>{order.receiverAddress}</Descriptions.Item>
          <Descriptions.Item label="备注">{order.remark || '-'}</Descriptions.Item>
          <Descriptions.Item label="下单时间">{order.createdAt}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="商品明细" style={{ marginTop: 16 }}>
        <Table
          dataSource={order.items ?? []}
          rowKey="id"
          pagination={false}
          columns={[
            { title: '商品', dataIndex: 'productName' },
            { title: '规格', dataIndex: 'specValues' },
            { title: '单价', dataIndex: 'price', render: (v: number) => formatPrice(v) },
            { title: '数量', dataIndex: 'quantity' },
          ]}
        />
      </Card>

      {logistics && (
        <Card title="物流信息" style={{ marginTop: 16 }}>
          <Descriptions>
            <Descriptions.Item label="物流公司">{logistics.carrierName}</Descriptions.Item>
            <Descriptions.Item label="运单号">{logistics.trackingNo}</Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {order.status === ORDER_STATUS.PAID && (
        <Card style={{ marginTop: 16 }}>
          <Space>
            <Button type="primary" onClick={() => setShipModal(true)}>发货</Button>
          </Space>
        </Card>
      )}

      <Modal title="发货" open={shipModal} onOk={handleShip} confirmLoading={loading} onCancel={() => setShipModal(false)}>
        <Input placeholder="物流公司编码" value={shipForm.carrierCode} onChange={(e) => setShipForm({ ...shipForm, carrierCode: e.target.value })} style={{ marginBottom: 12 }} />
        <Input placeholder="物流公司名称" value={shipForm.carrierName} onChange={(e) => setShipForm({ ...shipForm, carrierName: e.target.value })} style={{ marginBottom: 12 }} />
        <Input placeholder="运单号" value={shipForm.trackingNo} onChange={(e) => setShipForm({ ...shipForm, trackingNo: e.target.value })} />
      </Modal>
    </div>
  )
}
