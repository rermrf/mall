import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Form, Input, Button, Card, message } from 'antd'
import { ShopOutlined, PhoneOutlined, LockOutlined, ShoppingOutlined } from '@ant-design/icons'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

export default function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const setLoggedIn = useAuthStore((s) => s.setLoggedIn)
  const [loading, setLoading] = useState(false)
  const redirect = searchParams.get('redirect') || '/'

  const onFinish = async (values: { phone: string; password: string; tenantId?: number }) => {
    setLoading(true)
    try {
      await login({ ...values, tenantId: Number(values.tenantId) || 1 })
      setLoggedIn(true)
      message.success('登录成功')
      navigate(redirect, { replace: true })
    } catch (e: unknown) {
      message.error((e as Error).message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    }}>
      <Card style={{ width: 400, borderRadius: 8 }} bordered={false}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <ShopOutlined style={{ fontSize: 48, color: '#1890ff' }} />
          <h2 style={{ marginTop: 16, marginBottom: 4 }}>商家管理后台</h2>
          <p style={{ color: '#999' }}>登录你的商家账户</p>
        </div>
        <Form onFinish={onFinish} size="large" initialValues={{ tenantId: 1 }}>
          <Form.Item name="tenantId" rules={[{ required: true, message: '请输入商户ID' }]}>
            <Input prefix={<ShoppingOutlined />} placeholder="商户ID" type="number" />
          </Form.Item>
          <Form.Item name="phone" rules={[{ required: true, message: '请输入手机号' }]}>
            <Input prefix={<PhoneOutlined />} placeholder="手机号" maxLength={11} />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={loading}>
              登录
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}
