import { useEffect, useState, useCallback } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message, Button } from 'antd'
import { ProForm, ProFormText, ProFormTextArea, ProFormDigit, ProFormSelect, ProFormList, StepsForm, EditableProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
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
  const [editableKeys, setEditableKeys] = useState<React.Key[]>([])

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

  /** Generate cartesian product of spec values to create SKU combinations */
  const generateSkuCombinations = useCallback((specList: ProductSpec[]): CreateSKUReq[] => {
    const validSpecs = specList.filter((s) => s.name && s.values.length > 0)
    if (validSpecs.length === 0) return []

    const combine = (arrays: string[][]): string[][] => {
      if (arrays.length === 0) return [[]]
      const [first, ...rest] = arrays
      const restCombos = combine(rest)
      return first.flatMap((v) => restCombos.map((rc) => [v, ...rc]))
    }

    const combos = combine(validSpecs.map((s) => s.values))
    return combos.map((combo, idx) => {
      const specValues = combo.join(',')
      // Try to find an existing SKU with the same spec_values to preserve its data
      const existing = skus.find((s) => s.spec_values === specValues)
      return {
        sku_code: existing?.sku_code || `SKU-${idx + 1}`,
        price: existing?.price ?? 0,
        original_price: existing?.original_price ?? 0,
        cost_price: existing?.cost_price ?? 0,
        bar_code: existing?.bar_code || '',
        spec_values: specValues,
        status: existing?.status ?? 1,
      }
    })
  }, [skus])

  const handleSpecsChange = (rawSpecs: Array<{ spec_name?: string; spec_values?: string }> | undefined) => {
    const parsed: ProductSpec[] = (rawSpecs ?? [])
      .filter((r) => r.spec_name)
      .map((r) => ({
        name: r.spec_name!,
        values: (r.spec_values || '').split(',').map((v) => v.trim()).filter(Boolean),
      }))
    setSpecs(parsed)
    const newSkus = generateSkuCombinations(parsed)
    setSkus(newSkus)
    setEditableKeys(newSkus.map((_, i) => i))
  }

  const skuColumns: ProColumns<CreateSKUReq & { id?: number }>[] = [
    { title: '规格值', dataIndex: 'spec_values', editable: false, width: 160 },
    { title: '价格（分）', dataIndex: 'price', valueType: 'digit', width: 120 },
    { title: '原价（分）', dataIndex: 'original_price', valueType: 'digit', width: 120 },
    { title: '成本价（分）', dataIndex: 'cost_price', valueType: 'digit', width: 120 },
    { title: '条码', dataIndex: 'bar_code', width: 140 },
    { title: 'SKU编码', dataIndex: 'sku_code', width: 140 },
    { title: '操作', valueType: 'option' },
  ]

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
          <ProFormList
            name="spec_list"
            label="商品规格"
            creatorButtonProps={{ creatorButtonText: '添加规格' }}
            initialValue={specs.map((s) => ({ spec_name: s.name, spec_values: s.values.join(',') }))}
            onAfterChange={(_, rawSpecs) => {
              handleSpecsChange(rawSpecs as Array<{ spec_name?: string; spec_values?: string }> | undefined)
            }}
            actionRender={(_field, action) => [
              <Button key="delete" type="link" danger onClick={() => action.remove(_field.name)}>
                删除
              </Button>,
            ]}
          >
            <ProForm.Group key="spec-group">
              <ProFormText name="spec_name" label="规格名" placeholder="如：颜色" rules={[{ required: true }]} />
              <ProFormText name="spec_values" label="规格值（逗号分隔）" placeholder="如：红,蓝,绿" rules={[{ required: true }]} />
            </ProForm.Group>
          </ProFormList>

          {specs.length > 0 && skus.length > 0 ? (
            <EditableProTable<CreateSKUReq & { id?: number }>
              headerTitle="SKU列表"
              rowKey="spec_values"
              value={skus.map((s, i) => ({ ...s, id: i }))}
              columns={skuColumns}
              editable={{
                type: 'multiple',
                editableKeys,
                onChange: setEditableKeys,
                onValuesChange: (_record, recordList) => {
                  setSkus(recordList.map(({ id: _id, ...rest }) => rest as CreateSKUReq))
                },
              }}
              recordCreatorProps={false}
            />
          ) : (
            <>
              <ProFormDigit name="price" label="售价（元）" rules={[{ required: true }]} min={0} fieldProps={{ precision: 2 }} />
              <ProFormDigit name="original_price" label="原价（元）" min={0} fieldProps={{ precision: 2 }} />
            </>
          )}
        </StepsForm.StepForm>
      </StepsForm>
    </Card>
  )
}
