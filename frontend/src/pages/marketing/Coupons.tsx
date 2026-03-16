import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Tabs, Button, Toast } from 'antd-mobile'
import { listAvailableCoupons, receiveCoupon, listMyCoupons, type Coupon, type UserCoupon } from '@/api/marketing'
import { useAuthStore } from '@/stores/auth'
import styles from './coupons.module.css'

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('zh-CN')
}

export default function CouponsPage() {
  const navigate = useNavigate()
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const [activeTab, setActiveTab] = useState('available')
  const [available, setAvailable] = useState<Coupon[]>([])
  const [mine, setMine] = useState<UserCoupon[]>([])

  useEffect(() => {
    listAvailableCoupons().then((v) => setAvailable(v ?? [])).catch(() => {})
    if (isLoggedIn) {
      listMyCoupons().then((v) => setMine(v ?? [])).catch(() => {})
    }
  }, [isLoggedIn])

  const handleReceive = async (id: number) => {
    if (!isLoggedIn) {
      navigate('/login')
      return
    }
    try {
      await receiveCoupon(id)
      Toast.show('领取成功')
      setAvailable((prev) =>
        prev.map((c) => (c.id === id ? { ...c, remaining: c.remaining - 1 } : c))
      )
      listMyCoupons().then((v) => setMine(v ?? [])).catch(() => {})
    } catch (e: unknown) {
      Toast.show((e as Error).message || '领取失败')
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>优惠券</NavBar>
      </div>

      <Tabs className={styles.tabs} activeKey={activeTab} onChange={setActiveTab}>
        <Tabs.Tab key="available" title="领券中心" />
        <Tabs.Tab key="mine" title="我的优惠券" />
      </Tabs>

      <div className={styles.content}>
        {activeTab === 'available' && (
          <>
            {available.map((c) => (
              <div key={c.id} className={styles.couponCard}>
                <div className={styles.couponLeft}>
                  <div className={styles.couponValue}>
                    {c.type === 1 ? `¥${(c.value / 100).toFixed(0)}` : `${c.value / 10}折`}
                  </div>
                  <div className={styles.couponCondition}>满{(c.minSpend / 100).toFixed(0)}可用</div>
                </div>
                <div className={styles.couponRight}>
                  <div className={styles.couponInfo}>
                    <div className={styles.couponName}>{c.name}</div>
                    <div className={styles.couponTime}>{formatDate(c.startTime)} - {formatDate(c.endTime)}</div>
                    <div className={styles.couponRemaining}>剩余 {c.remaining} 张</div>
                  </div>
                  <Button
                    size="mini"
                    color="primary"
                    className={styles.receiveBtn}
                    disabled={c.remaining <= 0}
                    onClick={() => handleReceive(c.id)}
                  >
                    {c.remaining > 0 ? '领取' : '已抢光'}
                  </Button>
                </div>
              </div>
            ))}
            {available.length === 0 && <div className={styles.empty}>暂无可领优惠券</div>}
          </>
        )}

        {activeTab === 'mine' && (
          <>
            {mine.map((c) => (
              <div key={c.id} className={styles.couponCard}>
                <div className={styles.couponLeft}>
                  <div className={styles.couponValue}>
                    {c.type === 1 ? `¥${(c.value / 100).toFixed(0)}` : `${c.value / 10}折`}
                  </div>
                  <div className={styles.couponCondition}>满{(c.minSpend / 100).toFixed(0)}可用</div>
                </div>
                <div className={styles.couponRight}>
                  <div className={styles.couponInfo}>
                    <div className={styles.couponName}>{c.name}</div>
                    <div className={styles.couponTime}>{formatDate(c.startTime)} - {formatDate(c.endTime)}</div>
                  </div>
                </div>
              </div>
            ))}
            {mine.length === 0 && <div className={styles.empty}>暂无优惠券</div>}
          </>
        )}
      </div>
    </div>
  )
}
