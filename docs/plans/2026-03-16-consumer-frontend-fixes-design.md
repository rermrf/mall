# Consumer Frontend 全面修复设计

**日期**: 2026-03-16
**范围**: frontend/ (Consumer 前端) + consumer-bff (后端补充接口)
**策略**: 分层修复 — 3 轮递进，每轮独立可验证

---

## Round 1: 严重问题修复

### 1.1 商品详情页直接访问 (Critical)

**问题**: 商品数据仅通过路由 state 传递，直接访问 `/product/:id` 时页面崩溃。

**后端改动**:
- consumer-bff 新增 `GET /api/v1/products/:id`
- 调用 product service 的 `GetProduct` RPC
- 返回完整商品信息（名称、描述、价格、图片列表、品牌、分类）

**前端改动**:
- `api/product.ts`: 新增 `getProductDetail(id)` 方法
- `pages/product/Detail.tsx`:
  - 优先使用路由 state 数据（避免重复请求）
  - state 为空时通过 `:id` 参数调 API 获取
  - 添加 loading skeleton 和错误状态（"商品不存在" / "加载失败，点击重试"）

### 1.2 确认订单页无地址处理 (Critical)

**问题**: 用户没有地址时只显示"请选择收货地址"，无法新增。

**前端改动** (`pages/order/Confirm.tsx`):
- 地址列表为空时显示引导区域 + "去添加" 按钮
- 跳转 `/me/addresses/edit`，路由 state 传递 `from: 'order-confirm'`
- 地址保存后检测来源自动 `navigate(-1)` 返回并刷新

### 1.3 购物车库存校验 (Critical)

**问题**: 修改数量不校验库存，可能超出实际库存。

**前端改动** (`pages/cart/index.tsx` + `stores/cart.ts`):
- 进入购物车时批量调用 `POST /api/v1/inventory/stock/batch`
- 增加数量时校验不超过库存上限，达上限 disable "+" 按钮
- 减少数量最低为 1，达最低 disable "-" 按钮
- 结算前再次校验库存，不足的商品标红提示

---

## Round 2: 核心体验改进

### 2.1 搜索筛选与排序

**前端改动** (`pages/search/index.tsx`):
- 搜索结果区域顶部增加排序栏：综合 | 销量 | 价格↑ | 价格↓
- 增加筛选抽屉（antd-mobile Popup）：分类选择、品牌选择、价格区间
- 筛选/排序变化时重置分页重新请求
- 对应后端已有参数：`sortBy`, `categoryId`, `brandId`, `priceMin`, `priceMax`

### 2.2 地址省市区级联选择器

**前端改动** (`pages/user/AddressEdit.tsx`):
- 省市区文本输入替换为 antd-mobile `CascadePicker`
- 省市区数据使用中国行政区划静态 JSON（前端内置，~30KB gzip）

### 2.3 全局加载状态补齐

为以下页面添加 antd-mobile `Skeleton` 骨架屏：
- 首页、搜索页、购物车、优惠券页、秒杀页、退款列表

列表加载更多使用 `DotLoading` 指示器。

### 2.4 "立即购买" 独立流程

**前端改动**:
- `pages/product/Detail.tsx`: "立即购买" 跳转 `/order/confirm`，路由 state 传 `{ directBuy: true, product, quantity }`
- `pages/order/Confirm.tsx`: 检测 `directBuy` 模式时不从购物车读取，使用直传商品数据

### 2.5 商品详情页增强

**前端改动** (`pages/product/Detail.tsx`):
- 图片轮播（antd-mobile `Swiper`）展示多图
- 商品描述区域
- 库存不足时禁用操作按钮
- 数量选择器（Stepper），上限为库存数

---

## Round 3: 完善细节

### 3.1 通知功能增强

**后端**: consumer-bff 新增 `DELETE /api/v1/notifications/:id`

**前端** (`pages/notification/index.tsx`):
- 通知类型筛选 Tab（全部 | 系统 | 订单 | 营销）
- 左滑删除（SwipeAction）
- 点击展开详情（Collapse 展开/收起）

### 3.2 退款取消功能

**后端**: consumer-bff 新增 `POST /api/v1/refunds/:refundNo/cancel`

**前端** (`pages/order/RefundDetail.tsx`):
- "待审核" 状态显示 "取消退款" 按钮
- 确认对话框 → 调用取消接口 → 刷新状态

### 3.3 多 Tab 登录状态同步

**前端** (`stores/auth.ts`):
- 监听 `storage` 事件检测 `accessToken` 变化
- Token 被清除时更新 store 并跳转登录页

### 3.4 购物车乐观更新

**前端** (`stores/cart.ts`):
- 操作先更新本地 state，再发 API
- API 失败回滚 + 提示

### 3.5 优惠券领取后自动刷新

**前端** (`pages/marketing/Coupons.tsx`):
- 领取成功后重新拉取可领列表和我的优惠券列表

### 3.6 UX 细节打磨

| 改进 | 页面 |
|------|------|
| 空购物车 "去逛逛" 按钮 → 首页 | Cart |
| 订单号一键复制 | OrderDetail |
| Tab 切换重置滚动位置 | OrderList |
| OAuth 按钮置灰 + "即将开放" | Login |
| 手机号 11 位格式校验 | Signup, AddressEdit |
| 秒杀倒计时服务端时间校准 | Seckill |
| 退款按钮检查已有退款记录 | OrderDetail |
| 错误提示优化为具体文案 | 全局 API Client |

---

## 后端新增接口汇总

| 接口 | 方法 | Round |
|------|------|-------|
| `/api/v1/products/:id` | GET | 1 |
| `/api/v1/notifications/:id` | DELETE | 3 |
| `/api/v1/refunds/:refundNo/cancel` | POST | 3 |
