import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, InfiniteScroll, Toast } from 'antd-mobile'
import { listNotifications, markRead, markAllRead, type Notification } from '@/api/notification'
import styles from './notification.module.css'

export default function NotificationPage() {
  const navigate = useNavigate()
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [hasMore, setHasMore] = useState(true)
  const [page, setPage] = useState(1)

  const loadMore = useCallback(async () => {
    const res = await listNotifications({ page, pageSize: 20 })
    const list = res.notifications || []
    setNotifications((prev) => (page === 1 ? list : [...prev, ...list]))
    setHasMore(list.length >= 20)
    setPage((p) => p + 1)
  }, [page])

  const handleMarkRead = async (n: Notification) => {
    if (n.is_read) return
    try {
      await markRead(n.id)
      setNotifications((prev) =>
        prev.map((item) => (item.id === n.id ? { ...item, is_read: true } : item))
      )
    } catch {
      // ignore
    }
  }

  const handleMarkAllRead = async () => {
    try {
      await markAllRead()
      setNotifications((prev) => prev.map((item) => ({ ...item, is_read: true })))
      Toast.show('已全部标记为已读')
    } catch (e: unknown) {
      Toast.show((e as Error).message || '操作失败')
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>消息通知</NavBar>
      </div>

      {notifications.length > 0 && (
        <div className={styles.actions}>
          <span className={styles.markAllBtn} onClick={handleMarkAllRead}>全部已读</span>
        </div>
      )}

      {notifications.map((n) => (
        <div
          key={n.id}
          className={`${styles.card} ${!n.is_read ? styles.unread : ''}`}
          onClick={() => handleMarkRead(n)}
        >
          <div className={styles.cardTitle}>{n.title}</div>
          <div className={styles.cardContent}>{n.content}</div>
          <div className={styles.cardTime}>{new Date(n.ctime).toLocaleString('zh-CN')}</div>
        </div>
      ))}

      {notifications.length === 0 && !hasMore && (
        <div className={styles.empty}>暂无消息</div>
      )}

      <div className={styles.loadMore}>
        <InfiniteScroll loadMore={loadMore} hasMore={hasMore} />
      </div>
    </div>
  )
}
