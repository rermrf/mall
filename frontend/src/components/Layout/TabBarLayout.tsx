import { useState, useEffect } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { TabBar, Badge } from 'antd-mobile'
import {
  AppOutline,
  SearchOutline,
  ShopbagOutline,
  UserOutline,
} from 'antd-mobile-icons'
import { useAuthStore } from '@/stores/auth'
import { getUnreadCount } from '@/api/notification'
import styles from './TabBarLayout.module.css'

export default function TabBarLayout() {
  const location = useLocation()
  const navigate = useNavigate()
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const [unread, setUnread] = useState(0)

  useEffect(() => {
    if (isLoggedIn) {
      getUnreadCount().then(setUnread).catch(() => {})
    } else {
      setUnread(0)
    }
  }, [isLoggedIn, location.pathname])

  const tabs = [
    { key: '/', title: '首页', icon: <AppOutline /> },
    { key: '/search', title: '搜索', icon: <SearchOutline /> },
    { key: '/cart', title: '购物车', icon: <ShopbagOutline /> },
    {
      key: '/me',
      title: '我的',
      icon: unread > 0 ? <Badge content={Badge.dot}><UserOutline /></Badge> : <UserOutline />,
    },
  ]

  return (
    <div className={styles.container}>
      <div className={styles.content}>
        <Outlet />
      </div>
      <div className={styles.tabBar}>
        <TabBar
          activeKey={location.pathname}
          onChange={(key) => navigate(key)}
        >
          {tabs.map((tab) => (
            <TabBar.Item key={tab.key} icon={tab.icon} title={tab.title} />
          ))}
        </TabBar>
      </div>
    </div>
  )
}
