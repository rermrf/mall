import { useRef, useState } from 'react'
import { message, Tag } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { ProTable, ModalForm, ProFormText, ProFormDigit, ProFormSelect, ProFormDateTimePicker } from '@ant-design/pro-components'
import type { ActionType, ProColumns } from '@ant-design/pro-components'
import { listPromotions, createPromotion, updatePromotion } from '@/api/marketing'
import type { PromotionRule, CreatePromotionReq } from '@/types/marketing'

export default function PromotionList() {
  const actionRef = useRef<ActionType>(null)
  const [editItem, setEditItem] = useState<PromotionRule | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const columns: ProColumns<PromotionRule>[] = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '名称', dataIndex: 'name' },
    { title: '类型', dataIndex: 'type', render: (_, r) => <Tag>{r.type === 1 ? '满减' : '满赠'}</Tag> },
    { title: '门槛(分)', dataIndex: 'threshold' },
    { title: '优惠值(分)', dataIndex: 'discount_value' },
    { title: '开始时间', dataIndex: 'start_time', valueType: 'dateTime' },
    { title: '结束时间', dataIndex: 'end_time', valueType: 'dateTime' },
    {
      title: '状态',
      dataIndex: 'status',
      valueEnum: { 0: { text: '禁用' }, 1: { text: '启用' } },
    },
    {
      title: '操作',
      render: (_, r) => <a onClick={() => { setEditItem(r); setModalOpen(true) }}>编辑</a>,
    },
  ]

  return (
    <>
      <ProTable<PromotionRule>
        headerTitle="促销规则"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        toolBarRender={() => [
          <ModalForm<CreatePromotionReq>
            key="add"
            title="创建促销"
            trigger={<a><PlusOutlined /> 创建促销</a>}
            onFinish={async (values) => {
              await createPromotion(values)
              message.success('创建成功')
              actionRef.current?.reload()
              return true
            }}
          >
            <ProFormText name="name" label="名称" rules={[{ required: true }]} />
            <ProFormSelect name="type" label="类型" rules={[{ required: true }]} options={[{ label: '满减', value: 1 }, { label: '满赠', value: 2 }]} />
            <ProFormDigit name="threshold" label="门槛(分)" rules={[{ required: true }]} min={0} />
            <ProFormDigit name="discount_value" label="优惠值(分)" rules={[{ required: true }]} min={0} />
            <ProFormDateTimePicker name="start_time" label="开始时间" rules={[{ required: true }]} />
            <ProFormDateTimePicker name="end_time" label="结束时间" rules={[{ required: true }]} />
            <ProFormSelect name="status" label="状态" initialValue={1} options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
          </ModalForm>,
        ]}
        request={async () => {
          const data = await listPromotions()
          return { data: data ?? [], success: true }
        }}
        pagination={false}
      />
      <ModalForm<CreatePromotionReq>
        title="编辑促销"
        open={modalOpen}
        initialValues={editItem ?? {}}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editItem) {
            await updatePromotion(editItem.id, values)
            message.success('更新成功')
            actionRef.current?.reload()
          }
          return true
        }}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormSelect name="type" label="类型" options={[{ label: '满减', value: 1 }, { label: '满赠', value: 2 }]} />
        <ProFormDigit name="threshold" label="门槛(分)" min={0} />
        <ProFormDigit name="discount_value" label="优惠值(分)" min={0} />
        <ProFormDateTimePicker name="start_time" label="开始时间" />
        <ProFormDateTimePicker name="end_time" label="结束时间" />
        <ProFormSelect name="status" label="状态" options={[{ label: '启用', value: 1 }, { label: '禁用', value: 0 }]} />
      </ModalForm>
    </>
  )
}
