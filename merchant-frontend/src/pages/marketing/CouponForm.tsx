import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormDigit, ProFormSelect, ProFormDateTimePicker, ProFormDependency } from '@ant-design/pro-components'
import { createCoupon, updateCoupon, listCoupons } from '@/api/marketing'
import type { CreateCouponReq } from '@/types/marketing'

export default function CouponForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id
  const [initialValues, setInitialValues] = useState<Record<string, unknown>>()

  useEffect(() => {
    if (isEdit) {
      listCoupons({ page: 1, pageSize: 100 }).then((res) => {
        const coupon = (res?.coupons ?? []).find((c) => c.id === Number(id))
        if (coupon) {
          setInitialValues({
            name: coupon.name,
            type: coupon.type,
            threshold: coupon.threshold,
            discount_value: coupon.discount_value,
            total_count: coupon.total_count,
            per_limit: coupon.per_limit,
            start_time: coupon.start_time,
            end_time: coupon.end_time,
            scope_type: coupon.scope_type,
            scope_ids: coupon.scope_ids?.length ? coupon.scope_ids.join(',') : '',
            status: coupon.status,
          })
        }
      }).catch(() => {})
    }
  }, [id, isEdit])

  if (isEdit && !initialValues) {
    return <Card title="编辑优惠券" loading />
  }

  return (
    <Card title={isEdit ? '编辑优惠券' : '创建优惠券'}>
      <ProForm<CreateCouponReq>
        initialValues={initialValues}
        onFinish={async (values) => {
          const scopeType = (values.scope_type as number) ?? 0
          const rawScopeIds = (values as unknown as Record<string, unknown>).scope_ids as string | undefined
          const scopeIds: number[] = scopeType > 0 && rawScopeIds
            ? rawScopeIds.split(',').map((s) => Number(s.trim())).filter((n) => !isNaN(n) && n > 0)
            : []

          const data: CreateCouponReq = {
            ...values,
            scope_type: scopeType,
            scope_ids: scopeIds,
          }

          try {
            if (isEdit) {
              await updateCoupon(Number(id), data)
              message.success('更新成功')
            } else {
              await createCoupon(data)
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
        <ProFormSelect
          name="scope_type"
          label="适用范围"
          initialValue={0}
          rules={[{ required: true }]}
          options={[
            { label: '全场', value: 0 },
            { label: '指定商品', value: 1 },
            { label: '指定分类', value: 2 },
          ]}
        />
        <ProFormDependency name={['scope_type']}>
          {({ scope_type }) => {
            if (scope_type && scope_type > 0) {
              return (
                <ProFormText
                  name="scope_ids"
                  label={scope_type === 1 ? '商品ID（逗号分隔）' : '分类ID（逗号分隔）'}
                  placeholder="例如：1,2,3"
                  rules={[{ required: true, message: '请输入ID' }]}
                />
              )
            }
            return null
          }}
        </ProFormDependency>
        <ProFormSelect name="status" label="状态" initialValue={1} options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
      </ProForm>
    </Card>
  )
}
