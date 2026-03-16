import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { NavBar, Input, Switch, Button, Toast, CascadePicker } from 'antd-mobile'
import { createAddress, updateAddress, type Address } from '@/api/user'
import { regionData } from '@/data/regions'
import styles from './addressEdit.module.css'

const PHONE_REG = /^1[3-9]\d{9}$/

export default function AddressEditPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const existing = (location.state as { address?: Address })?.address

  const [name, setName] = useState(existing?.name || '')
  const [phone, setPhone] = useState(existing?.phone || '')
  const [region, setRegion] = useState<string[]>(
    existing ? [existing.province, existing.city, existing.district].filter(Boolean) : []
  )
  const [regionVisible, setRegionVisible] = useState(false)
  const [detail, setDetail] = useState(existing?.detail || '')
  const [isDefault, setIsDefault] = useState(existing?.isDefault || false)
  const [loading, setLoading] = useState(false)

  const regionText = region.length >= 2 ? region.join(' ') : ''

  const handleSave = async () => {
    if (!name.trim()) { Toast.show('请输入姓名'); return }
    if (!phone.trim() || !PHONE_REG.test(phone.trim())) {
      Toast.show('请输入正确的11位手机号')
      return
    }
    if (region.length < 2) { Toast.show('请选择省市区'); return }
    if (!detail.trim()) { Toast.show('请输入详细地址'); return }

    const params = {
      name: name.trim(),
      phone: phone.trim(),
      province: region[0],
      city: region[1],
      district: region[2] || '',
      detail: detail.trim(),
      isDefault,
    }

    setLoading(true)
    try {
      if (existing) {
        await updateAddress(existing.id, params)
      } else {
        await createAddress(params)
      }
      Toast.show('保存成功')
      navigate(-1)
    } catch (e: unknown) {
      Toast.show((e as Error).message || '保存失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.navBar}>
        <NavBar onBack={() => navigate(-1)}>{existing ? '编辑地址' : '新增地址'}</NavBar>
      </div>

      <div className={styles.card}>
        <div className={styles.field}>
          <div className={styles.label}>姓名</div>
          <Input placeholder="收货人姓名" value={name} onChange={setName} clearable />
        </div>

        <div className={styles.field}>
          <div className={styles.label}>手机号</div>
          <Input placeholder="收货人手机号" type="tel" value={phone} onChange={setPhone} clearable maxLength={11} />
        </div>

        <div className={styles.field} onClick={() => setRegionVisible(true)}>
          <div className={styles.label}>所在地区</div>
          <div className={styles.regionValue}>
            {regionText || <span style={{ color: 'var(--color-text-secondary)' }}>请选择省/市/区</span>}
          </div>
        </div>

        <CascadePicker
          title="选择地区"
          options={regionData}
          visible={regionVisible}
          onClose={() => setRegionVisible(false)}
          onConfirm={(val) => {
            setRegion(val as string[])
            setRegionVisible(false)
          }}
          value={region}
        />

        <div className={styles.field}>
          <div className={styles.label}>详细地址</div>
          <Input placeholder="街道、门牌号等" value={detail} onChange={setDetail} clearable />
        </div>

        <div className={styles.defaultRow}>
          <span className={styles.defaultLabel}>设为默认地址</span>
          <Switch checked={isDefault} onChange={setIsDefault} />
        </div>

        <Button color="primary" className={styles.saveBtn} loading={loading} onClick={handleSave}>
          保存
        </Button>
      </div>
    </div>
  )
}
