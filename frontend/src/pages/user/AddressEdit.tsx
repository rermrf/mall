import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { NavBar, Input, Switch, Button, Toast } from 'antd-mobile'
import { createAddress, updateAddress, type Address } from '@/api/user'
import styles from './addressEdit.module.css'

export default function AddressEditPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const existing = (location.state as { address?: Address })?.address

  const [name, setName] = useState(existing?.name || '')
  const [phone, setPhone] = useState(existing?.phone || '')
  const [province, setProvince] = useState(existing?.province || '')
  const [city, setCity] = useState(existing?.city || '')
  const [district, setDistrict] = useState(existing?.district || '')
  const [detail, setDetail] = useState(existing?.detail || '')
  const [isDefault, setIsDefault] = useState(existing?.is_default || false)
  const [loading, setLoading] = useState(false)

  const handleSave = async () => {
    if (!name.trim()) { Toast.show('请输入姓名'); return }
    if (!phone.trim()) { Toast.show('请输入手机号'); return }
    if (!province.trim() || !city.trim()) { Toast.show('请输入省市'); return }
    if (!detail.trim()) { Toast.show('请输入详细地址'); return }

    const params = {
      name: name.trim(),
      phone: phone.trim(),
      province: province.trim(),
      city: city.trim(),
      district: district.trim(),
      detail: detail.trim(),
      is_default: isDefault,
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

        <div className={styles.row}>
          <div className={styles.field}>
            <div className={styles.label}>省份</div>
            <Input placeholder="省" value={province} onChange={setProvince} />
          </div>
          <div className={styles.field}>
            <div className={styles.label}>城市</div>
            <Input placeholder="市" value={city} onChange={setCity} />
          </div>
          <div className={styles.field}>
            <div className={styles.label}>区县</div>
            <Input placeholder="区" value={district} onChange={setDistrict} />
          </div>
        </div>

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
