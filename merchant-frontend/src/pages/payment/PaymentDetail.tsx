import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Descriptions, Tag, Button, Space, Modal, InputNumber, Input, message, Spin } from 'antd'
import { getPayment, refundPayment } from '@/api/payment'
import { PAYMENT_STATUS_MAP, PAYMENT_STATUS, formatPrice } from '@/constants'
import { silentApiError } from '@/utils/error'
import type { Payment } from '@/types/payment'

export default function PaymentDetail() {
  const { paymentNo } = useParams<{ paymentNo: string }>()
  const navigate = useNavigate()
  const [payment, setPayment] = useState<Payment | null>(null)
  const [loading, setLoading] = useState(true)
  const [refundModal, setRefundModal] = useState(false)
  const [refundAmount, setRefundAmount] = useState<number | null>(null)
  const [refundReason, setRefundReason] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (paymentNo) {
      setLoading(true)
      getPayment(paymentNo)
        .then(setPayment)
        .catch(silentApiError('paymentDetail:getPayment'))
        .finally(() => setLoading(false))
    }
  }, [paymentNo])

  const handleRefund = async () => {
    if (!paymentNo || !refundAmount) {
      message.warning('请填写退款金额')
      return
    }
    setSubmitting(true)
    try {
      const res = await refundPayment(paymentNo, { amount: refundAmount, reason: refundReason })
      message.success(`退款申请成功，退款单号：${res?.refund_no}`)
      setRefundModal(false)
      setRefundAmount(null)
      setRefundReason('')
      getPayment(paymentNo).then(setPayment).catch(silentApiError('paymentDetail:refreshPayment'))
    } catch (e: unknown) {
      message.error((e as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return <div style={{ display: 'flex', justifyContent: 'center', padding: '20vh 0' }}><Spin size="large" /></div>
  }

  if (!payment) return null

  const status = PAYMENT_STATUS_MAP[payment.status] ?? { text: '未知', color: 'default' }

  return (
    <div>
      <Card
        title="支付详情"
        extra={
          <Space>
            {payment.status === PAYMENT_STATUS.PAID && (
              <Button type="primary" danger onClick={() => setRefundModal(true)}>退款</Button>
            )}
            <Button onClick={() => navigate(-1)}>返回</Button>
          </Space>
        }
      >
        <Descriptions column={2}>
          <Descriptions.Item label="支付单号">{payment.payment_no}</Descriptions.Item>
          <Descriptions.Item label="订单号">{payment.order_no}</Descriptions.Item>
          <Descriptions.Item label="金额">{formatPrice(payment.amount)}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={status.color}>{status.text}</Tag></Descriptions.Item>
          <Descriptions.Item label="支付渠道">{payment.channel}</Descriptions.Item>
          <Descriptions.Item label="创建时间">{payment.created_at}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Modal
        title="申请退款"
        open={refundModal}
        onOk={handleRefund}
        confirmLoading={submitting}
        onCancel={() => setRefundModal(false)}
      >
        <div style={{ marginBottom: 12 }}>
          <div style={{ marginBottom: 4 }}>退款金额（分）</div>
          <InputNumber
            style={{ width: '100%' }}
            min={1}
            max={payment.amount}
            value={refundAmount}
            onChange={(v) => setRefundAmount(v)}
            placeholder="请输入退款金额（单位：分）"
          />
        </div>
        <div>
          <div style={{ marginBottom: 4 }}>退款原因</div>
          <Input
            value={refundReason}
            onChange={(e) => setRefundReason(e.target.value)}
            placeholder="请输入退款原因"
          />
        </div>
      </Modal>
    </div>
  )
}
