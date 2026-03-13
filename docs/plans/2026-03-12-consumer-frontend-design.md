# Consumer 消费者商城前端设计

## 概述

为 SaaS 多租户电商平台的 C 端消费者构建 Mobile-First 响应式商城，对接 consumer-bff 的 53 个 API 端点。

## 技术栈

| 层级 | 技术选型 | 版本 |
|------|---------|------|
| 构建工具 | Vite | 6.x |
| 框架 | React | 19.x |
| 语言 | TypeScript | 5.x |
| UI 组件库 | Ant Design Mobile | 5.x |
| 状态管理 | Zustand | 5.x |
| 路由 | React Router | 7.x |
| HTTP 客户端 | Axios | 1.x |
| CSS | antd-mobile 内置 + CSS Modules |
| 平台 | Mobile-First 响应式 |

## 项目结构

```
frontend/
├── public/
├── src/
│   ├── api/                ← API 层：按模块分文件
│   │   ├── client.ts       ← Axios 实例 + JWT 拦截器
│   │   ├── auth.ts         ← 登录/注册/刷新 token
│   │   ├── user.ts         ← 用户信息/地址
│   │   ├── product.ts      ← 商品/搜索
│   │   ├── cart.ts         ← 购物车 CRUD
│   │   ├── order.ts        ← 订单/退款
│   │   ├── payment.ts      ← 支付
│   │   ├── marketing.ts    ← 优惠券/秒杀
│   │   ├── logistics.ts    ← 物流查询
│   │   └── notification.ts ← 通知消息
│   ├── stores/             ← Zustand stores
│   │   ├── auth.ts         ← 用户登录态、token 管理
│   │   ├── cart.ts         ← 购物车状态
│   │   └── notification.ts ← 未读消息数
│   ├── pages/              ← 页面组件（按功能模块）
│   │   ├── home/           ← 首页
│   │   ├── search/         ← 搜索页
│   │   ├── product/        ← 商品详情
│   │   ├── cart/           ← 购物车
│   │   ├── order/          ← 订单列表/详情/退款
│   │   ├── payment/        ← 支付页
│   │   ├── user/           ← 个人中心/地址管理
│   │   ├── auth/           ← 登录/注册
│   │   ├── marketing/      ← 秒杀/优惠券
│   │   └── notification/   ← 消息中心
│   ├── components/         ← 通用组件
│   │   ├── Layout/         ← TabBar 底部导航 + 页面骨架
│   │   ├── ProductCard/    ← 商品卡片（瀑布流/列表）
│   │   ├── Empty/          ← 空状态提示
│   │   └── Price/          ← 价格显示（划线价 + 实际价）
│   ├── hooks/              ← 自定义 hooks（useAuth, useCart 等）
│   ├── utils/              ← 工具函数（格式化、校验等）
│   ├── types/              ← TypeScript 类型定义
│   ├── styles/             ← 全局样式、主题变量覆盖
│   ├── router/
│   │   └── index.tsx       ← 路由配置（懒加载）
│   ├── App.tsx
│   └── main.tsx
├── index.html
├── vite.config.ts
├── tsconfig.json
└── package.json
```

## 页面规划

共 ~15 个页面，分 4 个 Tab + 独立页面。

### 底部 TabBar 导航

| Tab | 图标 | 页面 | 路由 |
|-----|------|------|------|
| 首页 | Home | 首页 | `/` |
| 分类 | Search | 搜索/分类 | `/search` |
| 购物车 | Cart | 购物车（带角标） | `/cart` |
| 我的 | User | 个人中心 | `/me` |

### 完整页面清单

| 模块 | 页面 | 路由 | 对接 API |
|------|------|------|---------|
| **首页** | 首页 | `/` | GET /shop, GET /seckill, GET /coupons |
| **搜索** | 搜索页 | `/search` | GET /search, /suggestions, /hot |
| **商品** | 商品详情 | `/product/:id` | GET /inventory/stock/:skuId, POST /cart/items |
| **购物车** | 购物车 | `/cart` | GET /cart, PUT/DELETE cart items |
| **订单** | 确认下单 | `/order/confirm` | POST /orders, GET /addresses, GET /coupons/mine |
| | 订单列表 | `/orders` | GET /orders |
| | 订单详情 | `/orders/:orderNo` | GET /orders/:orderNo, GET /logistics |
| | 退款列表 | `/refunds` | GET /refunds |
| | 退款详情 | `/refunds/:refundNo` | GET /refunds/:refundNo |
| **支付** | 支付页 | `/payment/:orderNo` | POST /payments, GET /payments/:paymentNo |
| **营销** | 秒杀列表 | `/seckill` | GET /seckill, POST /seckill/:itemId |
| | 优惠券中心 | `/coupons` | GET /coupons, GET /mine, POST /:id/receive |
| **用户** | 个人中心 | `/me` | GET /profile |
| | 编辑资料 | `/me/profile` | PUT /profile |
| | 地址管理 | `/me/addresses` | GET/POST/PUT/DELETE /addresses |
| | 消息中心 | `/notifications` | GET /notifications, PUT read |
| **认证** | 登录 | `/login` | POST /login, /login/phone, /sms/send |
| | 注册 | `/signup` | POST /signup |

