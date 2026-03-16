import { useState, useEffect, useCallback, useRef } from 'react'
import { SearchBar, InfiniteScroll } from 'antd-mobile'
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
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined)

  useEffect(() => {
    getHotWords().then((v) => setHotWords(v ?? [])).catch(() => {})
    if (isLoggedIn) {
      getSearchHistory(20).then((v) => setHistory(v ?? [])).catch(() => {})
    }
  }, [isLoggedIn])

  const doSearch = useCallback(async (kw: string, pageNum: number = 1) => {
    setShowSuggestions(false)
    setSuggestions([])
    const res = await searchProducts({ keyword: kw, page: pageNum, pageSize: 20 })
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
    </div>
  )
}
