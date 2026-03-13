import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Button, SwipeAction, Dialog, Toast } from 'antd-mobile'
import { listAddresses, deleteAddress, type Address } from '@/api/user'
import styles from './addresses.module.css'

export default function AddressesPage() {
  const navigate = useNavigate()
  const [addresses, setAddresses] = useState<Address[]>([])
  const [loading, setLoading] = useState(true)

  const fetchAddresses = useCallback(async () => {
    try {
      const list = await listAddresses()
      setAddresses(list || [])
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchAddresses()
  }, [fetchAddresses])

  const handleDelete = async (id: number) => {
    const confirmed = await Dialog.confirm({ content: '确定删除该地址？' })
    if (!confirmed) return
    try {
      await deleteAddress(id)
      Toast.show('已删除')
      setAddresses((prev) => prev.filter((a) => a.id !== id))
    } catch (e: unknown) {
      Toast.show((e as Error).message || '删除失败')
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>收货地址</NavBar>
      </div>

      {!loading && addresses.length === 0 && (
        <div className={styles.empty}>暂无收货地址</div>
      )}

      {addresses.map((addr) => (
        <SwipeAction
          key={addr.id}
          rightActions={[
            {
              key: 'delete',
              text: '删除',
              color: 'danger',
              onClick: () => handleDelete(addr.id),
            },
          ]}
        >
          <div className={styles.addressCard}>
            <div className={styles.cardHeader}>
              <span className={styles.name}>{addr.name}</span>
              <span className={styles.phone}>{addr.phone}</span>
              {addr.is_default && <span className={styles.defaultTag}>默认</span>}
            </div>
            <div className={styles.address}>
              {addr.province}{addr.city}{addr.district}{addr.detail}
            </div>
            <div
              className={styles.cardAction}
              onClick={() => navigate('/me/addresses/edit', { state: { address: addr } })}
            >
              编辑
            </div>
          </div>
        </SwipeAction>
      ))}

      <div className={styles.footer}>
        <Button
          color="primary"
          className={styles.addBtn}
          onClick={() => navigate('/me/addresses/edit')}
        >
          新增地址
        </Button>
      </div>
    </div>
  )
}
