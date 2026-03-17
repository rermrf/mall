import { useRef, useState } from 'react'
import { ProTable, type ActionType, type ProColumns, ModalForm, ProFormText, ProFormDigit } from '@ant-design/pro-components'
import { Button, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { listCategories, createCategory, updateCategory } from '@/api/category'
import type { Category } from '@/types/product'

export default function CategoryList() {
  const actionRef = useRef<ActionType>()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Category | null>(null)

  const columns: ProColumns<Category>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '分类名称', dataIndex: 'name' },
    { title: '父级ID', dataIndex: 'parentId' },
    { title: '层级', dataIndex: 'level' },
    { title: '排序', dataIndex: 'sort' },
    { title: '图标', dataIndex: 'icon', ellipsis: true },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="edit" onClick={() => { setEditing(record); setModalOpen(true) }}>编辑</a>,
      ],
    },
  ]

  const handleSubmit = async (values: { name: string; parentId?: number; level?: number; sort?: number; icon?: string }) => {
    try {
      if (editing) {
        await updateCategory(editing.id, values)
        message.success('更新成功')
      } else {
        await createCategory(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      setEditing(null)
      actionRef.current?.reload()
      return true
    } catch {
      return false
    }
  }

  return (
    <>
      <ProTable<Category>
        headerTitle="分类管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        request={async () => {
          const res = await listCategories()
          return { data: res ?? [], total: res?.length ?? 0, success: true }
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => { setEditing(null); setModalOpen(true) }}>
            新建分类
          </Button>,
        ]}
      />
      <ModalForm
        title={editing ? '编辑分类' : '新建分类'}
        open={modalOpen}
        onOpenChange={setModalOpen}
        onFinish={handleSubmit}
        initialValues={editing ?? {}}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="分类名称" rules={[{ required: true }]} />
        <ProFormDigit name="parentId" label="父级ID" placeholder="顶级分类留空" min={0} />
        <ProFormDigit name="level" label="层级" min={1} />
        <ProFormDigit name="sort" label="排序值" min={0} />
        <ProFormText name="icon" label="图标" />
      </ModalForm>
    </>
  )
}
