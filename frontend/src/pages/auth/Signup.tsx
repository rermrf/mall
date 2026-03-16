import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Input, Button, Toast } from 'antd-mobile'
import { signup } from '@/api/auth'
import styles from './auth.module.css'

export default function SignupPage() {
  const navigate = useNavigate()
  const [phone, setPhone] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  const PHONE_REG = /^1[3-9]\d{9}$/

  const handleSignup = async () => {
    if (!phone || !PHONE_REG.test(phone)) {
      Toast.show('请输入正确的11位手机号')
      return
    }
    if (!password || password.length < 6) {
      Toast.show('密码至少6位')
      return
    }
    setLoading(true)
    try {
      await signup({ phone, email, password })
      Toast.show('注册成功，请登录')
      navigate('/login', { replace: true })
    } catch (e: unknown) {
      Toast.show((e as Error).message || '注册失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.title}>创建账户</div>
      <div className={styles.subtitle}>注册后即可开始购物</div>
      <div className={styles.form}>
        <Input placeholder='手机号' value={phone} onChange={setPhone} type='tel' maxLength={11} clearable />
        <Input placeholder='邮箱（选填）' value={email} onChange={setEmail} type='email' clearable />
        <Input placeholder='密码' value={password} onChange={setPassword} type='password' clearable />
        <Button block color='primary' className={styles.submitBtn} loading={loading} onClick={handleSignup}>
          注册
        </Button>
      </div>
      <div className={styles.footer}>
        已有账户？<Link to='/login' className={styles.link}>去登录</Link>
      </div>
    </div>
  )
}
