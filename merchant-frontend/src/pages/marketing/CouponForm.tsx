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
  const [initialValues, setInitialValues] = useState<Partial<CreateCouponReq & { scopeIdsStr: string }>>()

  useEffect(() => {
    if (isEdit) {
      getCoupon(Number(id)).then((coupon) => {
        if (coupon) {
          setInitialValues({
            name: coupon.name,
            type: coupon.type,
            threshold: coupon.threshold,
            discountValue: coupon.discountValue,
            totalCount: coupon.totalCount,
            perLimit: coupon.perLimit,
            startTime: coupon.startTime,
            endTime: coupon.endTime,
            scopeType: coupon.scopeType,
            scopeIdsStr: coupon.scopeIds?.length ? coupon.scopeIds.join(',') : '',
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
      <ProForm<CreateCouponReq & { scopeIdsStr?: string }>
        initialValues={initialValues}
        onFinish={async (values) => {
          const scopeType = values.scopeType ?? COUPON_SCOPE.ALL
          const rawScopeIds = values.scopeIdsStr
          const scopeIds: number[] = scopeType > COUPON_SCOPE.ALL && rawScopeIds
            ? rawScopeIds.split(',').map((s) => Number(s.trim())).filter((n) => !isNaN(n) && n > 0)
            : []

          const data: CreateCouponReq = {
            name: values.name,
            type: values.type,
            threshold: values.threshold,
            discountValue: values.discountValue,
            totalCount: values.totalCount,
            perLimit: values.perLimit,
            startTime: new Date(values.startTime).getTime(),
            endTime: new Date(values.endTime).getTime(),
            scopeType: scopeType,
            scopeIds: scopeIds.length ? scopeIds.join(',') : '',
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
        <ProFormDigit name="discountValue" label="优惠值（分）" rules={[{ required: true }]} min={0} />
        <ProFormDigit name="totalCount" label="发放总量" rules={[{ required: true }]} min={1} />
        <ProFormDigit name="perLimit" label="每人限领" initialValue={1} min={1} />
        <ProFormDateTimePicker name="startTime" label="开始时间" rules={[{ required: true }]} />
        <ProFormDateTimePicker name="endTime" label="结束时间" rules={[{ required: true }]} />
        <ProFormSelect
          name="scopeType"
          label="适用范围"
          initialValue={COUPON_SCOPE.ALL}
          rules={[{ required: true }]}
          options={COUPON_SCOPE_OPTIONS}
        />
        <ProFormDependency name={['scopeType']}>
          {({ scopeType }) => {
            if (scopeType && scopeType > COUPON_SCOPE.ALL) {
              return (
                <ProFormText
                  name="scopeIdsStr"
                  label={scopeType === COUPON_SCOPE.PRODUCT ? '商品ID（逗号分隔）' : '分类ID（逗号分隔）'}
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
