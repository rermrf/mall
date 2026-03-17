import { useState, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { NavBar, Button, TextArea, Toast, Popup } from 'antd-mobile'
import { useShallow } from 'zustand/react/shallow'
import { useCartStore } from '@/stores/cart'
import { listAddresses, type Address } from '@/api/user'
import { createOrder } from '@/api/order'
import { listMyCoupons, type UserCoupon } from '@/api/marketing'
import Price from '@/components/Price'
import styles from './confirm.module.css'

export default function OrderConfirm() {
  const navigate = useNavigate()
  const location = useLocation()
  const locationState = location.state as {
    directBuy?: boolean
    product?: { skuId: number; id: number; name: string; mainImage: string; price: number }
    quantity?: number
  } | null

  const directBuy = locationState?.directBuy
  const directItems = directBuy && locationState?.product
    ? [{
        skuId: locationState.product.skuId,
        productId: locationState.product.id,
        productName: locationState.product.name,
        productImage: locationState.product.mainImage,
        price: locationState.product.price,
        quantity: locationState.quantity || 1,
        selected: true,
      }]
    : null

  const selectedItems = useCartStore(useShallow((s) => s.items.filter((i) => i.selected)))
  const fetchCart = useCartStore((s) => s.fetchCart)

  const orderItems = directItems || selectedItems
  const orderTotal = orderItems.reduce((sum: number, i: any) => sum + i.price * i.quantity, 0)

  const [addresses, setAddresses] = useState<Address[]>([])
  const [selectedAddress, setSelectedAddress] = useState<Address | null>(null)
  const [remark, setRemark] = useState('')
  const [loading, setLoading] = useState(false)
  const [coupons, setCoupons] = useState<UserCoupon[]>([])
  const [selectedCoupon, setSelectedCoupon] = useState<UserCoupon | null>(null)
  const [showCouponPopup, setShowCouponPopup] = useState(false)

  useEffect(() => {
    if (!directBuy && selectedItems.length === 0) {
      navigate('/cart', { replace: true })
      return
    }
    listAddresses().then((list) => {
      setAddresses(list || [])
      const defaultAddr = (list || []).find((a) => a.isDefault) || (list || [])[0]
      if (defaultAddr) setSelectedAddress(defaultAddr)
    }).catch(() => {})
    listMyCoupons(1).then((list) => setCoupons(list || [])).catch(() => {})
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const usableCoupons = coupons.filter((c) => c.minSpend <= orderTotal)

  const discountAmount = selectedCoupon
    ? (selectedCoupon.type === 1 ? selectedCoupon.value : Math.round(orderTotal * (1 - selectedCoupon.value / 100)))
    : 0
  const payAmount = Math.max(orderTotal - discountAmount, 0)

  const handleSubmit = async () => {
    if (!selectedAddress) {
      Toast.show('请选择收货地址')
      return
    }
    setLoading(true)
    try {
      const result = await createOrder({
        items: orderItems.map((i: any) => ({ skuId: i.skuId, quantity: i.quantity })),
        addressId: selectedAddress.id,
        couponId: selectedCoupon?.couponId,
        remark,
      })
      if (!directBuy) await fetchCart()
      navigate(`/payment/${result.orderNo}`, { replace: true, state: { payAmount: result.payAmount } })
    } catch (e: unknown) {
      Toast.show((e as Error).message || '下单失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>确认订单</NavBar>
      </div>

      <div className={styles.addressCard}>
        {selectedAddress ? (
          <>
            <div>
              <span className={styles.addressName}>{selectedAddress.name}</span>
              <span className={styles.addressPhone}>{selectedAddress.phone}</span>
            </div>
            <div className={styles.addressDetail}>
              {selectedAddress.province}{selectedAddress.city}{selectedAddress.district}{selectedAddress.detail}
            </div>
          </>
        ) : (
          <div className={styles.noAddress}>
            {addresses.length === 0 ? (
              <>
                <span>请先添加收货地址</span>
                <Button
                  size="mini"
                  color="primary"
                  style={{ marginLeft: 8 }}
                  onClick={() => navigate('/me/addresses/edit', { state: { from: 'order-confirm' } })}
                >
                  新增地址
                </Button>
              </>
            ) : '请选择收货地址'}
          </div>
        )}
      </div>

      <div className={styles.itemsCard}>
        {orderItems.map((item: any) => (
          <div key={item.skuId} className={styles.orderItem}>
            <img className={styles.orderItemImage} src={item.productImage || 'https://via.placeholder.com/60'} alt='' />
            <div className={styles.orderItemInfo}>
              <div className={styles.orderItemName}>{item.productName}</div>
              <div className={styles.orderItemMeta}>
                <Price value={item.price} size='sm' />
                <span>x{item.quantity}</span>
              </div>
            </div>
          </div>
        ))}
      </div>

      <div className={styles.couponCard} onClick={() => setShowCouponPopup(true)}>
        <span className={styles.couponLabel}>优惠券</span>
        <span>
          {selectedCoupon ? (
            <span className={styles.couponSelected}>
              -{selectedCoupon.type === 1 ? `¥${(selectedCoupon.value / 100).toFixed(0)}` : `${selectedCoupon.value / 10}折`}
            </span>
          ) : (
            <span style={{ color: usableCoupons.length > 0 ? 'var(--color-accent)' : 'var(--color-text-secondary)' }}>
              {usableCoupons.length > 0 ? `${usableCoupons.length}张可用` : '无可用券'}
            </span>
          )}
          <span className={styles.couponArrow}> ›</span>
        </span>
      </div>

      <div className={styles.remarkInput}>
        <TextArea placeholder='订单备注（选填）' value={remark} onChange={setRemark} maxLength={200} rows={2} />
      </div>

      <div className={styles.footer}>
        <div className={styles.footerTotal}>
          合计: <Price value={payAmount} size='md' />
        </div>
        <Button color='primary' className={styles.submitBtn} loading={loading} onClick={handleSubmit}>
          提交订单
        </Button>
      </div>

      <Popup
        visible={showCouponPopup}
        onMaskClick={() => setShowCouponPopup(false)}
        bodyStyle={{ borderTopLeftRadius: 12, borderTopRightRadius: 12 }}
      >
        <div className={styles.popupContent}>
          <div className={styles.popupTitle}>选择优惠券</div>
          <div
            className={`${styles.popupNoCoupon} ${!selectedCoupon ? styles.popupNoCouponActive : ''}`}
            onClick={() => { setSelectedCoupon(null); setShowCouponPopup(false) }}
          >
            不使用优惠券
          </div>
          {usableCoupons.map((c) => (
            <div
              key={c.id}
              className={`${styles.popupCouponItem} ${selectedCoupon?.id === c.id ? styles.popupCouponItemActive : ''}`}
              onClick={() => { setSelectedCoupon(c); setShowCouponPopup(false) }}
            >
              <span className={styles.popupCouponValue}>
                {c.type === 1 ? `¥${(c.value / 100).toFixed(0)}` : `${c.value / 10}折`}
              </span>
              <div className={styles.popupCouponInfo}>
                <div className={styles.popupCouponName}>{c.name}</div>
                <div className={styles.popupCouponCondition}>满{(c.minSpend / 100).toFixed(0)}可用</div>
              </div>
            </div>
          ))}
        </div>
      </Popup>
    </div>
  )
}
