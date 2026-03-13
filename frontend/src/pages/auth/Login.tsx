import { useState, useRef, useCallback, useEffect } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { Input, Button, Toast } from 'antd-mobile'
import { login, loginByPhone, sendSmsCode } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import styles from './auth.module.css'

type LoginMode = 'password' | 'sms'

function useSmsCountdown() {
  const [countdown, setCountdown] = useState(0)
  const timerRef = useRef<ReturnType<typeof setInterval>>(undefined)

  useEffect(() => {
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [])

  const start = useCallback(() => {
    setCountdown(60)
    timerRef.current = setInterval(() => {
      setCountdown((v) => {
        if (v <= 1) {
          if (timerRef.current) clearInterval(timerRef.current)
          return 0
        }
        return v - 1
      })
    }, 1000)
  }, [])

  return { countdown, start }
}

export default function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const setLoggedIn = useAuthStore((s) => s.setLoggedIn)

  const [mode, setMode] = useState<LoginMode>('password')
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [smsCode, setSmsCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [sendingCode, setSendingCode] = useState(false)
  const { countdown, start: startCountdown } = useSmsCountdown()

  const redirect = searchParams.get('redirect') || '/'

  const handlePasswordLogin = async () => {
    if (!phone || !password) {
      Toast.show('请输入手机号和密码')
      return
    }
    setLoading(true)
    try {
      await login({ phone, password })
      setLoggedIn(true)
      navigate(redirect, { replace: true })
    } catch (e: unknown) {
      Toast.show((e as Error).message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  const handleSendCode = async () => {
    if (!phone || phone.length < 11) {
      Toast.show('请输入正确的手机号')
      return
    }
    setSendingCode(true)
    try {
      await sendSmsCode(phone, 1)
      Toast.show('验证码已发送')
      startCountdown()
    } catch (e: unknown) {
      Toast.show((e as Error).message || '发送失败')
    } finally {
      setSendingCode(false)
    }
  }

  const handleSmsLogin = async () => {
    if (!phone || !smsCode) {
      Toast.show('请输入手机号和验证码')
      return
    }
    setLoading(true)
    try {
      await loginByPhone({ phone, code: smsCode })
      setLoggedIn(true)
      navigate(redirect, { replace: true })
    } catch (e: unknown) {
      Toast.show((e as Error).message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  const handleOAuth = (provider: string) => {
    Toast.show(`${provider}登录暂未开放`)
  }

  return (
    <div className={styles.page}>
      <div className={styles.title}>欢迎回来</div>
      <div className={styles.subtitle}>登录你的账户继续购物</div>

      <div className={styles.tabs}>
        <span
          className={`${styles.tab} ${mode === 'password' ? styles.tabActive : ''}`}
          onClick={() => setMode('password')}
        >
          密码登录
        </span>
        <span
          className={`${styles.tab} ${mode === 'sms' ? styles.tabActive : ''}`}
          onClick={() => setMode('sms')}
        >
          验证码登录
        </span>
      </div>

      <div className={styles.form}>
        <Input placeholder='手机号' value={phone} onChange={setPhone} type='tel' maxLength={11} clearable />

        {mode === 'password' ? (
          <Input placeholder='密码' value={password} onChange={setPassword} type='password' clearable />
        ) : (
          <div className={styles.codeRow}>
            <Input
              placeholder='验证码'
              value={smsCode}
              onChange={setSmsCode}
              type='number'
              maxLength={6}
              clearable
              className={styles.codeInput}
            />
            <Button
              size='small'
              className={styles.codeBtn}
              disabled={countdown > 0}
              loading={sendingCode}
              onClick={handleSendCode}
            >
              {countdown > 0 ? `${countdown}s` : '发送验证码'}
            </Button>
          </div>
        )}

        <Button
          block
          color='primary'
          className={styles.submitBtn}
          loading={loading}
          onClick={mode === 'password' ? handlePasswordLogin : handleSmsLogin}
        >
          登录
        </Button>
      </div>

      <div className={styles.divider}>
        <span className={styles.dividerLine} />
        <span className={styles.dividerText}>其他登录方式</span>
        <span className={styles.dividerLine} />
      </div>

      <div className={styles.oauthRow}>
        <div className={styles.oauthBtn} onClick={() => handleOAuth('微信')}>
          <span className={styles.oauthIcon}>💚</span>
          <span className={styles.oauthLabel}>微信</span>
        </div>
        <div className={styles.oauthBtn} onClick={() => handleOAuth('支付宝')}>
          <span className={styles.oauthIcon}>🔵</span>
          <span className={styles.oauthLabel}>支付宝</span>
        </div>
      </div>

      <div className={styles.footer}>
        还没有账户？<Link to='/signup' className={styles.link}>立即注册</Link>
      </div>
    </div>
  )
}
