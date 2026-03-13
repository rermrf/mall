import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormSelect, ProFormDateTimePicker } from '@ant-design/pro-components'
import { createSeckill, updateSeckill } from '@/api/marketing'
import type { CreateSeckillReq } from '@/types/marketing'

export default function SeckillForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id

  return (
    <Card title={isEdit ? '编辑秒杀活动' : '创建秒杀活动'}>
      <ProForm
        onFinish={async (values: Record<string, unknown>) => {
          const data: CreateSeckillReq = {
            name: values.name as string,
            start_time: values.start_time as string,
            end_time: values.end_time as string,
            status: values.status as number,
            items: [],
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
      </ProForm>
    </Card>
  )
}
