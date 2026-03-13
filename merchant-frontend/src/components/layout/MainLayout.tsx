import { useState, useEffect } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { ProLayout } from '@ant-design/pro-components'
import { Badge, Dropdown, Avatar, message } from 'antd'
import {
  DashboardOutlined,
  ShoppingOutlined,
  OrderedListOutlined,
  InboxOutlined,
  GiftOutlined,
  CarOutlined,
  ShopOutlined,
  TeamOutlined,
  BellOutlined,
  LogoutOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '@/stores/auth'
import { useNotificationStore } from '@/stores/notification'
import { logout } from '@/api/auth'
import { getProfile } from '@/api/staff'

const menuRoutes = {
  path: '/',
  routes: [
    { path: '/', name: '仪表盘', icon: <DashboardOutlined /> },
    {
      path: '/product',
      name: '商品管理',
      icon: <ShoppingOutlined />,
      routes: [
        { path: '/product/list', name: '商品列表' },
        { path: '/product/category', name: '分类管理' },
        { path: '/product/brand', name: '品牌管理' },
      ],
    },
    {
      path: '/order',
      name: '订单管理',
      icon: <OrderedListOutlined />,
      routes: [
        { path: '/order/list', name: '订单列表' },
        { path: '/order/refund', name: '退款管理' },
      ],
    },
    {
      path: '/inventory',
      name: '库存管理',
      icon: <InboxOutlined />,
      routes: [
        { path: '/inventory', name: '库存查看' },
        { path: '/inventory/log', name: '变更日志' },
      ],
    },
    {
      path: '/marketing',
      name: '营销管理',
      icon: <GiftOutlined />,
      routes: [
        { path: '/marketing/coupon', name: '优惠券' },
        { path: '/marketing/seckill', name: '秒杀活动' },
        { path: '/marketing/promotion', name: '促销规则' },
      ],
    },
    {
      path: '/logistics',
      name: '物流管理',
      icon: <CarOutlined />,
      routes: [
        { path: '/logistics/template', name: '运费模板' },
      ],
    },
    { path: '/shop/settings', name: '店铺设置', icon: <ShopOutlined /> },
    {
      path: '/staff',
      name: '团队管理',
      icon: <TeamOutlined />,
      routes: [
        { path: '/staff/list', name: '员工列表' },
        { path: '/staff/role', name: '角色管理' },
      ],
    },
    { path: '/notification', name: '消息中心', icon: <BellOutlined /> },
  ],
}

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const clearAuth = useAuthStore((s) => s.clearAuth)
  const { unreadCount, fetchUnreadCount } = useNotificationStore()
  const [nickname, setNickname] = useState('商家')

  useEffect(() => {
    fetchUnreadCount()
    getProfile().then((u) => { if (u?.nickname) setNickname(u.nickname) }).catch(() => {})
  }, [fetchUnreadCount])

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
      title="商家管理后台"
      logo={<ShopOutlined style={{ fontSize: 28, color: '#1890ff' }} />}
      route={menuRoutes}
      location={{ pathname: location.pathname }}
      menuItemRender={(item, dom) => (
        <span onClick={() => item.path && navigate(item.path)}>{dom}</span>
      )}
      actionsRender={() => [
        <Badge key="bell" count={unreadCount} size="small" offset={[-2, 2]}>
          <BellOutlined style={{ fontSize: 18, cursor: 'pointer' }} onClick={() => navigate('/notification')} />
        </Badge>,
        <Dropdown
          key="user"
          menu={{
            items: [
              { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
            ],
          }}
        >
          <span style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 8 }}>
            <Avatar size="small" icon={<UserOutlined />} />
            <span>{nickname}</span>
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
