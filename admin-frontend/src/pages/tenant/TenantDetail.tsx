import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Descriptions, Card, Spin, Tag } from 'antd'
import { getTenant } from '@/api/tenant'
import type { Tenant } from '@/types/tenant'

const TENANT_STATUS_MAP: Record<number, { text: string; color: string }> = {
  0: { text: '待审核', color: 'orange' },
  1: { text: '正常', color: 'green' },
  2: { text: '已冻结', color: 'red' },
  3: { text: '已拒绝', color: 'default' },
}

export default function TenantDetail() {
  const { id } = useParams<{ id: string }>()
  const [tenant, setTenant] = useState<Tenant | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    getTenant(Number(id)).then(setTenant).catch(() => {}).finally(() => setLoading(false))
  }, [id])

  if (loading) return <Spin style={{ display: 'block', margin: '20vh auto' }} size="large" />
  if (!tenant) return <div>租户不存在</div>

  const s = TENANT_STATUS_MAP[tenant.status] || { text: '未知', color: 'default' }

  return (
    <Card title="租户详情">
      <Descriptions column={2} bordered>
        <Descriptions.Item label="ID">{tenant.id}</Descriptions.Item>
        <Descriptions.Item label="名称">{tenant.name}</Descriptions.Item>
        <Descriptions.Item label="联系人">{tenant.contactName}</Descriptions.Item>
        <Descriptions.Item label="联系电话">{tenant.contactPhone}</Descriptions.Item>
        <Descriptions.Item label="营业执照">{tenant.businessLicense}</Descriptions.Item>
        <Descriptions.Item label="套餐ID">{tenant.planId}</Descriptions.Item>
        <Descriptions.Item label="状态"><Tag color={s.color}>{s.text}</Tag></Descriptions.Item>
        <Descriptions.Item label="创建时间">{tenant.createdAt}</Descriptions.Item>
      </Descriptions>
    </Card>
  )
}
