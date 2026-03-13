import { useNavigate } from 'react-router-dom'
import { Button, Popconfirm, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
import { listFreightTemplates, deleteFreightTemplate } from '@/api/logistics'
import type { FreightTemplate } from '@/types/logistics'

const chargeTypeMap: Record<number, string> = { 1: '按重量', 2: '按件数' }

export default function TemplateList() {
  const navigate = useNavigate()

  const columns: ProColumns<FreightTemplate>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '模板名称', dataIndex: 'name' },
    { title: '计费方式', dataIndex: 'charge_type', render: (_, r) => chargeTypeMap[r.charge_type] ?? '未知' },
    { title: '免邮门槛(分)', dataIndex: 'free_threshold' },
    { title: '规则数', dataIndex: 'rules', render: (_, r) => r.rules?.length ?? 0 },
    { title: '创建时间', dataIndex: 'created_at', valueType: 'dateTime' },
    {
      title: '操作',
      render: (_, r) => (
        <>
          <a onClick={() => navigate(`/logistics/template/edit/${r.id}`)}>编辑</a>
          <Popconfirm title="确认删除？" onConfirm={async () => {
            await deleteFreightTemplate(r.id)
            message.success('已删除')
          }}>
            <a style={{ marginLeft: 8, color: 'red' }}>删除</a>
          </Popconfirm>
        </>
      ),
    },
  ]

  return (
    <ProTable<FreightTemplate>
      headerTitle="运费模板"
      rowKey="id"
      columns={columns}
      search={false}
      toolBarRender={() => [
        <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => navigate('/logistics/template/create')}>
          创建模板
        </Button>,
      ]}
      request={async () => {
        const data = await listFreightTemplates()
        return { data: data ?? [], success: true }
      }}
      pagination={false}
    />
  )
}
