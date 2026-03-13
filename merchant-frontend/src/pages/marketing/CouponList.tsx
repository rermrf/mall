import { useNavigate } from 'react-router-dom'
import { Button, Tag } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
import { listCoupons } from '@/api/marketing'
import type { Coupon } from '@/types/marketing'

const typeMap: Record<number, string> = { 1: '满减', 2: '折扣', 3: '固定金额' }

export default function CouponList() {
  const navigate = useNavigate()

  const columns: ProColumns<Coupon>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '名称', dataIndex: 'name' },
    { title: '类型', dataIndex: 'type', search: false, render: (_, r) => <Tag>{typeMap[r.type] ?? '未知'}</Tag> },
    { title: '门槛(分)', dataIndex: 'threshold', search: false },
    { title: '优惠值(分)', dataIndex: 'discount_value', search: false },
    { title: '总数', dataIndex: 'total_count', search: false },
    { title: '已用', dataIndex: 'used_count', search: false },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: { 0: { text: '禁用' }, 1: { text: '启用' } },
    },
    { title: '开始时间', dataIndex: 'start_time', valueType: 'dateTime', search: false },
    { title: '结束时间', dataIndex: 'end_time', valueType: 'dateTime', search: false },
    {
      title: '操作',
      search: false,
      render: (_, r) => <a onClick={() => navigate(`/marketing/coupon/edit/${r.id}`)}>编辑</a>,
    },
  ]

  return (
    <ProTable<Coupon>
      headerTitle="优惠券管理"
      rowKey="id"
      columns={columns}
      toolBarRender={() => [
        <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => navigate('/marketing/coupon/create')}>
          创建优惠券
        </Button>,
      ]}
      request={async (params) => {
        const res = await listCoupons({ status: params.status, page: params.current, pageSize: params.pageSize })
        return { data: res?.coupons ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
