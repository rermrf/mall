import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { NavBar, Button, Toast, Swiper, Stepper, Skeleton, ErrorBlock } from 'antd-mobile'
import { addCartItem } from '@/api/cart'
import { getProductDetail } from '@/api/product'
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

  const stateProduct = (location.state as { product?: Product })?.product
  const [product, setProduct] = useState<Product | null>(stateProduct || null)
  const [stock, setStock] = useState<number | null>(null)
  const [quantity, setQuantity] = useState(1)
  const [adding, setAdding] = useState(false)
  const [loading, setLoading] = useState(!stateProduct)
  const [error, setError] = useState(false)

  const imageList = useMemo(() => {
    if (!product) return []
    const list: string[] = []
    if (product.mainImage) list.push(product.mainImage)
    if (product.images) {
      try {
        const extras = JSON.parse(product.images) as string[]
        for (const img of extras) {
          if (img && !list.includes(img)) list.push(img)
        }
      } catch {}
    }
    return list.length > 0 ? list : ['https://via.placeholder.com/400']
  }, [product])

  useEffect(() => {
    if (!id) return
    if (!stateProduct) {
      setLoading(true)
      getProductDetail(Number(id))
        .then(setProduct)
        .catch(() => setError(true))
        .finally(() => setLoading(false))
    }
    getStock(Number(id)).then((s) => setStock(s.available)).catch(() => {})
  }, [id, stateProduct])

  if (loading) {
    return (
      <div className={styles.page}>
        <NavBar onBack={() => navigate(-1)}>商品详情</NavBar>
        <Skeleton.Paragraph lineCount={5} animated />
      </div>
    )
  }

  if (error || !product) {
    return (
      <div className={styles.page}>
        <NavBar onBack={() => navigate(-1)}>商品详情</NavBar>
        <ErrorBlock
          status="empty"
          title="商品不存在"
          description="该商品可能已下架或链接有误"
        />
      </div>
    )
  }

  const outOfStock = stock !== null && stock <= 0

  const requireLogin = () => {
    if (!isLoggedIn) {
      navigate(`/login?redirect=${encodeURIComponent(location.pathname)}`, { state: location.state })
      return true
    }
    return false
  }

  const handleAddCart = async () => {
    if (requireLogin()) return
    setAdding(true)
    try {
      await addCartItem({ skuId: Number(id), productId: product.id, quantity })
      Toast.show('已加入购物车')
    } catch (e: unknown) {
      Toast.show((e as Error).message || '添加失败')
    } finally {
      setAdding(false)
    }
  }

  const handleBuyNow = () => {
    if (requireLogin()) return
    navigate('/order/confirm', {
      state: {
        directBuy: true,
        product: { ...product, skuId: Number(id) },
        quantity,
      },
    })
  }

  return (
    <div className={styles.page}>
      <NavBar onBack={() => navigate(-1)} style={{ background: 'transparent', position: 'absolute', zIndex: 10, width: '100%' }} />

      {imageList.length > 1 ? (
        <Swiper className={styles.swiper}>
          {imageList.map((img, i) => (
            <Swiper.Item key={i}>
              <img className={styles.image} src={img} alt={product.name} />
            </Swiper.Item>
          ))}
        </Swiper>
      ) : (
        <img className={styles.image} src={imageList[0]} alt={product.name} />
      )}

      <div className={styles.info}>
        <div className={styles.name}>{product.name}</div>
        {product.subtitle && <div className={styles.subtitle}>{product.subtitle}</div>}
        <div className={styles.priceRow}>
          <Price value={product.price} original={product.originalPrice} size='lg' />
          {product.sales > 0 && <span className={styles.sales}>已售{product.sales}</span>}
        </div>
        {stock !== null && (
          <div className={`${styles.stock} ${stock < 10 ? styles.stockLow : ''}`}>
            {stock > 0 ? `库存 ${stock} 件` : '已售罄'}
          </div>
        )}
      </div>

      <div className={styles.quantitySection}>
        <span className={styles.quantityLabel}>数量</span>
        <Stepper
          min={1}
          max={stock ?? 999}
          value={quantity}
          onChange={(v) => setQuantity(v as number)}
          disabled={outOfStock}
        />
      </div>

      {product.description && (
        <div className={styles.description}>
          <div className={styles.descTitle}>商品详情</div>
          <div className={styles.descText}>{product.description}</div>
        </div>
      )}

      <div className={styles.footer}>
        <Button fill='outline' className={styles.cartBtn} loading={adding} onClick={handleAddCart} disabled={outOfStock}>
          加入购物车
        </Button>
        <Button color='primary' className={styles.buyBtn} onClick={handleBuyNow} disabled={outOfStock}>
          立即购买
        </Button>
      </div>
    </div>
  )
}
