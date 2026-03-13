import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Tabs } from 'antd-mobile'
import { listMyCoupons, type UserCoupon } from '@/api/marketing'
import styles from './myCoupons.module.css'

const STATUS_TABS = [
  { key: '0', title: '全部' },
  { key: '1', title: '可用' },
  { key: '2', title: '已使用' },
  { key: '3', title: '已过期' },
]

const STATUS_LABEL: Record<number, string> = { 1: '可用', 2: '已使用', 3: '已过期' }

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('zh-CN')
}

export default function MyCouponsPage() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('0')
  const [coupons, setCoupons] = useState<UserCoupon[]>([])

  useEffect(() => {
    const status = activeTab === '0' ? undefined : Number(activeTab)
    listMyCoupons(status).then(setCoupons).catch(() => setCoupons([]))
  }, [activeTab])

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>我的优惠券</NavBar>
      </div>

      <Tabs className={styles.tabs} activeKey={activeTab} onChange={setActiveTab}>
        {STATUS_TABS.map((t) => (
          <Tabs.Tab key={t.key} title={t.title} />
        ))}
      </Tabs>

      <div className={styles.content}>
        {coupons.map((c) => (
          <div
            key={c.id}
            className={`${styles.couponCard} ${c.status !== 1 ? styles.couponDisabled : ''}`}
          >
            <div className={styles.couponLeft}>
              <div className={styles.couponValue}>
                {c.type === 1 ? `¥${(c.value / 100).toFixed(0)}` : `${c.value / 10}折`}
              </div>
              <div className={styles.couponCondition}>满{(c.min_spend / 100).toFixed(0)}可用</div>
            </div>
            <div className={styles.couponRight}>
              <div className={styles.couponName}>{c.name}</div>
              <div className={styles.couponTime}>{formatDate(c.start_time)} - {formatDate(c.end_time)}</div>
              <span className={styles.statusTag}>{STATUS_LABEL[c.status] || '未知'}</span>
            </div>
          </div>
        ))}
        {coupons.length === 0 && <div className={styles.empty}>暂无优惠券</div>}
      </div>
    </div>
  )
}
