import { useNavigate } from 'react-router-dom'
import { Button } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
import { listSeckill } from '@/api/marketing'
import type { SeckillActivity } from '@/types/marketing'

export default function SeckillList() {
  const navigate = useNavigate()

  const columns: ProColumns<SeckillActivity>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '活动名称', dataIndex: 'name' },
    { title: '商品数', dataIndex: 'items', search: false, render: (_, r) => r.items?.length ?? 0 },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: { 0: { text: '未开始' }, 1: { text: '进行中' }, 2: { text: '已结束' } },
    },
    { title: '开始时间', dataIndex: 'startTime', valueType: 'dateTime', search: false },
    { title: '结束时间', dataIndex: 'endTime', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => <a onClick={() => navigate(`/marketing/seckill/edit/${r.id}`)}>编辑</a>,
    },
  ]

  return (
    <ProTable<SeckillActivity>
      headerTitle="秒杀活动"
      rowKey="id"
      columns={columns}
      toolBarRender={() => [
        <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => navigate('/marketing/seckill/create')}>
          创建秒杀
        </Button>,
      ]}
      request={async (params) => {
        const res = await listSeckill({ status: params.status, page: params.current, pageSize: params.pageSize })
        return { data: res?.activities ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
