# 跨服务联通补齐设计

## 概述

审计发现项目中存在以下联通缺失：merchant-bff 缺少商品管理（product-svc）集成、admin-bff 缺少 4 个服务集成、3 个 Kafka 事件无消费者、notification/marketing 的 consumer handler 不完整。本设计覆盖全部缺失项的补齐方案。

## 第一部分：merchant-bff 商品管理集成

### 缺失内容

merchant-bff 缺少 product-svc 客户端和 11 个商品/分类/品牌管理端点。

### 方案

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `merchant-bff/handler/product.go` | 新建 | ProductHandler + 11 方法 |
| 2 | `merchant-bff/ioc/grpc.go` | 修改 | +InitProductClient |
| 3 | `merchant-bff/ioc/gin.go` | 修改 | +productHandler 参数 + 11 路由 |
| 4 | `merchant-bff/wire.go` | 修改 | +ioc.InitProductClient, handler.NewProductHandler |
| 5 | `merchant-bff/wire_gen.go` | 重新生成 | Wire |

### 端点清单（11 个）

| HTTP | 路由 | 说明 | RPC |
|------|------|------|-----|
| POST | `/api/v1/products` | 创建商品 | CreateProduct |
| PUT | `/api/v1/products/:id` | 更新商品 | UpdateProduct |
| GET | `/api/v1/products/:id` | 商品详情 | GetProduct |
| GET | `/api/v1/products` | 商品列表 | ListProducts |
| PUT | `/api/v1/products/:id/status` | 上下架 | UpdateProductStatus |
| POST | `/api/v1/categories` | 创建分类 | CreateCategory |
| PUT | `/api/v1/categories/:id` | 更新分类 | UpdateCategory |
| GET | `/api/v1/categories` | 分类列表 | ListCategories |
| POST | `/api/v1/brands` | 创建品牌 | CreateBrand |
| PUT | `/api/v1/brands/:id` | 更新品牌 | UpdateBrand |
| GET | `/api/v1/brands` | 品牌列表 | ListBrands |

---

## 第二部分：admin-bff 缺失服务集成

### 缺失内容

admin-bff 只有 user-svc 和 tenant-svc 两个客户端。设计要求 6 个，缺少 product-svc、order-svc、payment-svc、notification-svc。

### 方案

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `admin-bff/handler/product.go` | 新建 | 分类管理(3) + 品牌管理(3) = 6 端点 |
| 2 | `admin-bff/handler/order.go` | 新建 | 订单监管 2 端点 |
| 3 | `admin-bff/handler/payment.go` | 新建 | 支付监管 2 端点 |
| 4 | `admin-bff/handler/notification.go` | 新建 | 模板管理(4) + 发送(1) = 5 端点 |
| 5 | `admin-bff/ioc/grpc.go` | 修改 | +InitProductClient, InitOrderClient, InitPaymentClient, InitNotificationClient |
| 6 | `admin-bff/ioc/gin.go` | 修改 | +4 handler 参数 + 15 路由 |
| 7 | `admin-bff/wire.go` | 修改 | +4 client + 4 handler |
| 8 | `admin-bff/wire_gen.go` | 重新生成 | Wire |

### 端点清单（15 个）

| 分组 | HTTP | 路由 | 说明 |
|------|------|------|------|
| 分类 | POST | `/api/v1/categories` | 创建平台分类(tenant_id=0) |
| 分类 | PUT | `/api/v1/categories/:id` | 更新分类 |
| 分类 | GET | `/api/v1/categories` | 分类列表 |
| 品牌 | POST | `/api/v1/brands` | 创建品牌(tenant_id=0) |
| 品牌 | PUT | `/api/v1/brands/:id` | 更新品牌 |
| 品牌 | GET | `/api/v1/brands` | 品牌列表 |
| 订单 | GET | `/api/v1/orders` | 全平台订单列表 |
| 订单 | GET | `/api/v1/orders/:orderNo` | 订单详情 |
| 支付 | GET | `/api/v1/payments/:paymentNo` | 支付详情 |
| 支付 | GET | `/api/v1/refunds/:refundNo` | 退款详情 |
| 通知 | POST | `/api/v1/notification-templates` | 创建通知模板 |
| 通知 | PUT | `/api/v1/notification-templates/:id` | 更新通知模板 |
| 通知 | GET | `/api/v1/notification-templates` | 模板列表 |
| 通知 | DELETE | `/api/v1/notification-templates/:id` | 删除模板 |
| 通知 | POST | `/api/v1/notifications/send` | 发送通知 |

---

## 第三部分：Kafka 未消费事件补齐

### 3.1 tenant_plan_changed → notification-svc

| 项 | 值 |
|----|-----|
| Topic | `tenant_plan_changed` |
| 生产者 | tenant-svc（UpdateTenant 套餐变更时） |
| 事件结构 | `{TenantId, OldPlanId, NewPlanId}` |
| 消费者 | notification-svc 新增 TenantPlanChangedConsumer |
| 动作 | 站内信+邮件通知商家套餐已变更 |
| 模板编码 | `tenant_plan_changed_inapp`, `tenant_plan_changed_email` |

