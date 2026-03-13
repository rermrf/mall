import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { NavBar, Input, Button, Toast } from 'antd-mobile'
import { getProfile, updateProfile } from '@/api/user'
import styles from './profile.module.css'

export default function ProfilePage() {
  const navigate = useNavigate()
  const [nickname, setNickname] = useState('')
  const [avatar, setAvatar] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    getProfile().then((p) => {
      setNickname(p.nickname || '')
      setAvatar(p.avatar || '')
    }).catch(() => {})
  }, [])

  const handleSave = async () => {
    if (!nickname.trim()) {
      Toast.show('请输入昵称')
      return
    }
    setLoading(true)
    try {
      await updateProfile({ nickname: nickname.trim(), avatar: avatar.trim() })
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
        <NavBar onBack={() => navigate(-1)}>编辑资料</NavBar>
      </div>

      <div className={styles.card}>
        <div className={styles.avatarRow}>
          {avatar ? (
            <img className={styles.avatarPreview} src={avatar} alt="" />
          ) : (
            <div className={styles.avatarPlaceholder}>👤</div>
          )}
          <div className={styles.avatarInput}>
            <Input placeholder="头像URL" value={avatar} onChange={setAvatar} clearable />
          </div>
        </div>

        <div className={styles.field}>
          <div className={styles.label}>昵称</div>
          <Input placeholder="请输入昵称" value={nickname} onChange={setNickname} clearable maxLength={20} />
        </div>

        <Button color="primary" className={styles.saveBtn} loading={loading} onClick={handleSave}>
          保存
        </Button>
      </div>
    </div>
  )
}
