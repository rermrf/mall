import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { NavBar, Toast, SpinLoading, Button, Dialog } from 'antd-mobile'
import { getRefund, cancelRefund, type RefundOrder } from '@/api/order'
import Price from '@/components/Price'
import styles from './refundDetail.module.css'

const REFUND_STATUS: Record<number, string> = {
  1: '待审核', 2: '已通过', 3: '退款中', 4: '已完成', 5: '已拒绝',
}

const REFUND_TYPE: Record<number, string> = {
  1: '仅退款', 2: '退货退款',
}

function getBannerClass(status: number) {
  if (status === 4) return styles.statusSuccess
  if (status === 5) return styles.statusRejected
  if (status === 1) return styles.statusPending
  return styles.statusDefault
}

interface TimelineEntry { title: string; time: string }

function getTimeline(refund: RefundOrder): TimelineEntry[] {
  const items: TimelineEntry[] = []
  items.push({ title: '提交退款申请', time: new Date(refund.ctime).toLocaleString('zh-CN') })
  if (refund.status >= 2 && refund.status !== 5) {
    items.push({ title: '审核通过', time: new Date(refund.utime).toLocaleString('zh-CN') })
  }
  if (refund.status >= 3 && refund.status !== 5) {
    items.push({ title: '退款处理中', time: new Date(refund.utime).toLocaleString('zh-CN') })
  }
  if (refund.status === 4) {
    items.push({ title: '退款完成', time: new Date(refund.utime).toLocaleString('zh-CN') })
  }
  if (refund.status === 5) {
    items.push({ title: '退款被拒绝', time: new Date(refund.utime).toLocaleString('zh-CN') })
  }
  return items
}

export default function RefundDetailPage() {
  const { refundNo } = useParams<{ refundNo: string }>()
  const navigate = useNavigate()
  const [refund, setRefund] = useState<RefundOrder | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!refundNo) return
    getRefund(refundNo)
      .then(setRefund)
      .catch((e: unknown) => Toast.show((e as Error).message || '获取退款信息失败'))
      .finally(() => setLoading(false))
  }, [refundNo])

  if (loading) {
    return (
      <div className={styles.page}>
        <div className={styles.navBar}>
          <NavBar onBack={() => navigate(-1)}>退款详情</NavBar>
        </div>
        <div className={styles.loading}><SpinLoading color="default" /></div>
      </div>
    )
  }

  if (!refund) return null

  const timeline = getTimeline(refund)

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>退款详情</NavBar>
      </div>

      <div className={`${styles.statusBanner} ${getBannerClass(refund.status)}`}>
        <div className={styles.statusText}>{REFUND_STATUS[refund.status] || '未知'}</div>
      </div>

      <div className={styles.card}>
        <div className={styles.cardTitle}>退款信息</div>
        <div className={styles.infoRow}>
          <span className={styles.infoLabel}>退款单号</span>
          <span className={styles.infoValue}>{refund.refundNo}</span>
        </div>
        <div className={styles.infoRow}>
          <span className={styles.infoLabel}>退款类型</span>
          <span className={styles.infoValue}>{REFUND_TYPE[refund.type] || '退款'}</span>
        </div>
        <div className={styles.infoRow}>
          <span className={styles.infoLabel}>退款金额</span>
          <span className={styles.infoValue}><Price value={refund.refundAmount} size="md" /></span>
        </div>
        <div className={styles.infoRow}>
          <span className={styles.infoLabel}>退款原因</span>
          <span className={styles.infoValue}>{refund.reason}</span>
        </div>
        <div className={styles.infoRow}>
          <span className={styles.infoLabel}>申请时间</span>
          <span className={styles.infoValue}>{new Date(refund.ctime).toLocaleString('zh-CN')}</span>
        </div>
      </div>

      <div className={styles.card}>
        <div className={styles.cardTitle}>退款进度</div>
        <div className={styles.timeline}>
          {timeline.map((item, i) => (
            <div key={i} className={styles.timelineItem}>
              <div className={`${styles.timelineDot} ${i === timeline.length - 1 ? styles.timelineDotActive : ''}`} />
              <div className={styles.timelineContent}>
                <div className={styles.timelineTitle}>{item.title}</div>
                <div className={styles.timelineTime}>{item.time}</div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {refund.status === 1 && (
        <div className={styles.footer}>
          <Button
            color="danger"
            fill="outline"
            onClick={() => {
              Dialog.confirm({
                content: '确定取消退款申请？',
                onConfirm: async () => {
                  try {
                    await cancelRefund(refund.refundNo)
                    Toast.show('已取消退款')
                    navigate(-1)
                  } catch (e: unknown) {
                    Toast.show((e as Error).message || '取消失败')
                  }
                },
              })
            }}
          >
            取消退款
          </Button>
        </div>
      )}
    </div>
  )
}
