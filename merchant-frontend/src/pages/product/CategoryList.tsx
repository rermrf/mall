import { useRef, useState } from 'react'
import { message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable, ModalForm, ProFormText, ProFormDigit, ProFormSelect } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listCategories, createCategory, updateCategory } from '@/api/product'
import type { Category, CreateCategoryReq } from '@/types/product'

export default function CategoryList() {
  const actionRef = useRef<ActionType>(null)
  const [editItem, setEditItem] = useState<Category | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const columns: ProColumns<Category>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '名称', dataIndex: 'name' },
    { title: '排序', dataIndex: 'sort', width: 80 },
    { title: '层级', dataIndex: 'level', width: 80 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      valueEnum: { 0: { text: '禁用', status: 'Default' }, 1: { text: '启用', status: 'Success' } },
    },
    {
      title: '操作',
      width: 100,
      render: (_, record) => (
        <a onClick={() => { setEditItem(record); setModalOpen(true) }}>编辑</a>
      ),
    },
  ]

  return (
    <>
      <ProTable<Category>
        headerTitle="分类管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        toolBarRender={() => [
          <ModalForm<CreateCategoryReq>
            key="add"
            title="新增分类"
            trigger={<a><PlusOutlined /> 新增分类</a>}
            onFinish={async (values) => {
              await createCategory(values)
              message.success('创建成功')
              actionRef.current?.reload()
              return true
            }}
          >
            <ProFormText name="name" label="名称" rules={[{ required: true }]} />
            <ProFormDigit name="sort" label="排序" initialValue={0} />
            <ProFormDigit name="parent_id" label="父级ID" initialValue={0} />
            <ProFormDigit name="level" label="层级" initialValue={1} />
            <ProFormText name="icon" label="图标" />
            <ProFormSelect name="status" label="状态" initialValue={1} options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
          </ModalForm>,
        ]}
        request={async () => {
          const data = await listCategories()
          return { data: data ?? [], success: true }
        }}
        pagination={false}
      />
      <ModalForm<CreateCategoryReq>
        title="编辑分类"
        open={modalOpen}
        initialValues={editItem ?? {}}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editItem) {
            await updateCategory(editItem.id, values)
            message.success('更新成功')
            actionRef.current?.reload()
          }
          return true
        }}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormDigit name="sort" label="排序" />
        <ProFormDigit name="parent_id" label="父级ID" />
        <ProFormDigit name="level" label="层级" />
        <ProFormText name="icon" label="图标" />
        <ProFormSelect name="status" label="状态" options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
      </ModalForm>
    </>
  )
}
