import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Skeleton, SearchBar } from 'antd-mobile'
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
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      getShop().then(setShop).catch(() => {}),
      listSeckillActivities().then((v) => setSeckills(v ?? [])).catch(() => {}),
      listAvailableCoupons().then((v) => setCoupons(v ?? [])).catch(() => {}),
    ]).finally(() => setLoading(false))
  }, [])

  const allSeckillItems = seckills.flatMap((s) => s.items || [])

  if (loading) {
    return (
      <div className={styles.page}>
        <Skeleton.Title animated />
        <Skeleton.Paragraph lineCount={5} animated />
      </div>
    )
  }

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

      <div style={{ padding: '0 12px 12px' }}>
        <SearchBar placeholder="搜索商品" onFocus={() => navigate('/search')} />
      </div>

      {allSeckillItems.length > 0 && (
        <div className={styles.section}>
          <div className={styles.sectionTitle}>限时秒杀</div>
          <div className={styles.seckillScroll}>
            {allSeckillItems.map((item) => (
              <div key={item.id} className={styles.seckillCard}>
                <img
                  className={styles.seckillImage}
                  src={item.productImage || 'https://via.placeholder.com/120'}
                  alt={item.productName}
                />
                <div className={styles.seckillInfo}>
                  <div className={styles.seckillName}>{item.productName}</div>
                  <div className={styles.seckillPrice}>
                    ¥{(item.seckillPrice / 100).toFixed(2)}
                  </div>
                  <div className={styles.seckillOriginal}>
                    ¥{(item.originalPrice / 100).toFixed(2)}
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
                  满¥{(c.minSpend / 100).toFixed(0)}可用
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
