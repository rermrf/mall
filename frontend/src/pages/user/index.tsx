import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Dialog, Toast, Button } from 'antd-mobile'
import { useAuthStore } from '@/stores/auth'
import { logout } from '@/api/auth'
import { getProfile, type UserProfile } from '@/api/user'
import { getUnreadCount } from '@/api/notification'
import styles from './user.module.css'

export default function UserPage() {
  const navigate = useNavigate()
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const clearAuth = useAuthStore((s) => s.clearAuth)
  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [unread, setUnread] = useState(0)

  useEffect(() => {
    if (isLoggedIn) {
      getProfile().then(setProfile).catch(() => {})
      getUnreadCount().then(setUnread).catch(() => {})
    }
  }, [isLoggedIn])

  const handleLogout = async () => {
    const confirmed = await Dialog.confirm({ content: '确定退出登录？' })
    if (confirmed) {
      try {
        await logout()
      } catch {
        // ignore network errors during logout
      }
      navigate('/', { replace: true })
      clearAuth()
      setProfile(null)
      Toast.show('已退出登录')
    }
  }

  const menus: Array<{ label: string; path: string; badge?: number }> = [
    { label: '我的订单', path: '/orders' },
    { label: '我的退款', path: '/refunds' },
    { label: '消息通知', path: '/notifications', badge: unread },
    { label: '优惠券', path: '/coupons' },
    { label: '收货地址', path: '/me/addresses' },
    { label: '编辑资料', path: '/me/profile' },
  ]

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        {isLoggedIn && profile ? (
          <>
            {profile.avatar ? (
              <img className={styles.avatar} src={profile.avatar} alt="" />
            ) : (
              <div className={styles.avatarPlaceholder}>👤</div>
            )}
            <div className={styles.userInfo}>
              <div className={styles.nickname}>{profile.nickname || '用户'}</div>
              <div className={styles.phone}>{profile.phone}</div>
            </div>
          </>
        ) : (
          <>
            <div className={styles.avatarPlaceholder}>👤</div>
            <div className={styles.userInfo}>
              <div className={styles.loginBtn} onClick={() => navigate('/login')}>
                登录 / 注册
              </div>
            </div>
          </>
        )}
      </div>

      <div className={styles.section}>
        {menus.map((item) => (
          <div key={item.path} className={styles.menuItem} onClick={() => navigate(item.path)}>
            <span className={styles.menuLabel}>{item.label}</span>
            {item.badge && item.badge > 0 ? (
              <span className={styles.menuBadge}>{item.badge > 99 ? '99+' : item.badge}</span>
            ) : null}
            <span className={styles.menuArrow}>›</span>
          </div>
        ))}
      </div>

      {isLoggedIn && (
        <div className={styles.logoutSection}>
          <Button className={styles.logoutBtn} onClick={handleLogout}>
            退出登录
          </Button>
        </div>
      )}
    </div>
  )
}
