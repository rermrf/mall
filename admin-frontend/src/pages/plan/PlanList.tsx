import { useRef, useState } from 'react'
import { ProTable, type ActionType, type ProColumns, ModalForm, ProFormText, ProFormDigit, ProFormTextArea } from '@ant-design/pro-components'
import { Button, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { listPlans, createPlan, updatePlan, type PlanData } from '@/api/plan'
import type { TenantPlan } from '@/types/tenant'
import { formatPrice, parsePriceToFen } from '@/constants'

export default function PlanList() {
  const actionRef = useRef<ActionType>()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TenantPlan | null>(null)

  const columns: ProColumns<TenantPlan>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '套餐名称', dataIndex: 'name' },
    { title: '价格', dataIndex: 'price', render: (_, r) => formatPrice(r.price) },
    { title: '有效天数', dataIndex: 'durationDays' },
    { title: '最大商品数', dataIndex: 'maxProducts' },
    { title: '最大员工数', dataIndex: 'maxStaff' },
    { title: '特性说明', dataIndex: 'features', ellipsis: true },
    {
      title: '操作',
      valueType: 'option',
      render: (_, record) => [
        <a key="edit" onClick={() => { setEditing(record); setModalOpen(true) }}>编辑</a>,
      ],
    },
  ]

  const handleSubmit = async (values: { name: string; price: number; durationDays: number; maxProducts: number; maxStaff: number; features: string }) => {
    const data: PlanData = { ...values, price: parsePriceToFen(values.price) }
    try {
      if (editing) {
        await updatePlan(editing.id, data)
        message.success('更新成功')
      } else {
        await createPlan(data)
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
      <ProTable<TenantPlan>
        headerTitle="套餐管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        request={async () => {
          const res = await listPlans()
          return { data: res?.plans ?? [], total: res?.plans?.length ?? 0, success: true }
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => { setEditing(null); setModalOpen(true) }}>
            新建套餐
          </Button>,
        ]}
      />
      <ModalForm
        title={editing ? '编辑套餐' : '新建套餐'}
        open={modalOpen}
        onOpenChange={setModalOpen}
        onFinish={handleSubmit}
        initialValues={editing ? { ...editing, price: editing.price / 100 } : {}}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="套餐名称" rules={[{ required: true }]} />
        <ProFormDigit name="price" label="价格（元）" rules={[{ required: true }]} min={0} fieldProps={{ precision: 2 }} />
        <ProFormDigit name="durationDays" label="有效天数" rules={[{ required: true }]} min={1} />
        <ProFormDigit name="maxProducts" label="最大商品数" rules={[{ required: true }]} min={1} />
        <ProFormDigit name="maxStaff" label="最大员工数" rules={[{ required: true }]} min={1} />
        <ProFormTextArea name="features" label="特性说明" />
      </ModalForm>
    </>
  )
}
