import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Tabs, Button, InfiniteScroll } from 'antd-mobile'
import { listOrders, type Order } from '@/api/order'
import Price from '@/components/Price'
import styles from './list.module.css'

const STATUS_MAP: Record<number, string> = {
  1: '待付款', 2: '待发货', 3: '待收货', 4: '已收货', 5: '已完成', 6: '已取消', 7: '已退款',
}

const TABS = [
  { key: '0', title: '全部' },
  { key: '1', title: '待付款' },
  { key: '2', title: '待发货' },
  { key: '3', title: '待收货' },
  { key: '5', title: '已完成' },
]

export default function OrderListPage() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('0')
  const [orders, setOrders] = useState<Order[]>([])
  const [hasMore, setHasMore] = useState(true)
  const [page, setPage] = useState(1)

  const loadMore = useCallback(async () => {
    const status = activeTab === '0' ? undefined : Number(activeTab)
    const res = await listOrders({ status, page, page_size: 10 })
    const list = res.list || []
    setOrders((prev) => (page === 1 ? list : [...prev, ...list]))
    setHasMore(list.length >= 10)
    setPage((p) => p + 1)
  }, [activeTab, page])

  const handleTabChange = (key: string) => {
    setActiveTab(key)
    setOrders([])
    setPage(1)
    setHasMore(true)
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>我的订单</NavBar>
      </div>

      <Tabs className={styles.tabs} activeKey={activeTab} onChange={handleTabChange}>
        {TABS.map((tab) => (
          <Tabs.Tab key={tab.key} title={tab.title} />
        ))}
      </Tabs>

      <div className={styles.content}>
        {orders.map((order) => (
          <div
            key={order.order_no}
            className={styles.orderCard}
            onClick={() => navigate(`/orders/${order.order_no}`)}
          >
            <div className={styles.cardHeader}>
              <span className={styles.orderNo}>订单号: {order.order_no}</span>
              <span className={styles.statusTag}>{STATUS_MAP[order.status] || '未知'}</span>
            </div>

            <div className={styles.cardBody}>
              <div className={styles.itemsRow}>
                {(order.items || []).slice(0, 3).map((item) => (
                  <img
                    key={item.id}
                    className={styles.itemThumb}
                    src={item.product_image || 'https://via.placeholder.com/60'}
                    alt={item.product_name}
                  />
                ))}
              </div>
            </div>

            <div className={styles.cardFooter}>
              <span className={styles.payAmount}>
                实付 <Price value={order.pay_amount} size="sm" />
              </span>
              <div className={styles.actions} onClick={(e) => e.stopPropagation()}>
                {order.status === 1 && (
                  <Button
                    size="mini"
                    color="primary"
                    className={styles.actionBtn}
                    onClick={() => navigate(`/payment/${order.order_no}`)}
                  >
                    去付款
                  </Button>
                )}
                {order.status === 3 && (
                  <Button
                    size="mini"
                    color="primary"
                    className={styles.actionBtn}
                    onClick={() => navigate(`/orders/${order.order_no}`)}
                  >
                    确认收货
                  </Button>
                )}
                <Button
                  size="mini"
                  className={styles.actionBtn}
                  onClick={() => navigate(`/orders/${order.order_no}`)}
                >
                  查看详情
                </Button>
              </div>
            </div>
          </div>
        ))}

        {orders.length === 0 && !hasMore && (
          <div className={styles.empty}>暂无订单</div>
        )}

        <div className={styles.loadMore}>
          <InfiniteScroll loadMore={loadMore} hasMore={hasMore} />
        </div>
      </div>
    </div>
  )
}
