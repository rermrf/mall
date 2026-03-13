import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormDigit, ProFormSelect } from '@ant-design/pro-components'
import { createFreightTemplate, updateFreightTemplate, getFreightTemplate } from '@/api/logistics'
import type { CreateFreightTemplateReq } from '@/types/logistics'

export default function TemplateForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id
  const [initialValues, setInitialValues] = useState<Record<string, unknown>>({})

  useEffect(() => {
    if (isEdit) {
      getFreightTemplate(Number(id)).then((t) => {
        if (t) setInitialValues({ name: t.name, charge_type: t.charge_type, free_threshold: t.free_threshold })
      }).catch(() => {})
    }
  }, [id, isEdit])

  return (
    <Card title={isEdit ? '编辑运费模板' : '创建运费模板'}>
      <ProForm<CreateFreightTemplateReq>
        initialValues={initialValues}
        onFinish={async (values) => {
          const data = { ...values, rules: [] }
          try {
            if (isEdit) {
              await updateFreightTemplate(Number(id), data)
              message.success('更新成功')
            } else {
              await createFreightTemplate(data)
              message.success('创建成功')
            }
            navigate('/logistics/template')
          } catch (e: unknown) {
            message.error((e as Error).message)
          }
        }}
      >
        <ProFormText name="name" label="模板名称" rules={[{ required: true }]} />
        <ProFormSelect name="charge_type" label="计费方式" rules={[{ required: true }]} options={[
          { label: '按重量', value: 1 },
          { label: '按件数', value: 2 },
        ]} />
        <ProFormDigit name="free_threshold" label="免邮门槛（分）" initialValue={0} min={0} />
      </ProForm>
    </Card>
  )
}
