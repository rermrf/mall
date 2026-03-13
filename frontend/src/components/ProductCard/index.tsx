import { useNavigate } from 'react-router-dom'
import Price from '@/components/Price'
import type { Product } from '@/types/product'
import styles from './ProductCard.module.css'

export default function ProductCard({ product }: { product: Product }) {
  const navigate = useNavigate()

  return (
    <div className={styles.card} onClick={() => navigate(`/product/${product.id}`, { state: { product } })}>
      <img
        className={styles.image}
        src={product.main_image || 'https://via.placeholder.com/300'}
        alt={product.name}
        loading='lazy'
      />
      <div className={styles.info}>
        <div className={styles.name}>{product.name}</div>
        <div className={styles.meta}>
          <Price value={product.price} original={product.original_price} size='sm' />
          {product.sales > 0 && (
            <span className={styles.sales}>已售{product.sales}</span>
          )}
        </div>
      </div>
    </div>
  )
}
