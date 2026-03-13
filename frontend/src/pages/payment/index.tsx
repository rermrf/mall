import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, useLocation, useNavigate } from 'react-router-dom'
import { NavBar, Button, Toast, SpinLoading } from 'antd-mobile'
import { CheckCircleFill } from 'antd-mobile-icons'
import { createPayment, getPayment } from '@/api/payment'
import styles from './payment.module.css'

const channels = [
  { key: 'mock', name: '模拟支付', icon: '💳' },
  { key: 'wechat', name: '微信支付', icon: '💚' },
  { key: 'alipay', name: '支付宝', icon: '🔵' },
] as const

export default function PaymentPage() {
  const { orderNo } = useParams<{ orderNo: string }>()
  const location = useLocation()
  const navigate = useNavigate()
  const payAmount = (location.state as { payAmount?: number })?.payAmount || 0

  const [selectedChannel, setSelectedChannel] = useState<string>('mock')
  const [loading, setLoading] = useState(false)
  const [polling, setPolling] = useState(false)
  const [paid, setPaid] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const attemptsRef = useRef(0)

  const stopPolling = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = undefined
    }
  }, [])

  useEffect(() => {
    return stopPolling
  }, [stopPolling])

  const pollPaymentStatus = useCallback((paymentNo: string) => {
    setPolling(true)
    attemptsRef.current = 0

    const poll = async () => {
      attemptsRef.current += 1
      try {
        const payment = await getPayment(paymentNo)
        // status 2 = paid
        if (payment.status >= 2) {
          setPolling(false)
          setPaid(true)
          Toast.show('支付成功')
          return
        }
      } catch {
        // ignore polling errors
      }

      if (attemptsRef.current >= 15) {
        setPolling(false)
        setPaid(true)
        Toast.show('支付处理中，请稍后查看订单')
        return
      }

      timerRef.current = setTimeout(poll, 2000)
    }

    poll()
  }, [])

  const handlePay = async () => {
    if (!orderNo) return
    setLoading(true)
    try {
      const result = await createPayment({
        order_id: 0,
        order_no: orderNo,
        channel: selectedChannel as 'mock' | 'wechat' | 'alipay',
        amount: payAmount,
      })
      setLoading(false)
      pollPaymentStatus(result.payment_no)
    } catch (e: unknown) {
      Toast.show((e as Error).message || '支付失败')
      setLoading(false)
    }
  }

  if (paid) {
    return (
      <div className={styles.successPage}>
        <CheckCircleFill className={styles.successIcon} />
        <div className={styles.successText}>支付成功</div>
        <Button fill='outline' onClick={() => navigate(`/orders/${orderNo}`, { replace: true })}>
          查看订单
        </Button>
        <Button
          fill='none'
          style={{ marginTop: 8, color: 'var(--color-text-secondary)' }}
          onClick={() => navigate('/', { replace: true })}
        >
          返回首页
        </Button>
      </div>
    )
  }

  if (polling) {
    return (
      <div className={styles.successPage}>
        <SpinLoading color='default' style={{ '--size': '48px' }} />
        <div className={styles.successText} style={{ marginTop: 16 }}>支付处理中...</div>
        <div style={{ color: 'var(--color-text-secondary)', fontSize: 14 }}>请稍候，正在确认支付结果</div>
      </div>
    )
  }

  return (
    <div className={styles.page}>
      <NavBar onBack={() => navigate(-1)}>收银台</NavBar>
      <div className={styles.content}>
        <div className={styles.amountLabel}>支付金额</div>
        <div className={styles.amount}>¥{(payAmount / 100).toFixed(2)}</div>
      </div>

      <div className={styles.channels}>
        {channels.map((ch) => (
          <div
            key={ch.key}
            className={`${styles.channelItem} ${selectedChannel === ch.key ? styles.channelItemSelected : ''}`}
            onClick={() => setSelectedChannel(ch.key)}
          >
            <span className={styles.channelIcon}>{ch.icon}</span>
            <span className={styles.channelName}>{ch.name}</span>
          </div>
        ))}
      </div>

      <Button
        block
        color='primary'
        className={styles.payBtn}
        loading={loading}
        onClick={handlePay}
      >
        立即支付
      </Button>
    </div>
  )
}
