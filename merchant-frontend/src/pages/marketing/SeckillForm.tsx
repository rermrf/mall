import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormSelect, ProFormDateTimePicker, ProFormList, ProFormDigit } from '@ant-design/pro-components'
import { createSeckill, updateSeckill, getSeckill } from '@/api/marketing'
import type { CreateSeckillReq, SeckillItem } from '@/types/marketing'
import { silentApiError } from '@/utils/error'

export default function SeckillForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id
  const [initialValues, setInitialValues] = useState<Partial<CreateSeckillReq>>()

  useEffect(() => {
    if (isEdit) {
      getSeckill(Number(id)).then((activity) => {
        if (activity) {
          setInitialValues({
            name: activity.name,
            startTime: activity.startTime,
            endTime: activity.endTime,
            status: activity.status,
            items: activity.items ?? [],
          })
        }
      }).catch(silentApiError('seckillForm:getSeckill'))
    }
  }, [id, isEdit])

  if (isEdit && !initialValues) {
    return <Card title="编辑秒杀活动" loading />
  }

  return (
    <Card title={isEdit ? '编辑秒杀活动' : '创建秒杀活动'}>
      <ProForm<CreateSeckillReq>
        initialValues={initialValues}
        onFinish={async (values) => {
          const rawItems = values.items ?? []
          const items: SeckillItem[] = rawItems.map((item) => ({
            skuId: Number(item.skuId) || 0,
            seckillPrice: Number(item.seckillPrice) || 0,
            seckillStock: Number(item.seckillStock) || 0,
            perLimit: Number(item.perLimit) || 1,
          }))

          const data: CreateSeckillReq = {
            name: values.name,
            startTime: values.startTime,
            endTime: values.endTime,
            status: values.status,
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
        <ProFormDateTimePicker name="startTime" label="开始时间" rules={[{ required: true }]} />
        <ProFormDateTimePicker name="endTime" label="结束时间" rules={[{ required: true }]} />
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
            <ProFormDigit name="skuId" label="SKU ID" rules={[{ required: true }]} min={1} />
            <ProFormDigit name="seckillPrice" label="秒杀价（分）" rules={[{ required: true }]} min={0} />
            <ProFormDigit name="seckillStock" label="秒杀库存" rules={[{ required: true }]} min={1} />
            <ProFormDigit name="perLimit" label="每人限购" initialValue={1} min={1} />
          </ProForm.Group>
        </ProFormList>
      </ProForm>
    </Card>
  )
}
