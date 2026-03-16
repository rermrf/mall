import { useEffect, useState, useCallback } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, message, Button } from 'antd'
import { ProForm, ProFormText, ProFormTextArea, ProFormDigit, ProFormSelect, ProFormList, StepsForm, EditableProTable } from '@ant-design/pro-components'
import type { ProColumns } from '@ant-design/pro-components'
import { createProduct, updateProduct, getProduct, listCategories, listBrands } from '@/api/product'
import type { Category, Brand, CreateProductReq, CreateSKUReq, ProductSpec } from '@/types/product'
import { silentApiError } from '@/utils/error'

interface BasicFormValues {
  name: string
  subtitle?: string
  categoryId: number
  brandId?: number
  mainImage?: string
  description?: string
  price?: number
  originalPrice?: number
}

export default function ProductForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id

  const [categories, setCategories] = useState<Category[]>([])
  const [brands, setBrands] = useState<Brand[]>([])
  const [initialValues, setInitialValues] = useState<Partial<BasicFormValues>>({})
  const [skus, setSkus] = useState<CreateSKUReq[]>([])
  const [specs, setSpecs] = useState<ProductSpec[]>([])
  const [editableKeys, setEditableKeys] = useState<React.Key[]>([])

  useEffect(() => {
    listCategories().then((c) => setCategories(c ?? [])).catch(silentApiError('productForm:listCategories'))
    listBrands({ page: 1, pageSize: 100 }).then((r) => setBrands(r?.brands ?? [])).catch(silentApiError('productForm:listBrands'))
    if (isEdit) {
      getProduct(Number(id)).then((p) => {
        if (p) {
          setInitialValues({
            name: p.name,
            subtitle: p.subtitle,
            categoryId: p.categoryId,
            brandId: p.brandId,
            mainImage: p.mainImage,
            description: p.description,
          })
          setSkus(p.skus?.map((s) => ({
            skuCode: s.skuCode,
            price: s.price,
            originalPrice: s.originalPrice,
            costPrice: s.costPrice,
            barCode: s.barCode,
            specValues: s.specValues,
            status: s.status,
          })) ?? [])
          setSpecs(p.specs ?? [])
        }
      }).catch(silentApiError('productForm:getProduct'))
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
      // Try to find an existing SKU with the same specValues to preserve its data
      const existing = skus.find((s) => s.specValues === specValues)
      return {
        skuCode: existing?.skuCode || `SKU-${idx + 1}`,
        price: existing?.price ?? 0,
        originalPrice: existing?.originalPrice ?? 0,
        costPrice: existing?.costPrice ?? 0,
        barCode: existing?.barCode || '',
        specValues: specValues,
        status: existing?.status ?? 1,
      }
    })
  }, [skus])

  const handleSpecsChange = (rawSpecs: Array<{ specName?: string; specValues?: string }> | undefined) => {
    const parsed: ProductSpec[] = (rawSpecs ?? [])
      .filter((r) => r.specName)
      .map((r) => ({
        name: r.specName!,
        values: (r.specValues || '').split(',').map((v) => v.trim()).filter(Boolean),
      }))
    setSpecs(parsed)
    const newSkus = generateSkuCombinations(parsed)
    setSkus(newSkus)
    setEditableKeys(newSkus.map((_, i) => i))
  }

  const skuColumns: ProColumns<CreateSKUReq & { id?: number }>[] = [
    { title: '规格值', dataIndex: 'specValues', editable: false, width: 160 },
    { title: '价格（分）', dataIndex: 'price', valueType: 'digit', width: 120 },
    { title: '原价（分）', dataIndex: 'originalPrice', valueType: 'digit', width: 120 },
    { title: '成本价（分）', dataIndex: 'costPrice', valueType: 'digit', width: 120 },
    { title: '条码', dataIndex: 'barCode', width: 140 },
    { title: 'SKU编码', dataIndex: 'skuCode', width: 140 },
    { title: '操作', valueType: 'option' },
  ]

  const handleSubmit = async (values: BasicFormValues) => {
    const data: CreateProductReq = {
      categoryId: values.categoryId,
      brandId: values.brandId ?? 0,
      name: values.name,
      subtitle: values.subtitle || '',
      mainImage: values.mainImage || '',
      images: [],
      description: values.description || '',
      status: 0,
      skus: skus.length > 0 ? skus : [{
        skuCode: 'DEFAULT',
        price: (values.price ?? 0) * 100,
        originalPrice: (values.originalPrice ?? 0) * 100,
        costPrice: 0,
        barCode: '',
        specValues: '',
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
          <ProFormSelect name="categoryId" label="分类" rules={[{ required: true }]} options={flatCategories(categories)} />
          <ProFormSelect name="brandId" label="品牌" options={brands.map((b) => ({ label: b.name, value: b.id }))} />
          <ProFormText name="mainImage" label="主图URL" />
          <ProFormTextArea name="description" label="商品描述" />
        </StepsForm.StepForm>
        <StepsForm.StepForm name="sku" title="价格库存">
          <ProFormList
            name="specList"
            label="商品规格"
            creatorButtonProps={{ creatorButtonText: '添加规格' }}
            initialValue={specs.map((s) => ({ specName: s.name, specValues: s.values.join(',') }))}
            actionRender={(_field: { name: number }, action: { remove: (index: number) => void }) => [
              <Button key="delete" type="link" danger onClick={() => action.remove(_field.name)}>
                删除
              </Button>,
            ]}
          >
            <ProForm.Group key="spec-group">
              <ProFormText name="specName" label="规格名" placeholder="如：颜色" rules={[{ required: true }]} />
              <ProFormText name="specValues" label="规格值（逗号分隔）" placeholder="如：红,蓝,绿" rules={[{ required: true }]} />
            </ProForm.Group>
          </ProFormList>

          {specs.length > 0 && skus.length > 0 ? (
            <EditableProTable<CreateSKUReq & { id?: number }>
              headerTitle="SKU列表"
              rowKey="specValues"
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
              <ProFormDigit name="originalPrice" label="原价（元）" min={0} fieldProps={{ precision: 2 }} />
            </>
          )}
        </StepsForm.StepForm>
      </StepsForm>
    </Card>
  )
}
