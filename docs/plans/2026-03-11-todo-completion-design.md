# TODO 全部补齐设计

## 概述

代码审计发现 15 个 TODO 分布在 8 个文件中，涉及 5 类未完成功能。本设计覆盖全部 15 个 TODO 的解决方案。

## A 类：Kafka Consumer 跨服务数据补齐（7 处）

### 问题

notification-svc 和 marketing-svc 的 Kafka consumer handler 缺少跨服务数据（买家ID、商家管理员ID、优惠券ID等），导致通知无法发送、优惠券无法释放。

### 方案

为 notification-svc 注入 order-svc gRPC 客户端，为 marketing-svc 注入 order-svc gRPC 客户端。在 consumer handler 中发起 gRPC 调用获取缺失数据。

| # | 文件 | Consumer | 注入客户端 | 调用链 |
|---|------|----------|-----------|--------|
| 1 | `notification/ioc/kafka.go:60` | order_paid | orderv1.OrderServiceClient | GetOrder(orderNo) → buyer_id + tenant_id → 发送站内信通知商家"有新订单已付款" |
| 2 | `notification/ioc/kafka.go:77` | order_shipped | orderv1.OrderServiceClient | GetOrder(orderId→orderNo) → buyer_id + receiver_phone → 发送 SMS+站内信通知买家"您的包裹已发出" |
| 3 | `notification/ioc/kafka.go:96` | inventory_alert | - | 保留 tenantId 作 userId（商家管理员ID=tenantId 是合理近似，TODO 注释改为说明） |
| 4 | `notification/ioc/kafka.go:115` | tenant_approved | - | 保留现状（同上，合理近似） |
| 5 | `notification/ioc/kafka.go:132` | tenant_plan_changed | - | 保留现状（同上） |
| 6 | `notification/ioc/kafka.go:150` | order_completed | orderv1.OrderServiceClient | GetOrder(orderNo) → buyer_id → 发送站内信通知买家"订单已完成，欢迎评价" |
| 7 | `marketing/ioc/kafka.go:53` | order_cancelled | orderv1.OrderServiceClient | GetOrder(orderNo) → coupon_id → svc.ReleaseCoupon(couponId) |

### 文件变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `notification/ioc/grpc.go` | 修改 | 添加 `initServiceConn` + `InitOrderClient` |
| `notification/ioc/kafka.go` | 修改 | 4 个 handler 函数签名添加 `orderv1.OrderServiceClient`，实现真实调用 |
| `notification/wire.go` | 修改 | thirdPartySet 添加 `ioc.InitOrderClient` |
| `notification/config/dev.yaml` | 修改 | 确认 etcd 配置已有（用于发现 order-svc） |
| `marketing/ioc/grpc.go` | 修改 | 添加 `initServiceConn` + `InitOrderClient` |
| `marketing/ioc/kafka.go` | 修改 | handler 签名添加 `orderv1.OrderServiceClient`，实现 ReleaseCoupon 调用 |
| `marketing/wire.go` | 修改 | thirdPartySet 添加 `ioc.InitOrderClient` |

---

## B 类：JWT Token 黑名单（3 BFF + 2 user-svc = 5 处）

### 问题

3 个 BFF 的 Logout 都是空实现（TODO: 将 token 加入黑名单）。user-svc 的 Logout 和 RefreshToken 也是 TODO。

### 架构分析

JWT 完全在 BFF 本地管理（生成+验证），user-svc 只负责认证（密码校验、OAuth）。因此黑名单也应在 BFF 本地 Redis 实现，不需要改 user-svc。

### 方案

**BFF 侧（admin-bff, merchant-bff, consumer-bff）**：

1. **新建 `ioc/redis.go`**：初始化 Redis 连接（3 个 BFF 共用同一 Redis 实例）
2. **JWTHandler 注入 `redis.Cmdable`**：
   - `SetTokenHeaders` 生成 JTI（uuid），写入 Claims
   - `Logout` 将 access token 和 refresh token 的 JTI 写入 Redis 黑名单
   - 黑名单 key: `jwt:blacklist:{jti}`，value: `1`，TTL = token 剩余有效期
3. **LoginJWTMiddleware 增加黑名单检查**：
   - 解析 token 后从 Claims 取 JTI
   - 查 Redis 是否存在 `jwt:blacklist:{jti}`
   - 存在则拒绝（token 已登出）

**user-svc 侧**：
- `Logout` — 移除 TODO，改为注释说明"黑名单由 BFF 管理"，return nil
- `RefreshToken` — 移除 TODO，改为注释说明"刷新由 BFF 管理"，return nil（不再返回错误）

### 文件变更（per BFF × 3）

| 文件 | 操作 | 说明 |
|------|------|------|
| `{bff}/ioc/redis.go` | 新建 | Viper 读 redis 配置，返回 `redis.Cmdable` |
| `{bff}/handler/jwt/handler.go` | 修改 | JWTHandler 添加 `redis.Cmdable` 字段，Logout 写黑名单，SetTokenHeaders 添加 JTI |
| `{bff}/handler/middleware/login_jwt.go` | 修改 | 注入 `redis.Cmdable`，Build() 中增加黑名单校验 |
| `{bff}/wire.go` | 修改 | 添加 `ioc.InitRedis` |
| `{bff}/config/dev.yaml` | 修改 | 添加 `redis.addr: localhost:6379` |
| `user/service/user.go` | 修改 | 更新 Logout 和 RefreshToken 注释 |

