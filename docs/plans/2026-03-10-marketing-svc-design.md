# Marketing Service + BFF 设计

## 概述

实现 marketing-svc 营销微服务（16 个 gRPC RPC）+ Kafka Producer（秒杀成功）+ Kafka Consumer（订单取消释放优惠券）+ merchant-bff 营销管理接口（10 个端点）+ consumer-bff 营销消费者接口（5 个端点）。涵盖优惠券、秒杀活动、满减规则三大功能域。

## 架构决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 数据库 | MySQL (`mall_marketing`) | 优惠券、秒杀活动、满减规则持久化 |
| 缓存 | Redis | 优惠券库存计数、秒杀库存 Lua 原子扣减 |
| 秒杀架构 | Redis+Lua → Kafka 异步 | Lua 原子扣减库存 → `seckill_success` → order-svc 消费创建订单 |
| Kafka 角色 | Producer + Consumer | 生产 `seckill_success`；消费 `order_cancelled` 释放优惠券 |
| BFF 分布 | merchant-bff + consumer-bff | 商家管理 + C 端领券/秒杀 |
| gRPC 端口 | 8089 | |
| Redis DB | 8 | |
| 服务名 | marketing | |

## 数据模型

### MySQL 表（6 张）

| 表 | 关键字段 | 索引 |
|----|---------|------|
| `coupons` | id, tenant_id, name, type(1满减/2折扣/3无门槛), threshold, discount_value, total_count, received_count, used_count, per_limit, start_time, end_time, scope_type, scope_ids, status | `idx_tenant_status` |
| `user_coupons` | id, user_id, coupon_id, tenant_id, status(1未使用/2已使用/3已过期), order_id, receive_time, use_time | `uk_user_coupon(user_id,coupon_id)` 限制重复领取, `idx_user_tenant_status` |
| `seckill_activities` | id, tenant_id, name, start_time, end_time, status | `idx_tenant_status` |
| `seckill_items` | id, activity_id, tenant_id, sku_id, seckill_price, seckill_stock, per_limit | `idx_activity`, `uk_activity_sku` |
| `seckill_orders` | id, user_id, item_id, tenant_id, order_no, status | `uk_user_item` 防重复抢购 |
| `promotion_rules` | id, tenant_id, name, type, threshold, discount_value, start_time, end_time, status | `idx_tenant_status` |

### Redis 缓存策略

- 优惠券库存：`coupon:stock:{couponId}` — String，领券时 DECR 原子扣减
- 秒杀库存：`seckill:stock:{itemId}` — String，Lua 脚本原子扣减 + 防重复
- 秒杀用户记录：`seckill:user:{itemId}` — Set，SADD 判重

## marketing-svc RPC（16 个）

| RPC | 说明 | 实现方式 |
|-----|------|---------|
| CreateCoupon | 创建优惠券 | MySQL INSERT + Redis SET 库存 |
| UpdateCoupon | 更新优惠券 | MySQL UPDATE |
| ListCoupons | 优惠券列表 | MySQL 分页查询 |
| ReceiveCoupon | 领券 | Redis DECR 库存 → MySQL INSERT user_coupon |
| ListUserCoupons | 我的优惠券 | MySQL 查询 + JOIN coupon |
| UseCoupon | 使用优惠券（下单锁定） | MySQL UPDATE status=2 |
| ReleaseCoupon | 释放优惠券（取消） | MySQL UPDATE status=1 + Redis INCR 库存 |
| CalculateDiscount | 计算优惠 | 查优惠券+满减规则，计算折扣 |
| CreateSeckillActivity | 创建秒杀活动 | MySQL INSERT + Redis SET 库存 |
| UpdateSeckillActivity | 更新秒杀活动 | MySQL UPDATE |
| ListSeckillActivities | 秒杀活动列表 | MySQL 分页查询 |
| GetSeckillActivity | 秒杀活动详情 | MySQL 查询 + items |
| Seckill | 秒杀抢购 | Redis Lua(扣库存+防重) → Kafka `seckill_success` |
| CreatePromotionRule | 创建满减规则 | MySQL INSERT |
| UpdatePromotionRule | 更新满减规则 | MySQL UPDATE |
| ListPromotionRules | 满减规则列表 | MySQL 查询 |

## Kafka 事件

| Topic | 方向 | 事件 | 说明 |
|-------|------|------|------|
| `seckill_success` | 生产 | SeckillSuccessEvent{UserId, ItemId, SkuId, SeckillPrice, TenantId} | 秒杀成功 → order-svc 消费创建订单 |
| `order_cancelled` | 消费 | OrderCancelledEvent{OrderNo, UserId, CouponId, TenantId} | 订单取消 → 释放优惠券 |

