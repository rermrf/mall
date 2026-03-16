import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Tabs, SwipeAction, Toast, SpinLoading } from 'antd-mobile'
import {
  listNotifications,
  markRead,
  markAllRead,
  deleteNotification,
  type Notification,
} from '@/api/notification'
import styles from './notification.module.css'

const CHANNELS = [
  { key: 'all', title: '全部' },
  { key: '1', title: '系统' },
  { key: '2', title: '订单' },
  { key: '3', title: '营销' },
]

export default function NotificationPage() {
  const navigate = useNavigate()
  const [channel, setChannel] = useState('all')
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [expandedIds, setExpandedIds] = useState<Set<number>>(new Set())
  const [loading, setLoading] = useState(true)

  const loadData = (ch: string) => {
    setLoading(true)
    const params: { page: number; pageSize: number; channel?: number } = {
      page: 1,
      pageSize: 50,
    }
    if (ch !== 'all') params.channel = Number(ch)
    listNotifications(params)
      .then((res) => setNotifications(res?.notifications ?? []))
      .catch(() => Toast.show('加载失败'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    loadData(channel)
  }, [channel])

  const handleMarkRead = async (n: Notification) => {
    if (n.isRead) return
    try {
      await markRead(n.id)
      setNotifications((prev) =>
        prev.map((item) => (item.id === n.id ? { ...item, isRead: true } : item)),
      )
    } catch {
      // ignore
    }
  }

  const handleMarkAllRead = async () => {
    try {
      await markAllRead()
      setNotifications((prev) => prev.map((item) => ({ ...item, isRead: true })))
      Toast.show('已全部标记为已读')
    } catch (e: unknown) {
      Toast.show((e as Error).message || '操作失败')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteNotification(id)
      setNotifications((prev) => prev.filter((n) => n.id !== id))
      Toast.show('已删除')
    } catch {
      Toast.show('删除失败')
    }
  }

  const toggleExpand = (id: number) => {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleItemClick = (n: Notification) => {
    handleMarkRead(n)
    toggleExpand(n.id)
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>消息通知</NavBar>
      </div>

      <div className={styles.tabs}>
        <Tabs activeKey={channel} onChange={setChannel}>
          {CHANNELS.map((ch) => (
            <Tabs.Tab key={ch.key} title={ch.title} />
          ))}
        </Tabs>
      </div>

      {notifications.length > 0 && (
        <div className={styles.actions}>
          <span className={styles.markAllBtn} onClick={handleMarkAllRead}>
            全部已读
          </span>
        </div>
      )}

      {loading ? (
        <div className={styles.loading}>
          <SpinLoading />
        </div>
      ) : notifications.length === 0 ? (
        <div className={styles.empty}>暂无消息</div>
      ) : (
        notifications.map((n) => (
          <SwipeAction
            key={n.id}
            rightActions={[
              {
                key: 'delete',
                text: '删除',
                color: 'danger',
                onClick: () => handleDelete(n.id),
              },
            ]}
          >
            <div
              className={`${styles.card} ${!n.isRead ? styles.unread : ''}`}
              onClick={() => handleItemClick(n)}
            >
              <div className={styles.cardHeader}>
                <span className={styles.cardTitle}>{n.title}</span>
                <span className={styles.channelTag}>
                  {n.channel === 1 ? '系统' : n.channel === 2 ? '订单' : n.channel === 3 ? '营销' : '通知'}
                </span>
              </div>
              <div
                className={
                  expandedIds.has(n.id) ? styles.cardContentExpanded : styles.cardContent
                }
              >
                {n.content}
              </div>
              <div className={styles.cardTime}>
                {new Date(n.ctime).toLocaleString('zh-CN')}
              </div>
            </div>
          </SwipeAction>
        ))
      )}
    </div>
  )
}
