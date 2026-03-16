import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button, Checkbox, SwipeAction, Dialog, Toast } from 'antd-mobile'
import { useCartStore } from '@/stores/cart'
import Price from '@/components/Price'
import styles from './cart.module.css'

export default function CartPage() {
  const navigate = useNavigate()
  const { items, loading, stockMap, fetchCart, fetchStock, toggleSelect, updateQuantity, remove, clearAll, batchRemoveSelected } = useCartStore()
  const selectedItems = items.filter((i) => i.selected)
  const totalAmount = selectedItems.reduce((sum, i) => sum + i.price * i.quantity, 0)

  useEffect(() => {
    fetchCart().then(() => {
      useCartStore.getState().fetchStock()
    })
  }, [fetchCart])

  const handleCheckout = () => {
    if (selectedItems.length === 0) {
      Toast.show('请选择商品')
      return
    }
    const outOfStockItem = selectedItems.find((i) => {
      const max = stockMap[i.skuId]
      return max !== undefined && i.quantity > max
    })
    if (outOfStockItem) {
      Toast.show(`${outOfStockItem.productName} 库存不足，请调整数量`)
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
          <div>购物车是空的</div>
          <Button
            color="primary"
            fill="outline"
            size="small"
            style={{ marginTop: 16 }}
            onClick={() => navigate('/')}
          >
            去逛逛
          </Button>
        </div>
      ) : (
        items.map((item) => (
          <SwipeAction
            key={item.skuId}
            rightActions={[{
              key: 'delete',
              text: '删除',
              color: 'danger',
              onClick: () => remove(item.skuId),
            }]}
          >
            <div className={styles.item}>
              <Checkbox
                checked={item.selected}
                onChange={(v) => toggleSelect(item.skuId, v)}
              />
              <img className={styles.itemImage} src={item.productImage || 'https://via.placeholder.com/80'} alt='' />
              <div className={styles.itemInfo}>
                <div className={styles.itemName}>{item.productName}</div>
                {item.skuSpec && <div className={styles.itemSpec}>{item.skuSpec}</div>}
                <div className={styles.itemBottom}>
                  <Price value={item.price} size='sm' />
                  <div className={styles.quantityControl}>
                    <span
                      className={`${styles.qtyBtn} ${item.quantity <= 1 ? styles.qtyBtnDisabled : ''}`}
                      onClick={() => item.quantity > 1 && updateQuantity(item.skuId, item.quantity - 1)}
                    >-</span>
                    <span className={styles.qtyValue}>{item.quantity}</span>
                    <span
                      className={`${styles.qtyBtn} ${stockMap[item.skuId] !== undefined && item.quantity >= stockMap[item.skuId] ? styles.qtyBtnDisabled : ''}`}
                      onClick={() => {
                        const max = stockMap[item.skuId]
                        if (max !== undefined && item.quantity >= max) {
                          Toast.show('库存不足')
                          return
                        }
                        updateQuantity(item.skuId, item.quantity + 1)
                      }}
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
