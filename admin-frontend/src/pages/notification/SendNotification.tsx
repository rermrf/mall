import { useNavigate } from 'react-router-dom'
import { ProForm, ProFormText, ProFormDigit, ProFormSelect } from '@ant-design/pro-components'
import { Card, message } from 'antd'
import { sendNotification } from '@/api/notification'

export default function SendNotification() {
  const navigate = useNavigate()

  const handleFinish = async (values: { userId: number; tenantId?: number; templateCode: string; channel: number }) => {
    try {
      await sendNotification(values)
      message.success('发送成功')
      navigate('/notification-templates')
    } catch { /* handled */ }
  }

  return (
    <Card title="发送通知">
      <ProForm onFinish={handleFinish}>
        <ProFormDigit name="userId" label="用户ID" rules={[{ required: true }]} />
        <ProFormDigit name="tenantId" label="租户ID" placeholder="可选" />
        <ProFormText name="templateCode" label="模板编码" rules={[{ required: true }]} />
        <ProFormSelect name="channel" label="渠道" rules={[{ required: true }]} options={[
          { label: '站内信', value: 1 },
          { label: '短信', value: 2 },
          { label: '邮件', value: 3 },
        ]} />
      </ProForm>
    </Card>
  )
}
