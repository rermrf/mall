import { useState, useEffect } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { NavBar, Button, Toast } from 'antd-mobile'
import { addCartItem } from '@/api/cart'
import { getStock } from '@/api/inventory'
import { useAuthStore } from '@/stores/auth'
import Price from '@/components/Price'
import type { Product } from '@/types/product'
import styles from './detail.module.css'

export default function ProductDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)

  // Product data passed from search/list via route state
  const product = (location.state as { product?: Product })?.product
  const [stock, setStock] = useState<number | null>(null)
  const [adding, setAdding] = useState(false)

  useEffect(() => {
    if (id) {
      getStock(Number(id)).then((s) => setStock(s.available)).catch(() => {})
    }
  }, [id])

  if (!product) {
    return (
      <div className={styles.page}>
        <NavBar onBack={() => navigate(-1)}>商品详情</NavBar>
        <div style={{ textAlign: 'center', padding: 60, color: 'var(--color-text-secondary)' }}>
          商品信息加载失败
        </div>
      </div>
    )
  }

  const handleAddCart = async () => {
    if (!isLoggedIn) {
      navigate(`/login?redirect=${encodeURIComponent(location.pathname)}`, { state: location.state })
      return
    }
    setAdding(true)
    try {
      await addCartItem({ sku_id: Number(id), product_id: product.id, quantity: 1 })
      Toast.show('已加入购物车')
    } catch (e: unknown) {
      Toast.show((e as Error).message || '添加失败')
    } finally {
      setAdding(false)
    }
  }

  return (
    <div className={styles.page}>
      <NavBar onBack={() => navigate(-1)} style={{ background: 'transparent', position: 'absolute', zIndex: 10, width: '100%' }} />
      <img
        className={styles.image}
        src={product.main_image || 'https://via.placeholder.com/400'}
        alt={product.name}
      />
      <div className={styles.info}>
        <div className={styles.name}>{product.name}</div>
        <div className={styles.priceRow}>
          <Price value={product.price} original={product.original_price} size='lg' />
        </div>
        {stock !== null && (
          <div className={`${styles.stock} ${stock < 10 ? styles.stockLow : ''}`}>
            {stock > 0 ? `库存 ${stock} 件` : '已售罄'}
          </div>
        )}
      </div>

      {product.description && (
        <div className={styles.description}>
          <div className={styles.descTitle}>商品详情</div>
          <div className={styles.descText}>{product.description}</div>
        </div>
      )}

      <div className={styles.footer}>
        <Button fill='outline' className={styles.cartBtn} loading={adding} onClick={handleAddCart}>
          加入购物车
        </Button>
        <Button color='primary' className={styles.buyBtn} onClick={handleAddCart}>
          立即购买
        </Button>
      </div>
    </div>
  )
}
