import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Col, Row, Statistic, List, Button, Typography } from 'antd'
import {
  SafetyCertificateOutlined,
  TeamOutlined,
  OrderedListOutlined,
  AppstoreOutlined,
} from '@ant-design/icons'
import { listTenants } from '@/api/tenant'
import { listUsers } from '@/api/user'
import { listOrders } from '@/api/order'
import { silentApiError } from '@/utils/error'

const { Title } = Typography

export default function Dashboard() {
  const navigate = useNavigate()
  const [tenantCount, setTenantCount] = useState(0)
  const [userCount, setUserCount] = useState(0)
  const [orderCount, setOrderCount] = useState(0)

  useEffect(() => {
    listTenants({ page: 1, pageSize: 1 }).then((r) => setTenantCount(r?.total ?? 0)).catch(silentApiError('dashboard:tenants'))
    listUsers({ page: 1, pageSize: 1 }).then((r) => setUserCount(r?.total ?? 0)).catch(silentApiError('dashboard:users'))
    listOrders({ page: 1, pageSize: 1 }).then((r) => setOrderCount(r?.total ?? 0)).catch(silentApiError('dashboard:orders'))
  }, [])

  const shortcuts = [
    { title: '租户管理', path: '/tenants' },
    { title: '用户管理', path: '/users' },
    { title: '订单监管', path: '/orders' },
    { title: '分类管理', path: '/categories' },
    { title: '通知管理', path: '/notification-templates' },
  ]

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>平台概览</Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/tenants')}>
            <Statistic title="租户总数" value={tenantCount} prefix={<SafetyCertificateOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/users')}>
            <Statistic title="用户总数" value={userCount} prefix={<TeamOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card hoverable onClick={() => navigate('/orders')}>
            <Statistic title="订单总数" value={orderCount} prefix={<OrderedListOutlined />} />
          </Card>
        </Col>
      </Row>

      <Card title="快捷入口" style={{ marginTop: 24 }}>
        <List
          grid={{ gutter: 16, xs: 2, sm: 3, md: 5 }}
          dataSource={shortcuts}
          renderItem={(item) => (
            <List.Item>
              <Button block onClick={() => navigate(item.path)} icon={<AppstoreOutlined />}>
                {item.title}
              </Button>
            </List.Item>
          )}
        />
      </Card>
    </div>
  )
}