## Merchant BFF 接口（10 个）

| HTTP 方法 | 路由 | 说明 | 认证 |
|-----------|------|------|------|
| POST | `/api/v1/coupons` | 创建优惠券 | 需登录 |
| PUT | `/api/v1/coupons/:id` | 更新优惠券 | 需登录 |
| GET | `/api/v1/coupons` | 优惠券列表 | 需登录 |
| POST | `/api/v1/seckill` | 创建秒杀活动 | 需登录 |
| PUT | `/api/v1/seckill/:id` | 更新秒杀活动 | 需登录 |
| GET | `/api/v1/seckill` | 秒杀活动列表 | 需登录 |
| GET | `/api/v1/seckill/:id` | 秒杀活动详情 | 需登录 |
| POST | `/api/v1/promotions` | 创建满减规则 | 需登录 |
| PUT | `/api/v1/promotions/:id` | 更新满减规则 | 需登录 |
| GET | `/api/v1/promotions` | 满减规则列表 | 需登录 |

> merchant-bff 所有路由都在 auth 路由组中（需登录 + 商家身份）。

## Consumer BFF 接口（5 个）

| HTTP 方法 | 路由 | 说明 | 认证 |
|-----------|------|------|------|
| GET | `/api/v1/coupons` | 可领优惠券列表 | 公开 |
| POST | `/api/v1/coupons/:id/receive` | 领券 | 需登录 |
| GET | `/api/v1/coupons/mine` | 我的优惠券 | 需登录 |
| GET | `/api/v1/seckill` | 秒杀活动列表 | 公开 |
| POST | `/api/v1/seckill/:itemId` | 秒杀抢购 | 需登录 |

> 优惠券列表和秒杀活动列表放 pub 路由组（无需登录），领券/秒杀/我的优惠券放 auth 路由组。

## 基础设施

- gRPC 端口：8089
- 服务名：marketing
- MySQL 库：mall_marketing
- Redis DB：8

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `marketing/domain/marketing.go` | 新建 | 域模型 |
| 2 | `marketing/repository/dao/marketing.go` | 新建 | 6 GORM 模型 + 3 DAO |
| 3 | `marketing/repository/dao/init.go` | 新建 | AutoMigrate |
| 4 | `marketing/repository/cache/marketing.go` | 新建 | Redis（优惠券库存+秒杀 Lua） |
| 5 | `marketing/repository/marketing.go` | 新建 | Repository |
| 6 | `marketing/service/marketing.go` | 新建 | MarketingService |
| 7 | `marketing/grpc/marketing.go` | 新建 | 16 RPC Handler |
| 8 | `marketing/events/types.go` | 新建 | 事件类型 |
| 9 | `marketing/events/producer.go` | 新建 | Kafka Producer |
| 10 | `marketing/events/consumer.go` | 新建 | Kafka Consumer |
| 11 | `marketing/ioc/db.go` | 新建 | MySQL |
| 12 | `marketing/ioc/redis.go` | 新建 | Redis |
| 13 | `marketing/ioc/logger.go` | 新建 | Logger |
| 14 | `marketing/ioc/grpc.go` | 新建 | etcd + gRPC Server |
| 15 | `marketing/ioc/kafka.go` | 新建 | Kafka |
| 16 | `marketing/wire.go` | 新建 | Wire DI |
| 17 | `marketing/app.go` | 新建 | App |
| 18 | `marketing/main.go` | 新建 | 入口 |
| 19 | `marketing/config/dev.yaml` | 新建 | 配置 |
| 20 | `marketing/wire_gen.go` | 生成 | Wire |
| 21 | `merchant-bff/handler/marketing.go` | 新建 | MarketingHandler + 10 方法 |
| 22 | `merchant-bff/ioc/grpc.go` | 修改 | +InitMarketingClient |
| 23 | `merchant-bff/ioc/gin.go` | 修改 | +marketingHandler + 10 路由 |
| 24 | `merchant-bff/wire.go` | 修改 | +InitMarketingClient + NewMarketingHandler |
| 25 | `merchant-bff/wire_gen.go` | 重新生成 | Wire |
| 26 | `consumer-bff/handler/marketing.go` | 新建 | MarketingHandler + 5 方法 |
| 27 | `consumer-bff/ioc/grpc.go` | 修改 | +InitMarketingClient |
| 28 | `consumer-bff/ioc/gin.go` | 修改 | +marketingHandler + 5 路由 |
| 29 | `consumer-bff/wire.go` | 修改 | +InitMarketingClient + NewMarketingHandler |
| 30 | `consumer-bff/wire_gen.go` | 重新生成 | Wire |

共 30 个文件（20 新建 + 5 修改 + 1 生成 + 4 重新生成）。
