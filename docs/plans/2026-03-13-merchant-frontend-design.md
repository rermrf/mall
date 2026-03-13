# 商家端前端设计文档

## 概述

为 SaaS 多租户商城平台创建商家管理后台前端（`merchant-frontend/`），对接已有的 merchant-bff（57+ API 端点），提供商品管理、订单处理、库存管理、营销活动、物流配置、团队管理等完整的商家运营功能。

## 决策记录

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 目标平台 | PC 桌面端 | 商家管理需要大屏幕操作复杂表单和数据 |
| 功能范围 | 全量（所有模块） | 后端 API 已就绪，一次到位 |
| 项目位置 | 同一 mono-repo | 与现有 11 个微服务 + 3 个 BFF + 消费者前端保持统一管理 |
| 技术栈 | React 19 + TS + Vite + Zustand + Axios | 与消费者端一致，降低维护成本 |
| UI 组件库 | Ant Design 5 + ProComponents | 桌面端最佳选择，ProTable/ProForm 加速 CRUD |
| 架构模式 | 标准 SPA + 侧边栏布局 | 经典管理后台模式，Ant Design ProLayout 直接支持 |

## 技术栈

| 层 | 选型 | 版本 |
|---|---|---|
| 框架 | React + TypeScript | 19.x / 5.x |
| 构建 | Vite | 7.x |
| UI | Ant Design + ProComponents | 5.x |
| 状态管理 | Zustand | 5.x |
| HTTP | Axios | 1.x |
| 路由 | React Router | 7.x |

## 项目结构

```
merchant-frontend/
├── src/
│   ├── api/                # API 模块
│   │   ├── request.ts      # Axios 实例 + 拦截器
│   │   ├── auth.ts         # 登录/登出/刷新 token
│   │   ├── shop.ts         # 店铺信息/配额
│   │   ├── product.ts      # 商品/分类/品牌 CRUD
│   │   ├── inventory.ts    # 库存查询/设置/日志
│   │   ├── order.ts        # 订单列表/详情/发货/退款
│   │   ├── payment.ts      # 支付/退款查询
│   │   ├── marketing.ts    # 优惠券/秒杀/促销
│   │   ├── logistics.ts    # 运费模板 CRUD
│   │   ├── staff.ts        # 员工/角色管理
│   │   └── notification.ts # 通知
│   ├── pages/
│   │   ├── login/          # 登录页
│   │   ├── dashboard/      # 仪表盘首页
│   │   ├── product/        # 商品管理
│   │   │   ├── ProductList.tsx
│   │   │   ├── ProductForm.tsx
│   │   │   ├── CategoryList.tsx
│   │   │   └── BrandList.tsx
│   │   ├── order/          # 订单管理
│   │   │   ├── OrderList.tsx
│   │   │   ├── OrderDetail.tsx
│   │   │   └── RefundList.tsx
│   │   ├── inventory/      # 库存管理
│   │   │   ├── StockList.tsx
│   │   │   └── StockLog.tsx
│   │   ├── marketing/      # 营销管理
│   │   │   ├── CouponList.tsx
│   │   │   ├── CouponForm.tsx
│   │   │   ├── SeckillList.tsx
│   │   │   ├── SeckillForm.tsx
│   │   │   └── PromotionList.tsx
│   │   ├── logistics/      # 物流管理
│   │   │   ├── TemplateList.tsx
│   │   │   └── TemplateForm.tsx
│   │   ├── shop/           # 店铺设置
│   │   │   └── ShopSettings.tsx
│   │   ├── staff/          # 团队管理
│   │   │   ├── StaffList.tsx
│   │   │   └── RoleList.tsx
│   │   └── notification/   # 消息中心
│   │       └── NotificationList.tsx
│   ├── components/
│   │   └── layout/         # ProLayout 主布局
│   │       └── MainLayout.tsx
│   ├── stores/
│   │   ├── auth.ts         # 登录态 (token, user info)
│   │   └── notification.ts # 未读通知计数
│   ├── router/
│   │   └── index.tsx       # 路由配置 + 守卫
│   ├── types/              # TypeScript 类型
│   │   ├── api.ts          # 通用响应类型
│   │   ├── product.ts
│   │   ├── order.ts
│   │   ├── marketing.ts
│   │   └── ...
│   ├── App.tsx
│   └── main.tsx
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── .env.development        # VITE_API_BASE_URL 等
```

## 功能模块设计

### 1. 认证 (Auth)

- 登录页：账号 + 密码表单
- JWT 机制：与消费者端完全一致
  - Access Token (30min) + Refresh Token (7d)
  - 请求头注入 `Authorization: Bearer {token}`
  - 响应头读取 `X-Jwt-Token` / `X-Refresh-Token`
  - 401 自动刷新，队列重试
- 路由守卫：未登录跳转登录页，已登录访问登录页跳转仪表盘

### 2. 仪表盘 (Dashboard)

- 概览卡片：今日订单数、今日销售额、待发货、待退款
- 快捷入口：常用功能跳转
- 注：数据来源于现有订单/支付 API 的聚合查询

### 3. 商品管理 (Product)

