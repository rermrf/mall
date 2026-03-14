import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormSelect, ProFormDateTimePicker, ProFormList, ProFormDigit } from '@ant-design/pro-components'
import { createSeckill, updateSeckill, getSeckill } from '@/api/marketing'
import type { CreateSeckillReq, SeckillItem } from '@/types/marketing'

export default function SeckillForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id
  const [initialValues, setInitialValues] = useState<Record<string, unknown>>()

  useEffect(() => {
    if (isEdit) {
      getSeckill(Number(id)).then((activity) => {
        if (activity) {
          setInitialValues({
            name: activity.name,
            start_time: activity.start_time,
            end_time: activity.end_time,
            status: activity.status,
            items: activity.items ?? [],
          })
        }
      }).catch(() => {})
    }
  }, [id, isEdit])

  if (isEdit && !initialValues) {
    return <Card title="编辑秒杀活动" loading />
  }

  return (
    <Card title={isEdit ? '编辑秒杀活动' : '创建秒杀活动'}>
      <ProForm
        initialValues={initialValues}
        onFinish={async (values: Record<string, unknown>) => {
          const rawItems = (values.items as Array<Record<string, unknown>>) ?? []
          const items: SeckillItem[] = rawItems.map((item) => ({
            sku_id: Number(item.sku_id) || 0,
            seckill_price: Number(item.seckill_price) || 0,
            seckill_stock: Number(item.seckill_stock) || 0,
            per_limit: Number(item.per_limit) || 1,
          }))

          const data: CreateSeckillReq = {
            name: values.name as string,
            start_time: values.start_time as string,
            end_time: values.end_time as string,
            status: values.status as number,
            items,
          }
          try {
            if (isEdit) {
              await updateSeckill(Number(id), data)
              message.success('更新成功')
            } else {
              await createSeckill(data)
              message.success('创建成功')
            }
            navigate('/marketing/seckill')
          } catch (e: unknown) {
            message.error((e as Error).message)
          }
        }}
      >
        <ProFormText name="name" label="活动名称" rules={[{ required: true }]} />
        <ProFormDateTimePicker name="start_time" label="开始时间" rules={[{ required: true }]} />
        <ProFormDateTimePicker name="end_time" label="结束时间" rules={[{ required: true }]} />
        <ProFormSelect name="status" label="状态" initialValue={0} options={[
          { label: '未开始', value: 0 },
          { label: '进行中', value: 1 },
        ]} />

        <ProFormList
          name="items"
          label="秒杀商品"
          creatorButtonProps={{ creatorButtonText: '添加秒杀商品' }}
          min={1}
          itemRender={({ listDom, action }) => (
            <div style={{ display: 'flex', alignItems: 'flex-end', gap: 8, marginBottom: 8 }}>
              {listDom}
              {action}
            </div>
          )}
        >
          <ProForm.Group key="seckill-item-group">
            <ProFormDigit name="sku_id" label="SKU ID" rules={[{ required: true }]} min={1} />
            <ProFormDigit name="seckill_price" label="秒杀价（分）" rules={[{ required: true }]} min={0} />
            <ProFormDigit name="seckill_stock" label="秒杀库存" rules={[{ required: true }]} min={1} />
            <ProFormDigit name="per_limit" label="每人限购" initialValue={1} min={1} />
          </ProForm.Group>
        </ProFormList>
      </ProForm>
    </Card>
  )
}