---

## C 类：OAuth 简化实现（1 处）

### 问题

`user/service/user.go:147` OAuthLogin 用 code 直接作为 provider_uid。

### 方案

保留当前实现。OAuth 对接需要真实第三方凭证（Google/GitHub OAuth App），是外部集成不在 MVP 范围。将 TODO 改为明确的 MVP 版本说明注释。

### 文件变更

| 文件 | 操作 |
|------|------|
| `user/service/user.go:147` | 修改注释 |

---

## D 类：user_registered consumer 空实现（1 处）

### 问题

`user/events/consumer.go:51` user-svc 自身消费 user_registered 事件但什么都不做。

### 方案

实际的 user_registered 消费者是 notification-svc（发欢迎短信+邮件），user-svc 自身的 consumer 用于内部后续扩展（如初始化默认用户偏好设置等）。将 TODO 改为说明注释，保留 consumer 结构供未来使用。

### 文件变更

| 文件 | 操作 |
|------|------|
| `user/events/consumer.go:51` | 修改注释 |

---

## E 类：秒杀自动创建订单（1 处）

### 问题

`order/ioc/kafka.go:83` seckill_success consumer 收到秒杀成功事件后不创建订单。

### 架构分析

order-svc 已有全部所需 gRPC 客户端：product-svc、user-svc、inventory-svc、payment-svc。`OrderService.CreateOrder` 接受 `CreateOrderReq{BuyerID, TenantID, Items, AddressID, CouponID}`。

### 方案

在 seckill_success consumer handler 中：
1. 调用 `userClient.ListAddresses` 获取用户默认收货地址
2. 构建 `CreateOrderReq` 使用秒杀价格
3. 调用 `svc.CreateOrder` 创建订单
4. 订单创建成功后日志记录

注意：秒杀订单不走优惠券（CouponID=0），直接使用 SeckillPrice 作为价格。但 CreateOrder 内部会从 product-svc 获取 SKU 价格，秒杀价格需要特殊处理。

最简方案：在 seckill consumer handler 中调用 `svc.CreateOrder`，让 CreateOrder 的现有逻辑处理。秒杀价格覆盖需要在 OrderService 添加一个 `CreateSeckillOrder` 方法或在 `CreateOrderReq` 中添加秒杀标记。

推荐方案：在 `CreateOrderReq` 中添加 `IsSeckill bool` + `SeckillPrice int64` 字段，`buildOrderItems` 中如果 IsSeckill 则使用 SeckillPrice 替代 SKU 原价。

### 文件变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `order/service/order.go` | 修改 | CreateOrderReq 添加 IsSeckill + SeckillPrice 字段，buildOrderItems 支持秒杀价 |
| `order/ioc/kafka.go` | 修改 | seckill handler 注入 userClient，实现自动创建订单逻辑 |
| `order/ioc/grpc.go` | 确认 | 已有 InitUserClient，无需修改 |
| `order/wire.go` | 确认 | 已有 ioc.InitUserClient，无需修改 |

---

## 文件变更汇总

| # | 文件 | 操作 | 类别 |
|---|------|------|------|
| 1 | `notification/ioc/grpc.go` | 修改 | A |
| 2 | `notification/ioc/kafka.go` | 修改 | A |
| 3 | `notification/wire.go` | 修改 | A |
| 4 | `marketing/ioc/grpc.go` | 修改 | A |
| 5 | `marketing/ioc/kafka.go` | 修改 | A |
| 6 | `marketing/wire.go` | 修改 | A |
| 7 | `admin-bff/ioc/redis.go` | 新建 | B |
| 8 | `admin-bff/handler/jwt/handler.go` | 修改 | B |
| 9 | `admin-bff/handler/middleware/login_jwt.go` | 修改 | B |
| 10 | `admin-bff/wire.go` | 修改 | B |
| 11 | `merchant-bff/ioc/redis.go` | 新建 | B |
| 12 | `merchant-bff/handler/jwt/handler.go` | 修改 | B |
| 13 | `merchant-bff/handler/middleware/login_jwt.go` | 修改 | B |
| 14 | `merchant-bff/wire.go` | 修改 | B |
| 15 | `consumer-bff/ioc/redis.go` | 新建 | B |
| 16 | `consumer-bff/handler/jwt/handler.go` | 修改 | B |
| 17 | `consumer-bff/handler/middleware/login_jwt.go` | 修改 | B |
| 18 | `consumer-bff/wire.go` | 修改 | B |
| 19 | `user/service/user.go` | 修改 | B+C |
| 20 | `user/events/consumer.go` | 修改 | D |
| 21 | `order/service/order.go` | 修改 | E |
| 22 | `order/ioc/kafka.go` | 修改 | E |

共 22 个文件（3 新建 + 19 修改），需重新生成 wire_gen.go 的有：notification, marketing, admin-bff, merchant-bff, consumer-bff（5 个）
