import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Descriptions, Card, Spin, Table, Divider } from 'antd'
import { getFreightTemplate } from '@/api/logistics'
import type { FreightTemplate } from '@/types/logistics'
import { formatPrice } from '@/constants'

export default function FreightDetail() {
  const { id } = useParams<{ id: string }>()
  const [template, setTemplate] = useState<FreightTemplate | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    getFreightTemplate(Number(id)).then(setTemplate).catch(() => {}).finally(() => setLoading(false))
  }, [id])

  if (loading) return <Spin style={{ display: 'block', margin: '20vh auto' }} size="large" />
  if (!template) return <div>模板不存在</div>

  return (
    <div>
      <Card title="运费模板详情">
        <Descriptions column={2} bordered>
          <Descriptions.Item label="ID">{template.id}</Descriptions.Item>
          <Descriptions.Item label="模板名称">{template.name}</Descriptions.Item>
          <Descriptions.Item label="租户ID">{template.tenantId}</Descriptions.Item>
          <Descriptions.Item label="计费方式">{template.chargeType === 1 ? '按重量' : '按件数'}</Descriptions.Item>
        </Descriptions>
      </Card>

      {template.regions && template.regions.length > 0 && (
        <>
          <Divider />
          <Card title="区域运费规则">
            <Table
              dataSource={template.regions}
              rowKey="region"
              pagination={false}
              columns={[
                { title: '区域', dataIndex: 'region' },
                { title: '首重(g)', dataIndex: 'firstWeight' },
                { title: '首费', dataIndex: 'firstFee', render: (v: number) => formatPrice(v) },
                { title: '续重(g)', dataIndex: 'continueWeight' },
                { title: '续费', dataIndex: 'continueFee', render: (v: number) => formatPrice(v) },
              ]}
            />
          </Card>
        </>
      )}
    </div>
  )
}
