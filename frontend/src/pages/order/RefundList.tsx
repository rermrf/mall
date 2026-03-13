import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, InfiniteScroll } from 'antd-mobile'
import { listRefunds, type RefundOrder } from '@/api/order'
import Price from '@/components/Price'
import styles from './refundList.module.css'

const REFUND_STATUS: Record<number, string> = {
  1: '待审核', 2: '已通过', 3: '退款中', 4: '已完成', 5: '已拒绝',
}

export default function RefundListPage() {
  const navigate = useNavigate()
  const [refunds, setRefunds] = useState<RefundOrder[]>([])
  const [hasMore, setHasMore] = useState(true)
  const [page, setPage] = useState(1)

  const loadMore = useCallback(async () => {
    const res = await listRefunds({ page, page_size: 10 })
    const list = res.list || []
    setRefunds((prev) => (page === 1 ? list : [...prev, ...list]))
    setHasMore(list.length >= 10)
    setPage((p) => p + 1)
  }, [page])

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>我的退款</NavBar>
      </div>

      {refunds.map((r) => (
        <div
          key={r.refund_no}
          className={styles.refundCard}
          onClick={() => navigate(`/refunds/${r.refund_no}`)}
        >
          <div className={styles.cardHeader}>
            <span className={styles.refundNo}>{r.refund_no}</span>
            <span className={styles.statusTag}>{REFUND_STATUS[r.status] || '未知'}</span>
          </div>
          <div className={styles.amount}>
            <Price value={r.refund_amount} size="md" />
          </div>
          <div className={styles.reason}>原因: {r.reason}</div>
          <div className={styles.time}>{new Date(r.ctime).toLocaleString('zh-CN')}</div>
        </div>
      ))}

      {refunds.length === 0 && !hasMore && (
        <div className={styles.empty}>暂无退款记录</div>
      )}

      <div className={styles.loadMore}>
        <InfiniteScroll loadMore={loadMore} hasMore={hasMore} />
      </div>
    </div>
  )
}