- **商品列表**: ProTable 分页 + 按分类/状态筛选 + 搜索
- **商品创建/编辑**: ProForm 分步表单
  - Step 1: 基本信息（名称、分类、品牌、描述、图片）
  - Step 2: 规格/SKU 设置（动态规格定义 + SKU 组合）
  - Step 3: 价格/库存（各 SKU 售价、成本价、库存量）
- **状态管理**: 上架/下架切换
- **分类管理**: 树形列表 + CRUD
- **品牌管理**: ProTable 列表 + CRUD

### 4. 订单管理 (Order)

- **订单列表**: ProTable 按状态 tab 筛选（全部/待付款/待发货/已发货/已完成/已关闭）
- **订单详情**: 订单信息 + 商品明细 + 收货地址 + 物流信息 + 操作按钮
- **发货操作**: 填写物流公司 + 运单号
- **退款列表**: 独立列表，审核通过/拒绝操作

### 5. 库存管理 (Inventory)

- **库存列表**: 按 SKU 显示当前库存、预警值
- **设置库存**: 修改指定 SKU 的库存量
- **批量查询**: 批量获取多个 SKU 库存
- **变更日志**: 库存变动记录（入库/出库/扣减/回滚）

### 6. 营销管理 (Marketing)

- **优惠券**: 创建（满减/折扣/固定金额）、编辑、列表（ProTable）
- **秒杀活动**: 创建（选择商品/时间/限量）、编辑、列表
- **促销规则**: 创建（满减/满赠/组合优惠）、编辑、列表

### 7. 物流管理 (Logistics)

- **运费模板列表**: ProTable 列表 + CRUD
- **模板编辑**: 区域规则配置（首重/续重、免邮条件）

### 8. 店铺设置 (Shop)

- 店铺信息编辑：名称、Logo、描述、子域名、自定义域名
- 配额查看：当前 SaaS 套餐的各项限额

### 9. 团队管理 (Staff)

- **员工列表**: ProTable + 分页
- **角色分配**: 为员工指定角色
- **角色管理**: 角色 CRUD（名称/权限集合）

### 10. 消息中心 (Notification)

- 通知列表：ProTable 分页
- 标记已读 / 全部已读
- 顶部导航栏显示未读 badge

## 布局设计

Ant Design ProLayout 经典管理后台布局：

```
┌─────────────────────────────────────────────────────┐
│  Logo  商家名称         🔔 3  [头像] 退出            │
├──────────┬──────────────────────────────────────────┤
│          │  首页 > 商品管理 > 商品列表               │
│ 📊 仪表盘 │──────────────────────────────────────────│
│ 📦 商品   │                                          │
│   商品列表│           页面内容区域                     │
│   分类管理│                                          │
│   品牌管理│                                          │
│ 📋 订单   │                                          │
│ 📦 库存   │                                          │
│ 🎯 营销   │                                          │
│ 🚚 物流   │                                          │
│ 🏪 店铺   │                                          │
│ 👥 团队   │                                          │
│ 🔔 消息   │                                          │
└──────────┴──────────────────────────────────────────┘
```

## HTTP 客户端

复用消费者端的 Axios 模式，配置独立的 base URL：

- Vite dev proxy: `/api/v1` → `http://localhost:8281`（merchant-bff 端口）
- 请求拦截器：注入 Bearer token
- 响应拦截器：统一错误处理 + 401 token 刷新
- 统一响应类型：`ApiResponse<T> = { code: number; msg: string; data: T }`

## 路由设计

```typescript
/login                          // 登录页（公开）
/                               // 仪表盘（需登录）
/product/list                   // 商品列表
/product/create                 // 创建商品
/product/edit/:id               // 编辑商品
/product/category               // 分类管理
/product/brand                  // 品牌管理
/order/list                     // 订单列表
/order/:orderNo                 // 订单详情
/order/refund                   // 退款列表
/inventory                      // 库存管理
/inventory/log                  // 库存日志
/marketing/coupon               // 优惠券列表
/marketing/coupon/create        // 创建优惠券
/marketing/coupon/edit/:id      // 编辑优惠券
/marketing/seckill              // 秒杀列表
/marketing/seckill/create       // 创建秒杀
/marketing/seckill/edit/:id     // 编辑秒杀
/marketing/promotion            // 促销规则
/logistics/template             // 运费模板列表
/logistics/template/create      // 创建模板
/logistics/template/edit/:id    // 编辑模板
/shop/settings                  // 店铺设置
/staff/list                     // 员工列表
/staff/role                     // 角色管理
/notification                   // 消息中心
```

## 状态管理

Zustand stores：

- **authStore**: token、userInfo、login/logout actions
- **notificationStore**: unreadCount、fetchUnreadCount action（轮询或定时刷新）

页面级数据通过 ProTable/ProForm 自带的请求管理，不需要额外的全局 store。

## 错误处理

- API 层：统一拦截 `code !== 0` 的响应，message.error 提示
- 401：自动刷新 token，失败后清除登录态跳转登录页
- 网络错误：统一提示网络异常
- 表单验证：Ant Design Form 内置校验
