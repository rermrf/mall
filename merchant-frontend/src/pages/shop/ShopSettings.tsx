import { useEffect, useState } from 'react'
import { Card, message, Spin } from 'antd'
import { ProForm, ProFormText, ProFormTextArea } from '@ant-design/pro-components'
import { getShop, updateShop } from '@/api/shop'
import type { Shop, UpdateShopReq } from '@/types/shop'
import { silentApiError } from '@/utils/error'

export default function ShopSettings() {
  const [shop, setShop] = useState<Shop | null>(null)

  useEffect(() => {
    getShop().then(setShop).catch(silentApiError('shop:getShop'))
  }, [])

  return (
    <Card title="店铺设置">
      <Spin spinning={!shop}>
        {shop && (
          <ProForm<UpdateShopReq>
          initialValues={{
            name: shop.name,
            logo: shop.logo,
            description: shop.description,
            subdomain: shop.subdomain,
            custom_domain: shop.custom_domain,
          }}
          onFinish={async (values) => {
            try {
              await updateShop(values)
              message.success('保存成功')
            } catch (e: unknown) {
              message.error((e as Error).message)
            }
          }}
        >
          <ProFormText name="name" label="店铺名称" rules={[{ required: true }]} />
          <ProFormText name="logo" label="Logo URL" />
          <ProFormTextArea name="description" label="店铺描述" />
          <ProFormText name="subdomain" label="子域名" />
          <ProFormText name="custom_domain" label="自定义域名" />
        </ProForm>
      )}
      </Spin>
    </Card>
  )
}
