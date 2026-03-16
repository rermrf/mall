import { useState, useEffect, useCallback, useRef } from 'react'
import { SearchBar, InfiniteScroll, Input, Popup, Button } from 'antd-mobile'
import { searchProducts, getHotWords, getSuggestions, getSearchHistory, clearSearchHistory } from '@/api/search'
import { useAuthStore } from '@/stores/auth'
import ProductCard from '@/components/ProductCard'
import type { Product } from '@/types/product'
import styles from './search.module.css'

export default function SearchPage() {
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  const [keyword, setKeyword] = useState('')
  const [products, setProducts] = useState<Product[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [hotWords, setHotWords] = useState<string[]>([])
  const [history, setHistory] = useState<string[]>([])
  const [suggestions, setSuggestions] = useState<string[]>([])
  const [showSuggestions, setShowSuggestions] = useState(false)
  const [searched, setSearched] = useState(false)
  const [sortBy, setSortBy] = useState('default')
  const [showFilter, setShowFilter] = useState(false)
  const [filterPriceMin, setFilterPriceMin] = useState('')
  const [filterPriceMax, setFilterPriceMax] = useState('')
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined)

  useEffect(() => {
    getHotWords().then((v) => setHotWords(v ?? [])).catch(() => {})
    if (isLoggedIn) {
      getSearchHistory(20).then((v) => setHistory(v ?? [])).catch(() => {})
    }
  }, [isLoggedIn])

  useEffect(() => {
    if (keyword.trim() && searched) {
      setProducts([])
      setPage(1)
      doSearch(keyword.trim(), 1)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sortBy])

  const doSearch = useCallback(async (kw: string, pageNum: number = 1) => {
    setShowSuggestions(false)
    setSuggestions([])
    const res = await searchProducts({
      keyword: kw,
      page: pageNum,
      pageSize: 20,
      sortBy: sortBy !== 'default' ? sortBy : undefined,
      priceMin: filterPriceMin ? Number(filterPriceMin) * 100 : undefined,
      priceMax: filterPriceMax ? Number(filterPriceMax) * 100 : undefined,
    })
    if (pageNum === 1) {
      setProducts(res.products || [])
    } else {
      setProducts((prev) => [...prev, ...(res.products || [])])
    }
    setTotal(res.total || 0)
    setPage(pageNum)
    setSearched(true)
    if (isLoggedIn) {
      getSearchHistory(20).then((v) => setHistory(v ?? [])).catch(() => {})
    }
  }, [isLoggedIn])

  const handleSearch = (val: string) => {
    setKeyword(val)
    if (val.trim()) {
      doSearch(val.trim())
    }
  }

  const handleInputChange = (val: string) => {
    setKeyword(val)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    if (!val.trim()) {
      setSuggestions([])
      setShowSuggestions(false)
      return
    }
    debounceRef.current = setTimeout(() => {
      getSuggestions(val.trim()).then((list) => {
        setSuggestions(list || [])
        setShowSuggestions((list || []).length > 0)
      }).catch(() => {})
    }, 300)
  }

  const handleClearHistory = async () => {
    try {
      await clearSearchHistory()
      setHistory([])
    } catch {
      // ignore
    }
  }

  const loadMore = async () => {
    if (!keyword.trim()) return
    await doSearch(keyword.trim(), page + 1)
  }

  const hasMore = products.length < total

  return (
    <div className={styles.page}>
      <div className={styles.searchBar}>
        <SearchBar
          placeholder='搜索商品'
          value={keyword}
          onChange={handleInputChange}
          onSearch={handleSearch}
          onFocus={() => {
            if (suggestions.length > 0) setShowSuggestions(true)
          }}
          onClear={() => { setSearched(false); setProducts([]); setSuggestions([]); setShowSuggestions(false) }}
        />
      </div>

      <div className={styles.suggestionsWrap}>
        {showSuggestions && suggestions.length > 0 && (
          <div className={styles.suggestions}>
            {suggestions.map((s) => (
              <div key={s} className={styles.suggestionItem} onClick={() => handleSearch(s)}>
                {s}
              </div>
            ))}
          </div>
        )}
      </div>

      {!searched && (
        <>
          {hotWords.length > 0 && (
            <div className={styles.hotSection}>
              <div className={styles.hotTitle}>热门搜索</div>
              <div className={styles.hotTags}>
                {hotWords.map((w) => (
                  <span key={w} className={styles.hotTag} onClick={() => handleSearch(w)}>
                    {w}
                  </span>
                ))}
              </div>
            </div>
          )}

          {history.length > 0 && (
            <div className={styles.historySection}>
              <div className={styles.historyHeader}>
                <span className={styles.historyTitle}>搜索历史</span>
                <span className={styles.historyClear} onClick={handleClearHistory}>清空</span>
              </div>
              <div className={styles.historyTags}>
                {history.map((w) => (
                  <span key={w} className={styles.historyTag} onClick={() => handleSearch(w)}>
                    {w}
                  </span>
                ))}
              </div>
            </div>
          )}
        </>
      )}

      {searched && (
        <>
          <div className={styles.sortBar}>
            {[
              { key: 'default', label: '综合' },
              { key: 'sales_desc', label: '销量' },
              { key: 'price_asc', label: '价格↑' },
              { key: 'price_desc', label: '价格↓' },
            ].map((s) => (
              <span
                key={s.key}
                className={`${styles.sortItem} ${sortBy === s.key ? styles.sortItemActive : ''}`}
                onClick={() => setSortBy(s.key)}
              >
                {s.label}
              </span>
            ))}
            <span className={styles.sortItem} onClick={() => setShowFilter(true)}>筛选</span>
          </div>
          <div className={styles.grid}>
            {products.map((p) => (
              <ProductCard key={p.id} product={p} />
            ))}
          </div>
          {products.length === 0 && (
            <div className={styles.empty}>没有找到相关商品</div>
          )}
          <InfiniteScroll loadMore={loadMore} hasMore={hasMore} />
        </>
      )}

      <Popup visible={showFilter} onMaskClick={() => setShowFilter(false)} position="right" bodyStyle={{ width: '75vw', padding: 16 }}>
        <div className={styles.filterPanel}>
          <div className={styles.filterTitle}>筛选</div>
          <div className={styles.filterSection}>
            <div className={styles.filterLabel}>价格区间 (元)</div>
            <div className={styles.filterRow}>
              <Input placeholder="最低价" type="number" value={filterPriceMin} onChange={setFilterPriceMin} style={{ flex: 1 }} />
              <span style={{ margin: '0 8px', color: '#999' }}>-</span>
              <Input placeholder="最高价" type="number" value={filterPriceMax} onChange={setFilterPriceMax} style={{ flex: 1 }} />
            </div>
          </div>
          <div className={styles.filterActions}>
            <Button block fill="outline" onClick={() => { setFilterPriceMin(''); setFilterPriceMax('') }}>
              重置
            </Button>
            <Button block color="primary" onClick={() => {
              setShowFilter(false)
              setProducts([])
              setPage(1)
              if (keyword.trim()) doSearch(keyword.trim(), 1)
            }}>
              确定
            </Button>
          </div>
        </div>
      </Popup>
    </div>
  )
}
