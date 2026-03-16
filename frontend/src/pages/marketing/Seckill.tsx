import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Button, Toast } from 'antd-mobile'
import { listSeckillActivities, seckill, type SeckillActivity } from '@/api/marketing'
import { useAuthStore } from '@/stores/auth'
import styles from './seckill.module.css'

function useCountdown(endTime: string): string {
  const [text, setText] = useState('')
  const intervalRef = useRef<ReturnType<typeof setInterval>>(undefined)

  const compute = useCallback(() => {
    const diff = new Date(endTime).getTime() - Date.now()
    if (diff <= 0) {
      setText('已结束')
      if (intervalRef.current) clearInterval(intervalRef.current)
      return
    }
    const h = Math.floor(diff / 3600000)
    const m = Math.floor((diff % 3600000) / 60000)
    const s = Math.floor((diff % 60000) / 1000)
    setText(`${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`)
  }, [endTime])

  useEffect(() => {
    compute()
    intervalRef.current = setInterval(compute, 1000)
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [compute])

  return text
}

function ActivityCard({ activity }: { activity: SeckillActivity }) {
  const navigate = useNavigate()
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const countdown = useCountdown(activity.endTime)
  const isStarted = new Date(activity.startTime).getTime() <= Date.now()
  const isEnded = countdown === '已结束'

  const handleBuy = async (itemId: number) => {
    if (!isLoggedIn) {
      navigate('/login')
      return
    }
    try {
      const res = await seckill(itemId)
      if (res.orderNo) {
        Toast.show('抢购成功！')
        navigate(`/payment/${res.orderNo}`)
      } else {
        Toast.show(res.message || '抢购成功')
      }
    } catch (e: unknown) {
      Toast.show((e as Error).message || '抢购失败')
    }
  }

  return (
    <div className={styles.activityCard}>
      <div className={styles.activityHeader}>
        <span className={styles.activityName}>{activity.name}</span>
        <span className={styles.countdown}>
          {isEnded ? (
            '已结束'
          ) : !isStarted ? (
            '即将开始'
          ) : (
            <>
              <span className={styles.countdownLabel}>距结束</span>
              <span className={styles.countdownTime}>{countdown}</span>
            </>
          )}
        </span>
      </div>
      <div className={styles.itemsList}>
        {(activity.items || []).map((item) => (
          <div key={item.id} className={styles.seckillItem}>
            <img
              className={styles.itemImage}
              src={item.productImage || 'https://via.placeholder.com/80'}
              alt={item.productName}
            />
            <div className={styles.itemInfo}>
              <div className={styles.itemName}>{item.productName}</div>
              <div className={styles.itemPrices}>
                <span className={styles.seckillPrice}>¥{(item.seckillPrice / 100).toFixed(2)}</span>
                <span className={styles.originalPrice}>¥{(item.originalPrice / 100).toFixed(2)}</span>
              </div>
              <div className={styles.itemStock}>剩余 {item.availableStock} 件</div>
            </div>
            <Button
              size="mini"
              color="danger"
              className={styles.buyBtn}
              disabled={isEnded || !isStarted || item.availableStock <= 0}
              onClick={() => handleBuy(item.id)}
            >
              {item.availableStock <= 0 ? '已抢光' : !isStarted ? '未开始' : '抢购'}
            </Button>
          </div>
        ))}
      </div>
    </div>
  )
}

export default function SeckillPage() {
  const navigate = useNavigate()
  const [activities, setActivities] = useState<SeckillActivity[]>([])

  useEffect(() => {
    listSeckillActivities().then((v) => setActivities(v ?? [])).catch(() => {})
  }, [])

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>限时秒杀</NavBar>
      </div>

      {activities.map((a) => (
        <ActivityCard key={a.id} activity={a} />
      ))}

      {activities.length === 0 && <div className={styles.empty}>暂无秒杀活动</div>}
    </div>
  )
}
