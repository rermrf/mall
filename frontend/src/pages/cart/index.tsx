import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button, Checkbox, SwipeAction, Dialog, Toast } from 'antd-mobile'
import { useCartStore } from '@/stores/cart'
import Price from '@/components/Price'
import styles from './cart.module.css'

export default function CartPage() {
  const navigate = useNavigate()
  const { items, loading, fetchCart, toggleSelect, updateQuantity, remove, clearAll, batchRemoveSelected } = useCartStore()
  const selectedItems = items.filter((i) => i.selected)
  const totalAmount = selectedItems.reduce((sum, i) => sum + i.price * i.quantity, 0)

  useEffect(() => {
    fetchCart()
  }, [fetchCart])

  const handleCheckout = () => {
    if (selectedItems.length === 0) {
      Toast.show('请选择商品')
      return
    }
    navigate('/order/confirm')
  }

  const handleClear = () => {
    Dialog.confirm({
      content: '确定清空购物车？',
      onConfirm: async () => {
        try {
          await clearAll()
          Toast.show('已清空')
        } catch (e: unknown) {
          Toast.show((e as Error).message || '操作失败')
        }
      },
    })
  }

  const handleBatchRemove = () => {
    if (selectedItems.length === 0) {
      Toast.show('请先选择商品')
      return
    }
    Dialog.confirm({
      content: `删除选中的 ${selectedItems.length} 件商品？`,
      onConfirm: async () => {
        try {
          await batchRemoveSelected()
          Toast.show('已删除')
        } catch (e: unknown) {
          Toast.show((e as Error).message || '操作失败')
        }
      },
    })
  }

  if (loading && items.length === 0) {
    return <div style={{ textAlign: 'center', padding: '40vh 0', color: 'var(--color-text-secondary)' }}>加载中...</div>
  }

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <span className={styles.title}>购物车</span>
        {items.length > 0 && (
          <div className={styles.headerActions}>
            {selectedItems.length > 0 && (
              <span className={styles.headerAction} onClick={handleBatchRemove}>删除选中</span>
            )}
            <span className={styles.headerAction} onClick={handleClear}>清空</span>
          </div>
        )}
      </div>

      {items.length === 0 ? (
        <div style={{ textAlign: 'center', padding: 60, color: 'var(--color-text-secondary)' }}>
          购物车是空的
        </div>
      ) : (
        items.map((item) => (
          <SwipeAction
            key={item.sku_id}
            rightActions={[{
              key: 'delete',
              text: '删除',
              color: 'danger',
              onClick: () => remove(item.sku_id),
            }]}
          >
            <div className={styles.item}>
              <Checkbox
                checked={item.selected}
                onChange={(v) => toggleSelect(item.sku_id, v)}
              />
              <img className={styles.itemImage} src={item.product_image || 'https://via.placeholder.com/80'} alt='' />
              <div className={styles.itemInfo}>
                <div className={styles.itemName}>{item.product_name}</div>
                {item.sku_spec && <div className={styles.itemSpec}>{item.sku_spec}</div>}
                <div className={styles.itemBottom}>
                  <Price value={item.price} size='sm' />
                  <div className={styles.quantityControl}>
                    <span
                      className={styles.qtyBtn}
                      onClick={() => item.quantity > 1 && updateQuantity(item.sku_id, item.quantity - 1)}
                    >-</span>
                    <span className={styles.qtyValue}>{item.quantity}</span>
                    <span
                      className={styles.qtyBtn}
                      onClick={() => updateQuantity(item.sku_id, item.quantity + 1)}
                    >+</span>
                  </div>
                </div>
              </div>
            </div>
          </SwipeAction>
        ))
      )}

      {items.length > 0 && (
        <div className={styles.footer}>
          <div className={styles.footerTotal}>
            合计: <Price value={totalAmount} size='md' />
          </div>
          <Button
            color='primary'
            className={styles.checkoutBtn}
            onClick={handleCheckout}
          >
            结算({selectedItems.length})
          </Button>
        </div>
      )}
    </div>
  )
}