## JWT 认证机制

### Token 管理

- Access Token (30min) 存 `localStorage` key: `access_token`
- Refresh Token (7d) 存 `localStorage` key: `refresh_token`
- 登录成功从响应头 `X-Jwt-Token` / `X-Refresh-Token` 提取

### Axios 拦截器

```
请求拦截器:
  ├─ 附加 Authorization: Bearer {accessToken}
  └─ 开发环境附加 X-Tenant-Domain: {configured_domain}

响应拦截器:
  ├─ 200 → 正常返回
  └─ 401 → 用 refreshToken 调 POST /refresh-token
       ├─ 成功 → 更新 token + 重试原请求
       └─ 失败 → 清除 token + 跳转 /login
```

### 路由守卫

- 需登录页面：检查 `access_token` 存在，不存在则跳转 `/login?redirect=xxx`
- 登录页：已登录则跳转首页

## 设计语言：简约精品风

### 色彩系统

| 用途 | 色值 | 说明 |
|------|------|------|
| 主色 | `#1A1A1A` | 几乎黑色，用于标题和主按钮 |
| 强调色 | `#C9A96E` | 金色，用于价格、促销标签、关键操作 |
| 背景色 | `#F8F8F8` | 极浅灰背景 |
| 卡片色 | `#FFFFFF` | 纯白卡片 |
| 次要文字 | `#999999` | 灰色辅助信息 |
| 分割线 | `#EEEEEE` | 极淡分割 |
| 成功色 | `#52C41A` | 支付成功、物流到达 |
| 警告色 | `#FF4D4F` | 库存紧张、订单异常 |

### 排版

| 要素 | 规范 |
|------|------|
| 字体 | `-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif` |
| 标题 | 16-20px, font-weight: 600 |
| 正文 | 14px, line-height: 1.6 |
| 辅助 | 12px, color: #999 |
| 价格 | font-weight: 700, color: #C9A96E |

### 视觉特征

- **大留白**：模块间距 16-24px，内容有呼吸感
- **无边框卡片**：商品卡片用轻阴影 `box-shadow: 0 1px 4px rgba(0,0,0,0.06)`
- **大尺寸商品图**：占满卡片宽度，纯色背景
- **圆角**：统一 8px 圆角
- **动效**：页面切换 slide 过渡 300ms，按钮点击 scale(0.98)
- **TabBar**：纯线条图标，选中态黑色 + 金色下划线

## 核心交互流程

```
首页（秒杀/优惠券/推荐）
  → 搜索/分类浏览
  → 商品详情（选 SKU）
  → 加入购物车
  → 购物车（勾选/编辑数量）
  → 确认下单（选地址 + 选优惠券）
  → 支付
  → 订单详情 → 查看物流
  → 确认收货 / 申请退款
```

## API 对接规范

### 请求/响应格式

后端统一返回格式：
```json
{
  "code": 0,
  "msg": "success",
  "data": { ... }
}
```

- `code === 0` 表示成功
- `code !== 0` 展示 `msg` 给用户（Toast）

### 多租户支持

- Consumer-BFF 通过 `TenantResolve` 中间件自动从域名解析 tenant_id
- 前端需确保请求发送到正确的域名（生产环境自然满足）
- 开发环境通过 `X-Tenant-Domain` 头或 Vite proxy 转发

### Vite 开发代理

```ts
// vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080', // consumer-bff
        changeOrigin: true,
        headers: { 'X-Tenant-Domain': 'shop1' },
      },
    },
  },
})
```

## 分期实施建议

### Phase 1：核心购物流程（MVP）
- 项目脚手架 + 路由 + 布局
- 登录/注册
- 首页 + 搜索
- 商品详情
- 购物车
- 下单 + 支付

### Phase 2：用户体验完善
- 个人中心 + 地址管理
- 订单管理（列表/详情/取消/确认收货）
- 物流查询
- 退款流程

### Phase 3：营销 + 通知
- 优惠券中心
- 秒杀活动
- 消息中心