文件变更：
- `notification/events/types.go` — +TenantPlanChangedEvent
- `notification/events/consumer.go` — +TenantPlanChangedConsumer
- `notification/ioc/kafka.go` — +NewTenantPlanChangedConsumer + 更新 InitConsumers

### 3.2 order_completed → notification-svc

| 项 | 值 |
|----|-----|
| Topic | `order_completed` |
| 生产者 | order-svc（确认收货时） |
| 事件结构 | `{OrderNo, TenantID, Items[{ProductID, Quantity}]}` |
| 消费者 | notification-svc 新增 OrderCompletedConsumer |
| 动作 | 站内信通知买家订单已完成（需 TODO 获取买家 ID） |
| 模板编码 | `order_completed_inapp` |

文件变更：同上模式

### 3.3 seckill_success → order-svc

| 项 | 值 |
|----|-----|
| Topic | `seckill_success` |
| 生产者 | marketing-svc（秒杀成功时） |
| 事件结构 | `{UserId, ItemId, SkuId, SeckillPrice, TenantId}` |
| 消费者 | order-svc 新增 SeckillSuccessConsumer |
| 动作 | TODO: 自动创建秒杀订单（需跨服务获取商品详情和收货地址） |
| 当前实现 | log + TODO 标记 |

文件变更：
- `order/events/types.go` — +SeckillSuccessEvent
- `order/events/consumer.go` — +SeckillSuccessConsumer
- `order/ioc/kafka.go` — +NewSeckillSuccessConsumer + 更新 InitConsumers
- `order/app.go` — 如果当前无 Consumers 字段则需添加
- `order/wire.go` — +consumer 相关 providers

---

## 第四部分：Notification TODO 修复 + Marketing handler 补全

### 4.1 notification-svc order_paid consumer 完善

当前状态：仅 log + TODO
修复方案：用 `tenantId` 从事件中推断（OrderPaidEvent 无 tenant_id 字段，需 TODO 标记跨服务查询）
文件：`notification/ioc/kafka.go`

### 4.2 notification-svc order_shipped consumer 完善

当前状态：仅 log + TODO
修复方案：同上，事件缺少 buyer_id，保留 TODO 标记
文件：`notification/ioc/kafka.go`

### 4.3 marketing-svc order_cancelled handler 补全

当前状态：仅 log，未释放优惠券
修复方案：调用 `svc.ReleaseCoupon` + TODO 标记需要从 order-svc 获取 CouponID
文件：`marketing/ioc/kafka.go`

---

## 文件清单汇总

| # | 文件路径 | 操作 | 属于 |
|---|---------|------|------|
| 1 | `merchant-bff/handler/product.go` | 新建 | 第一部分 |
| 2 | `merchant-bff/ioc/grpc.go` | 修改 | 第一部分 |
| 3 | `merchant-bff/ioc/gin.go` | 修改 | 第一部分 |
| 4 | `merchant-bff/wire.go` | 修改 | 第一部分 |
| 5 | `merchant-bff/wire_gen.go` | 重新生成 | 第一部分 |
| 6 | `admin-bff/handler/product.go` | 新建 | 第二部分 |
| 7 | `admin-bff/handler/order.go` | 新建 | 第二部分 |
| 8 | `admin-bff/handler/payment.go` | 新建 | 第二部分 |
| 9 | `admin-bff/handler/notification.go` | 新建 | 第二部分 |
| 10 | `admin-bff/ioc/grpc.go` | 修改 | 第二部分 |
| 11 | `admin-bff/ioc/gin.go` | 修改 | 第二部分 |
| 12 | `admin-bff/wire.go` | 修改 | 第二部分 |
| 13 | `admin-bff/wire_gen.go` | 重新生成 | 第二部分 |
| 14 | `notification/events/types.go` | 修改 | 第三部分 |
| 15 | `notification/events/consumer.go` | 修改 | 第三部分 |
| 16 | `notification/ioc/kafka.go` | 修改 | 第三/四部分 |
| 17 | `notification/wire.go` | 修改 | 第三部分 |
| 18 | `notification/wire_gen.go` | 重新生成 | 第三部分 |
| 19 | `order/events/types.go` | 修改 | 第三部分 |
| 20 | `order/events/consumer.go` | 新建或修改 | 第三部分 |
| 21 | `order/ioc/kafka.go` | 修改 | 第三部分 |
| 22 | `order/app.go` | 可能修改 | 第三部分 |
| 23 | `order/wire.go` | 修改 | 第三部分 |
| 24 | `order/wire_gen.go` | 重新生成 | 第三部分 |
| 25 | `marketing/ioc/kafka.go` | 修改 | 第四部分 |

共 25 个文件（5 新建 + 15 修改 + 5 重新生成）
