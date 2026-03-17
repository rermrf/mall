import { useRef, useState } from 'react'
import { ProTable, type ActionType, type ProColumns, ModalForm, ProFormText } from '@ant-design/pro-components'
import { Button, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { listBrands, createBrand, updateBrand } from '@/api/brand'
import type { Brand } from '@/types/product'

export default function BrandList() {
  const actionRef = useRef<ActionType>()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Brand | null>(null)

  const columns: ProColumns<Brand>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '品牌名称', dataIndex: 'name' },
    { title: 'Logo', dataIndex: 'logo', ellipsis: true },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="edit" onClick={() => { setEditing(record); setModalOpen(true) }}>编辑</a>,
      ],
    },
  ]

  const handleSubmit = async (values: { name: string; logo?: string }) => {
    try {
      if (editing) {
        await updateBrand(editing.id, values)
        message.success('更新成功')
      } else {
        await createBrand(values)
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
      <ProTable<Brand>
        headerTitle="品牌管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        request={async (params) => {
          const res = await listBrands({ page: params.current ?? 1, pageSize: params.pageSize ?? 20 })
          return { data: res?.brands ?? [], total: res?.total ?? 0, success: true }
        }}
        pagination={{ defaultPageSize: 20 }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => { setEditing(null); setModalOpen(true) }}>
            新建品牌
          </Button>,
        ]}
      />
      <ModalForm
        title={editing ? '编辑品牌' : '新建品牌'}
        open={modalOpen}
        onOpenChange={setModalOpen}
        onFinish={handleSubmit}
        initialValues={editing ?? {}}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="品牌名称" rules={[{ required: true }]} />
        <ProFormText name="logo" label="Logo URL" />
      </ModalForm>
    </>
  )
}
