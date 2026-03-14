import { useEffect, useState } from 'react'
import { Card, message, Spin } from 'antd'
import { ProForm, ProFormText } from '@ant-design/pro-components'
import { getProfile, updateProfile } from '@/api/staff'
import type { User } from '@/types/user'

export default function ProfileEdit() {
  const [user, setUser] = useState<User | null>(null)

  useEffect(() => {
    getProfile().then(setUser).catch(() => {})
  }, [])

  return (
    <Card title="个人资料">
      <Spin spinning={!user}>
        {user && (
          <ProForm<{ nickname: string; avatar: string }>
            initialValues={{
              nickname: user.nickname,
              avatar: user.avatar,
            }}
            onFinish={async (values) => {
              try {
                await updateProfile({ nickname: values.nickname, avatar: values.avatar })
                message.success('保存成功')
              } catch (e: unknown) {
                message.error((e as Error).message)
              }
            }}
          >
            <ProFormText name="nickname" label="昵称" rules={[{ required: true, message: '请输入昵称' }]} />
            <ProFormText name="avatar" label="头像 URL" />
          </ProForm>
        )}
      </Spin>
    </Card>
  )
}
