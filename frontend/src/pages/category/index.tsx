import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { SearchBar, InfiniteScroll, SpinLoading } from 'antd-mobile'
import { listCategories, listProducts } from '@/api/product'
import ProductCard from '@/components/ProductCard'
import type { Category } from '@/types/product'
import type { Product } from '@/types/product'
import styles from './category.module.css'

export default function CategoryPage() {
  const navigate = useNavigate()
  const [categories, setCategories] = useState<Category[]>([])
  const [activeId, setActiveId] = useState<number>(0)
  const [products, setProducts] = useState<Product[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const mainRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    listCategories().then((list) => {
      const cats = list ?? []
      setCategories(cats)
      if (cats.length > 0) setActiveId(cats[0].id)
    }).catch(() => {}).finally(() => setLoading(false))
  }, [])

  const fetchProducts = useCallback(async (categoryId: number, pageNum: number) => {
    const res = await listProducts({ categoryId, page: pageNum, pageSize: 20 })
    if (pageNum === 1) {
      setProducts(res.products || [])
    } else {
      setProducts((prev) => [...prev, ...(res.products || [])])
    }
    setTotal(res.total || 0)
    setPage(pageNum)
  }, [])

  useEffect(() => {
    if (activeId > 0) {
      setProducts([])
      setPage(1)
      setTotal(0)
      fetchProducts(activeId, 1)
      mainRef.current?.scrollTo(0, 0)
    }
  }, [activeId, fetchProducts])

  const handleCategoryClick = (id: number) => {
    if (id !== activeId) setActiveId(id)
  }

  const loadMore = async () => {
    if (activeId > 0) await fetchProducts(activeId, page + 1)
  }

  if (loading) {
    return <div className={styles.loading}><SpinLoading color="default" /></div>
  }

  return (
    <div className={styles.page}>
      <div className={styles.searchBar}>
        <SearchBar placeholder="搜索商品" onFocus={() => navigate('/search')} />
      </div>
      <div className={styles.body}>
        <div className={styles.sidebar}>
          {categories.map((cat) => (
            <div
              key={cat.id}
              className={`${styles.sidebarItem} ${activeId === cat.id ? styles.sidebarItemActive : ''}`}
              onClick={() => handleCategoryClick(cat.id)}
            >
              {cat.name}
            </div>
          ))}
        </div>
        <div className={styles.main} ref={mainRef}>
          {products.length > 0 ? (
            <div className={styles.grid}>
              {products.map((p) => (
                <ProductCard key={p.id} product={p} />
              ))}
            </div>
          ) : (
            <div className={styles.empty}>该分类下暂无商品</div>
          )}
          <InfiniteScroll loadMore={loadMore} hasMore={products.length < total} />
        </div>
      </div>
    </div>
  )
}
