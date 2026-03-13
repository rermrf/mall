import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { getShop, type Shop } from '@/api/shop'
import {
  listSeckillActivities,
  listAvailableCoupons,
  type SeckillActivity,
  type Coupon,
} from '@/api/marketing'
import styles from './home.module.css'

export default function HomePage() {
  const navigate = useNavigate()
  const [shop, setShop] = useState<Shop | null>(null)
  const [seckills, setSeckills] = useState<SeckillActivity[]>([])
  const [coupons, setCoupons] = useState<Coupon[]>([])

  useEffect(() => {
    getShop().then(setShop).catch(() => {})
    listSeckillActivities().then(setSeckills).catch(() => {})
    listAvailableCoupons().then(setCoupons).catch(() => {})
  }, [])

  const allSeckillItems = seckills.flatMap((s) => s.items || [])

  return (
    <div className={styles.page}>
      {shop && (
        <div className={styles.header}>
          {shop.logo && (
            <img className={styles.shopLogo} src={shop.logo} alt={shop.name} />
          )}
          <div className={styles.shopName}>{shop.name}</div>
        </div>
      )}

      {allSeckillItems.length > 0 && (
        <div className={styles.section}>
          <div className={styles.sectionTitle}>限时秒杀</div>
          <div className={styles.seckillScroll}>
            {allSeckillItems.map((item) => (
              <div key={item.id} className={styles.seckillCard}>
                <img
                  className={styles.seckillImage}
                  src={item.product_image || 'https://via.placeholder.com/120'}
                  alt={item.product_name}
                />
                <div className={styles.seckillInfo}>
                  <div className={styles.seckillName}>{item.product_name}</div>
                  <div className={styles.seckillPrice}>
                    ¥{(item.seckill_price / 100).toFixed(2)}
                  </div>
                  <div className={styles.seckillOriginal}>
                    ¥{(item.original_price / 100).toFixed(2)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {coupons.length > 0 && (
        <div className={styles.section}>
          <div className={styles.sectionTitle}>优惠券</div>
          <div className={styles.couponScroll}>
            {coupons.map((c) => (
              <div key={c.id} className={styles.couponCard}>
                <div className={styles.couponValue}>
                  {c.type === 1 ? `¥${(c.value / 100).toFixed(0)}` : `${c.value / 10}折`}
                </div>
                <div className={styles.couponName}>{c.name}</div>
                <div className={styles.couponCondition}>
                  满¥{(c.min_spend / 100).toFixed(0)}可用
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className={styles.section}>
        <div className={styles.sectionTitle}>为你推荐</div>
        <div style={{ textAlign: 'center', padding: 40, color: 'var(--color-text-secondary)' }}>
          <span onClick={() => navigate('/search')} style={{ cursor: 'pointer', color: 'var(--color-accent)' }}>
            去搜索发现更多好物 →
          </span>
        </div>
      </div>
    </div>
  )
}
