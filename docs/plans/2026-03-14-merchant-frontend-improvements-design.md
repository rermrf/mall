# 商家前端完善设计文档

## 概述

对已有 merchant-frontend 进行三轮改进：修复关键缺陷、补齐缺失页面、完善 UX 体验。

## Round 1: 关键缺陷修复

### 1.1 CouponForm 补全字段
- 添加 `scope_type` Select（全场=0 / 指定商品=1 / 指定分类=2）
- 添加 `scope_ids` 输入（逗号分隔 ID）
- 编辑模式通过 `listCoupons` 查找已有数据回填

### 1.2 ProductForm 多 SKU/规格管理
- 动态规格定义：ProFormList 添加规格名 + 规格值（逗号分隔）
- 基于规格组合自动生成 SKU 行
- EditableProTable 编辑每个 SKU 的价格/原价/成本价/条码
- 无规格时保留单 SKU 模式

### 1.3 SeckillForm 秒杀商品管理
- ProFormList 管理秒杀商品项
- 每项字段：SKU ID、秒杀价、秒杀库存、每人限购
- 编辑模式通过 `getSeckill` 加载回填

### 1.4 OrderDetail 状态修复
- 用 statusMap 显示中文 Tag（与 OrderList 一致）
- 添加 Spin 加载状态

## Round 2: 缺失页面

### 2.1 PaymentList
- ProTable：支付单号、订单号、金额、状态、渠道、时间
- 状态筛选、分页
- 操作：查看详情

### 2.2 PaymentDetail
- Descriptions 展示支付信息
- 退款操作 Modal（金额 + 原因）
- 关联退款记录展示

### 2.3 ProfileEdit
- 展示当前用户信息
- ProForm 编辑昵称、头像

### 2.4 路由更新
- `/payment` → PaymentList
- `/payment/:paymentNo` → PaymentDetail
- `/profile` → ProfileEdit
- 侧边栏菜单添加支付管理入口

## Round 3: UX 体验完善

### 3.1 ErrorBoundary
- 全局错误边界组件，捕获渲染异常
- 显示友好错误页面 + 重试按钮

### 3.2 加载状态
- OrderDetail / ShopSettings 添加 Spin 包裹
- 所有 ModalForm 添加 try/catch + message.error

### 3.3 统一错误处理
- api/client.ts 响应拦截器添加 message.error 自动提示非 401 错误
- 页面级移除重复的 error toast
