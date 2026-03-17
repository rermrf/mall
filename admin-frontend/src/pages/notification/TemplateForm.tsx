import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ProForm, ProFormText, ProFormSelect, ProFormTextArea, ProFormDigit } from '@ant-design/pro-components'
import { Card, message, Spin } from 'antd'
import { createTemplate, updateTemplate, listTemplates } from '@/api/notification'

export default function TemplateForm() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isEdit = !!id
  const [loading, setLoading] = useState(isEdit)
  const [initialValues, setInitialValues] = useState<Record<string, unknown>>({})

  useEffect(() => {
    if (!isEdit) return
    listTemplates({}).then((templates) => {
      const tpl = templates?.find((t) => t.id === Number(id))
      if (tpl) setInitialValues({ tenantId: tpl.tenantId, code: tpl.code, channel: tpl.channel, title: tpl.title, content: tpl.content, status: tpl.status })
    }).finally(() => setLoading(false))
  }, [id, isEdit])

  if (loading) return <Spin style={{ display: 'block', margin: '20vh auto' }} size="large" />

  const handleFinish = async (values: { tenantId?: number; code: string; channel: number; title: string; content: string; status?: number }) => {
    try {
      if (isEdit) {
        await updateTemplate(Number(id), values)
        message.success('更新成功')
      } else {
        await createTemplate(values)
        message.success('创建成功')
      }
      navigate('/notification-templates')
    } catch { /* handled */ }
  }

  return (
    <Card title={isEdit ? '编辑通知模板' : '新建通知模板'}>
      <ProForm onFinish={handleFinish} initialValues={initialValues}>
        <ProFormDigit name="tenantId" label="租户ID" placeholder="留空为平台级模板" />
        <ProFormText name="code" label="模板编码" rules={[{ required: true }]} />
        <ProFormSelect name="channel" label="渠道" rules={[{ required: true }]} options={[
          { label: '站内信', value: 1 },
          { label: '短信', value: 2 },
          { label: '邮件', value: 3 },
        ]} />
        <ProFormText name="title" label="标题" rules={[{ required: true }]} />
        <ProFormTextArea name="content" label="内容" rules={[{ required: true }]} fieldProps={{ rows: 6 }} />
      </ProForm>
    </Card>
  )
}
