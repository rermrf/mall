import { useRef, useState } from 'react'
import { message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable, ModalForm, ProFormText, ProFormTextArea } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listRoles, createRole, updateRole } from '@/api/staff'
import type { Role, CreateRoleReq } from '@/types/user'

export default function RoleList() {
  const actionRef = useRef<ActionType>(null)
  const [editItem, setEditItem] = useState<Role | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const columns: ProColumns<Role>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '角色名称', dataIndex: 'name' },
    { title: '角色编码', dataIndex: 'code' },
    { title: '描述', dataIndex: 'description', ellipsis: true },
    {
      title: '操作',
      render: (_, r) => <a onClick={() => { setEditItem(r); setModalOpen(true) }}>编辑</a>,
    },
  ]

  return (
    <>
      <ProTable<Role>
        headerTitle="角色管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        toolBarRender={() => [
          <ModalForm<CreateRoleReq>
            key="add"
            title="新增角色"
            trigger={<a><PlusOutlined /> 新增角色</a>}
            onFinish={async (values) => {
              await createRole(values)
              message.success('创建成功')
              actionRef.current?.reload()
              return true
            }}
          >
            <ProFormText name="name" label="角色名称" rules={[{ required: true }]} />
            <ProFormText name="code" label="角色编码" rules={[{ required: true }]} />
            <ProFormTextArea name="description" label="描述" />
          </ModalForm>,
        ]}
        request={async () => {
          const data = await listRoles()
          return { data: data ?? [], success: true }
        }}
        pagination={false}
      />
      <ModalForm<CreateRoleReq>
        title="编辑角色"
        open={modalOpen}
        initialValues={editItem ?? {}}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editItem) {
            await updateRole(editItem.id, values)
            message.success('更新成功')
            actionRef.current?.reload()
          }
          return true
        }}
      >
        <ProFormText name="name" label="角色名称" rules={[{ required: true }]} />
        <ProFormText name="code" label="角色编码" rules={[{ required: true }]} />
        <ProFormTextArea name="description" label="描述" />
      </ModalForm>
    </>
  )
}
