import { useRef, useState } from 'react'
import { ProTable, type ActionType, type ProColumns, ModalForm, ProFormText, ProFormDigit } from '@ant-design/pro-components'
import { Button, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { listRoles, createRole, updateRole } from '@/api/role'
import type { Role } from '@/types/user'

export default function RoleList() {
  const actionRef = useRef<ActionType>()
  const [modalOpen, setModalOpen] = useState(false)
  const [editingRole, setEditingRole] = useState<Role | null>(null)

  const columns: ProColumns<Role>[] = [
    { title: 'ID', dataIndex: 'id', width: 80, search: false },
    { title: '角色名', dataIndex: 'name', search: false },
    { title: '角色编码', dataIndex: 'code', search: false },
    { title: '描述', dataIndex: 'description', search: false, ellipsis: true },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      valueType: 'digit',
      hideInTable: true,
      fieldProps: { placeholder: '按租户ID筛选' },
    },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="edit" onClick={() => { setEditingRole(record); setModalOpen(true) }}>编辑</a>,
      ],
    },
  ]

  const handleSubmit = async (values: { name: string; code: string; description: string; tenantId?: number }) => {
    try {
      if (editingRole) {
        await updateRole(editingRole.id, { name: values.name, code: values.code, description: values.description })
        message.success('更新成功')
      } else {
        await createRole({ tenantId: values.tenantId, name: values.name, code: values.code, description: values.description })
        message.success('创建成功')
      }
      setModalOpen(false)
      setEditingRole(null)
      actionRef.current?.reload()
      return true
    } catch {
      return false
    }
  }

  return (
    <>
      <ProTable<Role>
        headerTitle="角色管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        request={async (params) => {
          const res = await listRoles(params.tenantId)
          return { data: res ?? [], total: res?.length ?? 0, success: true }
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => { setEditingRole(null); setModalOpen(true) }}>
            新建角色
          </Button>,
        ]}
        search={{ labelWidth: 'auto' }}
      />
      <ModalForm
        title={editingRole ? '编辑角色' : '新建角色'}
        open={modalOpen}
        onOpenChange={setModalOpen}
        onFinish={handleSubmit}
        initialValues={editingRole ? { name: editingRole.name, code: editingRole.code, description: editingRole.description } : {}}
        modalProps={{ destroyOnClose: true }}
      >
        {!editingRole && (
          <ProFormDigit name="tenantId" label="租户ID" placeholder="留空为平台角色" />
        )}
        <ProFormText name="name" label="角色名" rules={[{ required: true }]} />
        <ProFormText name="code" label="角色编码" rules={[{ required: true }]} />
        <ProFormText name="description" label="描述" />
      </ModalForm>
    </>
  )
}
