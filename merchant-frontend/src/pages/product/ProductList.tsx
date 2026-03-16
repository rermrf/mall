import { useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button, message, Switch, Tag } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listProducts, updateProductStatus } from '@/api/product'
import type { Product } from '@/types/product'

export default function ProductList() {
  const navigate = useNavigate()
  const actionRef = useRef<ActionType>(null)

  const handleStatusChange = async (id: number, checked: boolean) => {
    try {
      await updateProductStatus(id, checked ? 1 : 0)
      message.success(checked ? '已上架' : '已下架')
      actionRef.current?.reload()
    } catch (e: unknown) {
      message.error((e as Error).message)
    }
  }

  const columns: ProColumns<Product>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '商品图片', dataIndex: 'mainImage', valueType: 'image', width: 80, search: false },
    { title: '商品名称', dataIndex: 'name', ellipsis: true },
    { title: '分类ID', dataIndex: 'categoryId', width: 80, search: false },
    {
      title: '价格',
      dataIndex: 'skus',
      search: false,
      width: 120,
      render: (_, record) => {
        const prices = (record.skus ?? []).map((s) => s.price)
        if (prices.length === 0) return '-'
        const min = Math.min(...prices)
        const max = Math.max(...prices)
        return min === max ? `¥${(min / 100).toFixed(2)}` : `¥${(min / 100).toFixed(2)} - ¥${(max / 100).toFixed(2)}`
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      valueEnum: { 0: { text: '下架', status: 'Default' }, 1: { text: '上架', status: 'Success' } },
      render: (_, record) => (
        <Switch
          checked={record.status === 1}
          checkedChildren="上架"
          unCheckedChildren="下架"
          onChange={(checked) => handleStatusChange(record.id, checked)}
        />
      ),
    },
    {
      title: 'SKU数',
      dataIndex: 'skus',
      width: 80,
      search: false,
      render: (_, record) => <Tag>{record.skus?.length ?? 0}</Tag>,
    },
    {
      title: '操作',
      width: 120,
      search: false,
      render: (_, record) => (
        <Button type="link" onClick={() => navigate(`/product/edit/${record.id}`)}>编辑</Button>
      ),
    },
  ]

  return (
    <ProTable<Product>
      headerTitle="商品列表"
      actionRef={actionRef}
      rowKey="id"
      columns={columns}
      toolBarRender={() => [
        <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => navigate('/product/create')}>
          发布商品
        </Button>,
      ]}
      request={async (params) => {
        const { current, pageSize, status } = params
        const res = await listProducts({ page: current, pageSize, status })
        return { data: res?.products ?? [], total: res?.total ?? 0, success: true }
      }}
      pagination={{ defaultPageSize: 20 }}
    />
  )
}
