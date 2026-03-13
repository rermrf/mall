import { useRef, useState } from 'react'
import { message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable, ModalForm, ProFormText, ProFormSelect } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listBrands, createBrand, updateBrand } from '@/api/product'
import type { Brand, CreateBrandReq } from '@/types/product'

export default function BrandList() {
  const actionRef = useRef<ActionType>(null)
  const [editItem, setEditItem] = useState<Brand | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const columns: ProColumns<Brand>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: 'Logo', dataIndex: 'logo', valueType: 'image', width: 80, search: false },
    { title: '品牌名称', dataIndex: 'name' },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      valueEnum: { 0: { text: '禁用', status: 'Default' }, 1: { text: '启用', status: 'Success' } },
    },
    {
      title: '操作',
      width: 100,
      search: false,
      render: (_, record) => (
        <a onClick={() => { setEditItem(record); setModalOpen(true) }}>编辑</a>
      ),
    },
  ]

  return (
    <>
      <ProTable<Brand>
        headerTitle="品牌管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        toolBarRender={() => [
          <ModalForm<CreateBrandReq>
            key="add"
            title="新增品牌"
            trigger={<a><PlusOutlined /> 新增品牌</a>}
            onFinish={async (values) => {
              await createBrand(values)
              message.success('创建成功')
              actionRef.current?.reload()
              return true
            }}
          >
            <ProFormText name="name" label="品牌名称" rules={[{ required: true }]} />
            <ProFormText name="logo" label="Logo URL" />
            <ProFormSelect name="status" label="状态" initialValue={1} options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
          </ModalForm>,
        ]}
        request={async (params) => {
          const res = await listBrands({ page: params.current, pageSize: params.pageSize })
          return { data: res?.brands ?? [], total: res?.total ?? 0, success: true }
        }}
        pagination={{ defaultPageSize: 20 }}
      />
      <ModalForm<CreateBrandReq>
        title="编辑品牌"
        open={modalOpen}
        initialValues={editItem ?? {}}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editItem) {
            await updateBrand(editItem.id, values)
            message.success('更新成功')
            actionRef.current?.reload()
          }
          return true
        }}
      >
        <ProFormText name="name" label="品牌名称" rules={[{ required: true }]} />
        <ProFormText name="logo" label="Logo URL" />
        <ProFormSelect name="status" label="状态" options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
      </ModalForm>
    </>
  )
}
