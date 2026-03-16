import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Col, Row, Statistic, List, Button, Typography } from 'antd'
import {
  ShoppingCartOutlined,
  OrderedListOutlined,
  AlertOutlined,
  GiftOutlined,
} from '@ant-design/icons'
import { listOrders, listRefunds } from '@/api/order'
import { silentApiError } from '@/utils/error'

const { Title } = Typography

export default function Dashboard() {
  const navigate = useNavigate()
  const [pendingShip, setPendingShip] = useState(0)
  const [pendingRefund, setPendingRefund] = useState(0)
  const [todayOrders, setTodayOrders] = useState(0)

  useEffect(() => {
    listOrders({ status: 2, page: 1, pageSize: 1 }).then((r) => {
      setPendingShip(r?.total ?? 0)
    }).catch(silentApiError('dashboard:pendingShip'))
    listRefunds({ status: 1, page: 1, pageSize: 1 }).then((r) => {
      setPendingRefund(r?.total ?? 0)
    }).catch(silentApiError('dashboard:pendingRefund'))
    listOrders({ page: 1, pageSize: 1 }).then((r) => {
      setTodayOrders(r?.total ?? 0)
    }).catch(silentApiError('dashboard:todayOrders'))
  }, [])

  const shortcuts = [
    { title: '发布商品', path: '/product/create' },
    { title: '处理订单', path: '/order/list' },
    { title: '管理库存', path: '/inventory' },
    { title: '创建优惠券', path: '/marketing/coupon/create' },
    { title: '店铺设置', path: '/shop/settings' },
  ]

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>概览</Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/order/list')}>
            <Statistic title="总订单数" value={todayOrders} prefix={<ShoppingCartOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/order/list')}>
            <Statistic title="待发货" value={pendingShip} prefix={<OrderedListOutlined />} valueStyle={pendingShip > 0 ? { color: '#faad14' } : undefined} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/order/refund')}>
            <Statistic title="待处理退款" value={pendingRefund} prefix={<AlertOutlined />} valueStyle={pendingRefund > 0 ? { color: '#ff4d4f' } : undefined} />
          </Card>
        </Col>
      </Row>

      <Card title="快捷入口" style={{ marginTop: 24 }}>
        <List
          grid={{ gutter: 16, xs: 2, sm: 3, md: 5 }}
          dataSource={shortcuts}
          renderItem={(item) => (
            <List.Item>
              <Button block onClick={() => navigate(item.path)} icon={<GiftOutlined />}>
                {item.title}
              </Button>
            </List.Item>
          )}
        />
      </Card>
    </div>
  )
}
