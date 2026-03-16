import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { NavBar, Button, Dialog, Toast, SpinLoading } from 'antd-mobile'
import { getOrder, cancelOrder, confirmReceive, applyRefund, type Order } from '@/api/order'
import { getOrderLogistics, type Shipment } from '@/api/logistics'
import Price from '@/components/Price'
import styles from './detail.module.css'

const STATUS_MAP: Record<number, string> = {
  1: '待付款', 2: '待发货', 3: '待收货', 4: '已收货', 5: '已完成', 6: '已取消', 7: '已退款',
}

function formatTime(ts: number | string) {
  if (!ts) return '-'
  const d = new Date(typeof ts === 'number' ? ts * 1000 : ts)
  return d.toLocaleString('zh-CN')
}

export default function OrderDetailPage() {
  const { orderNo } = useParams<{ orderNo: string }>()
  const navigate = useNavigate()
  const [order, setOrder] = useState<Order | null>(null)
  const [logistics, setLogistics] = useState<Shipment | null>(null)
  const [showAllTracks, setShowAllTracks] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!orderNo) return
    getOrder(orderNo)
      .then((o) => {
        setOrder(o)
        if (o.status === 3 || o.status === 4 || o.status === 5) {
          getOrderLogistics(orderNo).then(setLogistics).catch(() => {})
        }
      })
      .catch((e: unknown) => Toast.show((e as Error).message || '获取订单失败'))
      .finally(() => setLoading(false))
  }, [orderNo])

  const handleCancel = () => {
    if (!orderNo) return
    Dialog.confirm({
      content: '确定取消该订单？',
      onConfirm: async () => {
        try {
          await cancelOrder(orderNo)
          Toast.show('订单已取消')
          setOrder((prev) => prev ? { ...prev, status: 6 } : prev)
        } catch (e: unknown) {
          Toast.show((e as Error).message || '取消失败')
        }
      },
    })
  }

  const handleConfirmReceive = () => {
    if (!orderNo) return
    Dialog.confirm({
      content: '确认已收到商品？',
      onConfirm: async () => {
        try {
          await confirmReceive(orderNo)
          Toast.show('已确认收货')
          setOrder((prev) => prev ? { ...prev, status: 4 } : prev)
        } catch (e: unknown) {
          Toast.show((e as Error).message || '操作失败')
        }
      },
    })
  }

  const handleRefund = () => {
    if (!orderNo) return
    Dialog.confirm({
      content: '确定申请退款？',
      onConfirm: async () => {
        try {
          const res = await applyRefund(orderNo, { reason: '买家申请退款' })
          Toast.show('已提交退款申请')
          navigate(`/refunds/${res.refundNo}`)
        } catch (e: unknown) {
          Toast.show((e as Error).message || '申请失败')
        }
      },
    })
  }

  if (loading) {
    return (
      <div className={styles.page}>
        <div className={styles.navBar}>
          <NavBar onBack={() => navigate(-1)}>订单详情</NavBar>
        </div>
        <div className={styles.loading}><SpinLoading color="default" /></div>
      </div>
    )
  }

  if (!order) return null

  const tracks = logistics?.tracks || []
  const visibleTracks = showAllTracks ? tracks : tracks.slice(0, 1)

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>订单详情</NavBar>
      </div>

      <div className={styles.statusBanner}>
        <div className={styles.statusText}>{STATUS_MAP[order.status] || '未知状态'}</div>
      </div>

      {logistics && (
        <div className={styles.card}>
          <div className={styles.cardTitle}>物流信息</div>
          <div className={styles.logisticsInfo}>
            {logistics.carrierName} {logistics.trackingNo}
          </div>
          {visibleTracks.map((t, i) => (
            <div key={i} className={styles.trackItem}>
              <div>{t.description}</div>
              <div className={styles.trackTime}>{formatTime(t.trackTime)}</div>
            </div>
          ))}
          {tracks.length > 1 && (
            <div className={styles.trackToggle} onClick={() => setShowAllTracks(!showAllTracks)}>
              {showAllTracks ? '收起' : `查看全部 ${tracks.length} 条物流记录`}
            </div>
          )}
        </div>
      )}

      <div className={styles.card}>
        <div className={styles.cardTitle}>收货地址</div>
        <div className={styles.addressInfo}>
          <span className={styles.addressName}>{order.receiverName}</span>
          <span className={styles.addressPhone}>{order.receiverPhone}</span>
        </div>
        <div className={styles.addressDetail}>{order.receiverAddress}</div>
      </div>

      <div className={styles.card}>
        <div className={styles.cardTitle}>商品信息</div>
        {(order.items || []).map((item) => (
          <div key={item.id} className={styles.orderItem}>
            <img
              className={styles.itemImage}
              src={item.productImage || 'https://via.placeholder.com/70'}
              alt=""
            />
            <div className={styles.itemInfo}>
              <div className={styles.itemName}>{item.productName}</div>
              {item.skuSpec && <div className={styles.itemSpec}>{item.skuSpec}</div>}
              <div className={styles.itemBottom}>
                <Price value={item.price} size="sm" />
                <span className={styles.itemQty}>x{item.quantity}</span>
              </div>
            </div>
          </div>
        ))}
      </div>

      <div className={styles.card}>
        <div className={styles.cardTitle}>价格明细</div>
        <div className={styles.priceRow}>
          <span className={styles.priceLabel}>商品金额</span>
          <span><Price value={order.totalAmount} size="sm" /></span>
        </div>
        <div className={styles.priceRow}>
          <span className={styles.priceLabel}>运费</span>
          <span><Price value={order.freightAmount} size="sm" /></span>
        </div>
        <div className={styles.priceRow}>
          <span className={styles.priceLabel}>优惠</span>
          <span>-<Price value={order.discountAmount} size="sm" /></span>
        </div>
        <div className={styles.priceRow}>
          <span className={styles.priceLabel}>实付</span>
          <span className={styles.priceTotal}><Price value={order.payAmount} size="md" /></span>
        </div>
      </div>

      <div className={styles.card}>
        <div className={styles.cardTitle}>订单信息</div>
        <div className={styles.infoRow}>
          <span className={styles.infoLabel}>订单编号</span>
          <span className={styles.infoValue}>{order.orderNo}</span>
        </div>
        <div className={styles.infoRow}>
          <span className={styles.infoLabel}>下单时间</span>
          <span className={styles.infoValue}>{formatTime(order.ctime)}</span>
        </div>
        {order.payTime > 0 && (
          <div className={styles.infoRow}>
            <span className={styles.infoLabel}>支付时间</span>
            <span className={styles.infoValue}>{formatTime(order.payTime)}</span>
          </div>
        )}
        {order.shipTime > 0 && (
          <div className={styles.infoRow}>
            <span className={styles.infoLabel}>发货时间</span>
            <span className={styles.infoValue}>{formatTime(order.shipTime)}</span>
          </div>
        )}
        {order.receiveTime > 0 && (
          <div className={styles.infoRow}>
            <span className={styles.infoLabel}>收货时间</span>
            <span className={styles.infoValue}>{formatTime(order.receiveTime)}</span>
          </div>
        )}
        {order.remark && (
          <div className={styles.infoRow}>
            <span className={styles.infoLabel}>备注</span>
            <span className={styles.infoValue}>{order.remark}</span>
          </div>
        )}
      </div>

      {(order.status === 1 || order.status === 2 || order.status === 3) && (
        <div className={styles.footer}>
          {order.status === 1 && (
            <>
              <Button className={styles.footerBtn} onClick={handleCancel}>取消订单</Button>
              <Button
                color="primary"
                className={styles.footerBtn}
                onClick={() => navigate(`/payment/${order.orderNo}`)}
              >
                去付款
              </Button>
            </>
          )}
          {order.status === 3 && (
            <Button color="primary" className={styles.footerBtn} onClick={handleConfirmReceive}>
              确认收货
            </Button>
          )}
          {(order.status === 2 || order.status === 3) && (
            <Button className={styles.footerBtn} onClick={handleRefund}>申请退款</Button>
          )}
        </div>
      )}
    </div>
  )
}
