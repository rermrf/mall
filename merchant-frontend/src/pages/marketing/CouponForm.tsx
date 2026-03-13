import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormDigit, ProFormSelect, ProFormDateTimePicker } from '@ant-design/pro-components'
import { createCoupon, updateCoupon } from '@/api/marketing'
import type { CreateCouponReq } from '@/types/marketing'

export default function CouponForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id

  return (
    <Card title={isEdit ? '编辑优惠券' : '创建优惠券'}>
      <ProForm<CreateCouponReq>
        onFinish={async (values) => {
          try {
            if (isEdit) {
              await updateCoupon(Number(id), values)
              message.success('更新成功')
            } else {
              await createCoupon(values)
              message.success('创建成功')
            }
            navigate('/marketing/coupon')
          } catch (e: unknown) {
            message.error((e as Error).message)
          }
        }}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormSelect name="type" label="类型" rules={[{ required: true }]} options={[
          { label: '满减', value: 1 },
          { label: '折扣', value: 2 },
          { label: '固定金额', value: 3 },
        ]} />
        <ProFormDigit name="threshold" label="使用门槛（分）" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="discount_value" label="优惠值（分）" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="total_count" label="发放总量" rules={[{ required: true }]} min={1} />
        <ProFormDigit name="per_limit" label="每人限领" initialValue={1} min={1} />
        <ProFormDateTimePicker name="start_time" label="开始时间" rules={[{ required: true }]} />
        <ProFormDateTimePicker name="end_time" label="结束时间" rules={[{ required: true }]} />
        <ProFormSelect name="status" label="状态" initialValue={1} options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
      </ProForm>
    </Card>
  )
}
