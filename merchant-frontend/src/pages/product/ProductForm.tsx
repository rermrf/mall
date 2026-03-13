import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message } from 'antd'
import { ProForm, ProFormText, ProFormTextArea, ProFormDigit, ProFormSelect, StepsForm } from '@ant-design/pro-components'
import { createProduct, updateProduct, getProduct, listCategories, listBrands } from '@/api/product'
import type { Category, Brand, CreateProductReq, CreateSKUReq, ProductSpec } from '@/types/product'

export default function ProductForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id

  const [categories, setCategories] = useState<Category[]>([])
  const [brands, setBrands] = useState<Brand[]>([])
  const [initialValues, setInitialValues] = useState<Record<string, unknown>>({})
  const [skus, setSkus] = useState<CreateSKUReq[]>([])
  const [specs, setSpecs] = useState<ProductSpec[]>([])

  useEffect(() => {
    listCategories().then((c) => setCategories(c ?? [])).catch(() => {})
    listBrands({ page: 1, pageSize: 100 }).then((r) => setBrands(r?.brands ?? [])).catch(() => {})
    if (isEdit) {
      getProduct(Number(id)).then((p) => {
        if (p) {
          setInitialValues({
            name: p.name,
            subtitle: p.subtitle,
            category_id: p.category_id,
            brand_id: p.brand_id,
            main_image: p.main_image,
            description: p.description,
          })
          setSkus(p.skus?.map((s) => ({
            sku_code: s.sku_code,
            price: s.price,
            original_price: s.original_price,
            cost_price: s.cost_price,
            bar_code: s.bar_code,
            spec_values: s.spec_values,
            status: s.status,
          })) ?? [])
          setSpecs(p.specs ?? [])
        }
      }).catch(() => {})
    }
  }, [id, isEdit])

  const flatCategories = (cats: Category[], prefix = ''): { label: string; value: number }[] => {
    return cats.flatMap((c) => [
      { label: prefix + c.name, value: c.id },
      ...(c.children ? flatCategories(c.children, prefix + '  ') : []),
    ])
  }

  const handleSubmit = async (values: Record<string, unknown>) => {
    const data: CreateProductReq = {
      category_id: values.category_id as number,
      brand_id: values.brand_id as number,
      name: values.name as string,
      subtitle: (values.subtitle as string) || '',
      main_image: (values.main_image as string) || '',
      images: [],
      description: (values.description as string) || '',
      status: 0,
      skus: skus.length > 0 ? skus : [{
        sku_code: 'DEFAULT',
        price: ((values.price as number) || 0) * 100,
        original_price: ((values.original_price as number) || 0) * 100,
        cost_price: 0,
        bar_code: '',
        spec_values: '',
        status: 1,
      }],
      specs,
    }

    try {
      if (isEdit) {
        await updateProduct(Number(id), data)
        message.success('更新成功')
      } else {
        await createProduct(data)
        message.success('创建成功')
      }
      navigate('/product/list')
    } catch (e: unknown) {
      message.error((e as Error).message)
    }
  }

  return (
    <Card title={isEdit ? '编辑商品' : '发布商品'}>
      <StepsForm onFinish={handleSubmit}>
        <StepsForm.StepForm name="basic" title="基本信息" initialValues={initialValues}>
          <ProFormText name="name" label="商品名称" rules={[{ required: true }]} />
          <ProFormText name="subtitle" label="副标题" />
          <ProFormSelect name="category_id" label="分类" rules={[{ required: true }]} options={flatCategories(categories)} />
          <ProFormSelect name="brand_id" label="品牌" options={brands.map((b) => ({ label: b.name, value: b.id }))} />
          <ProFormText name="main_image" label="主图URL" />
          <ProFormTextArea name="description" label="商品描述" />
        </StepsForm.StepForm>
        <StepsForm.StepForm name="sku" title="价格库存">
          <ProFormDigit name="price" label="售价（元）" rules={[{ required: true }]} min={0} fieldProps={{ precision: 2 }} />
          <ProFormDigit name="original_price" label="原价（元）" min={0} fieldProps={{ precision: 2 }} />
        </StepsForm.StepForm>
      </StepsForm>
    </Card>
  )
}
