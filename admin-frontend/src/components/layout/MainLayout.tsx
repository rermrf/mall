import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { ProLayout } from '@ant-design/pro-components'
import { Dropdown, Avatar, message } from 'antd'
import {
  DashboardOutlined,
  TeamOutlined,
  ShopOutlined,
  AppstoreOutlined,
  TagsOutlined,
  OrderedListOutlined,
  BellOutlined,
  InboxOutlined,
  GiftOutlined,
  CarOutlined,
  LogoutOutlined,
  UserOutlined,
  CrownOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '@/stores/auth'
import { logout } from '@/api/auth'

const menuRoutes = {
  path: '/',
  routes: [
    { path: '/dashboard', name: '仪表盘', icon: <DashboardOutlined /> },
    {
      path: '/user-mgmt',
      name: '用户管理',
      icon: <TeamOutlined />,
      routes: [
        { path: '/users', name: '用户列表' },
        { path: '/roles', name: '角色管理' },
      ],
    },
    {
      path: '/tenant-mgmt',
      name: '租户管理',
      icon: <SafetyCertificateOutlined />,
      routes: [
        { path: '/tenants', name: '租户列表' },
        { path: '/plans', name: '套餐管理' },
      ],
    },
    {
      path: '/product-mgmt',
      name: '商品管理',
      icon: <AppstoreOutlined />,
      routes: [
        { path: '/categories', name: '分类管理' },
        { path: '/brands', name: '品牌管理' },
      ],
    },
    {
      path: '/order-supervision',
      name: '订单监管',
      icon: <OrderedListOutlined />,
      routes: [
        { path: '/orders', name: '订单列表' },
      ],
    },
    {
      path: '/notification-mgmt',
      name: '通知管理',
      icon: <BellOutlined />,
      routes: [
        { path: '/notification-templates', name: '模板列表' },
        { path: '/notifications/send', name: '发送通知' },
      ],
    },
    {
      path: '/inventory-supervision',
      name: '库存监管',
      icon: <InboxOutlined />,
      routes: [
        { path: '/inventory', name: '库存查询' },
        { path: '/inventory/logs', name: '库存日志' },
      ],
    },
    {
      path: '/marketing-supervision',
      name: '营销监管',
      icon: <GiftOutlined />,
      routes: [
        { path: '/coupons', name: '优惠券' },
        { path: '/seckill', name: '秒杀活动' },
        { path: '/promotions', name: '促销规则' },
      ],
    },
    {
      path: '/logistics-supervision',
      name: '物流监管',
      icon: <CarOutlined />,
      routes: [
        { path: '/freight-templates', name: '运费模板' },
      ],
    },
  ],
}

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const clearAuth = useAuthStore((s) => s.clearAuth)

  const handleLogout = async () => {
    try {
      await logout()
    } catch {
      // ignore
    }
    clearAuth()
    navigate('/login', { replace: true })
    message.success('已退出登录')
  }

  return (
    <ProLayout
      title="平台管理后台"
      logo={<CrownOutlined style={{ fontSize: 28, color: '#722ed1' }} />}
      route={menuRoutes}
      location={{ pathname: location.pathname }}
      menuItemRender={(item, dom) => (
        <span onClick={() => item.path && navigate(item.path)}>{dom}</span>
      )}
      actionsRender={() => [
        <Dropdown
          key="user"
          menu={{
            items: [
              { type: 'divider' as const },
              { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
            ],
          }}
        >
          <span style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 8 }}>
            <Avatar size="small" icon={<UserOutlined />} />
            <span>管理员</span>
          </span>
        </Dropdown>,
      ]}
      fixSiderbar
      layout="mix"
    >
      <Outlet />
    </ProLayout>
  )
}
