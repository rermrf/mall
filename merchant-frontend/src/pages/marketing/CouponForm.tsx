import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormDigit, ProFormSelect, ProFormDateTimePicker, ProFormDependency } from '@ant-design/pro-components'
import { createCoupon, updateCoupon, getCoupon } from '@/api/marketing'
import type { CreateCouponReq } from '@/types/marketing'
import { COUPON_TYPE_OPTIONS, COUPON_SCOPE_OPTIONS, COUPON_SCOPE } from '@/constants'
import { silentApiError } from '@/utils/error'

export default function CouponForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id
  const [initialValues, setInitialValues] = useState<Partial<CreateCouponReq & { scope_ids_str: string }>>()

  useEffect(() => {
    if (isEdit) {
      getCoupon(Number(id)).then((coupon) => {
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
            scope_ids_str: coupon.scope_ids?.length ? coupon.scope_ids.join(',') : '',
            status: coupon.status,
          })
        }
      }).catch(silentApiError('couponForm:getCoupon'))
    }
  }, [id, isEdit])

  if (isEdit && !initialValues) {
    return <Card title="编辑优惠券" loading />
  }

  return (
    <Card title={isEdit ? '编辑优惠券' : '创建优惠券'}>
      <ProForm<CreateCouponReq & { scope_ids_str?: string }>
        initialValues={initialValues}
        onFinish={async (values) => {
          const scopeType = values.scope_type ?? COUPON_SCOPE.ALL
          const rawScopeIds = values.scope_ids_str
          const scopeIds: number[] = scopeType > COUPON_SCOPE.ALL && rawScopeIds
            ? rawScopeIds.split(',').map((s) => Number(s.trim())).filter((n) => !isNaN(n) && n > 0)
            : []

          const data: CreateCouponReq = {
            name: values.name,
            type: values.type,
            threshold: values.threshold,
            discount_value: values.discount_value,
            total_count: values.total_count,
            per_limit: values.per_limit,
            start_time: values.start_time,
            end_time: values.end_time,
            scope_type: scopeType,
            scope_ids: scopeIds,
            status: values.status,
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
        <ProFormSelect name="type" label="类型" rules={[{ required: true }]} options={COUPON_TYPE_OPTIONS} />
        <ProFormDigit name="threshold" label="使用门槛（分）" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="discount_value" label="优惠值（分）" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="total_count" label="发放总量" rules={[{ required: true }]} min={1} />
        <ProFormDigit name="per_limit" label="每人限领" initialValue={1} min={1} />
        <ProFormDateTimePicker name="start_time" label="开始时间" rules={[{ required: true }]} />
        <ProFormDateTimePicker name="end_time" label="结束时间" rules={[{ required: true }]} />
        <ProFormSelect
          name="scope_type"
          label="适用范围"
          initialValue={COUPON_SCOPE.ALL}
          rules={[{ required: true }]}
          options={COUPON_SCOPE_OPTIONS}
        />
        <ProFormDependency name={['scope_type']}>
          {({ scope_type }) => {
            if (scope_type && scope_type > COUPON_SCOPE.ALL) {
              return (
                <ProFormText
                  name="scope_ids_str"
                  label={scope_type === COUPON_SCOPE.PRODUCT ? '商品ID（逗号分隔）' : '分类ID（逗号分隔）'}
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
