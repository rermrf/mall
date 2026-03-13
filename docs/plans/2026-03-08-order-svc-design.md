# Order Service (order-svc) 设计文档

## 概述

SaaS 多租户商城的订单微服务，支持完整的订单生命周期管理（创建、支付、发货、收货、完成、取消）和部分退款。核心亮点：布隆过滤器 + 唯一索引防重、go-delay 超时关单、Kafka 事件驱动支付回调、三阶段库存协调。

## 状态机

```
                  ┌─ 超时30min ──> cancelled（库存回滚）
                  │
pending ──支付成功──> paid ──商家发货──> shipped ──确认收货──> received ──7天自动完成──> completed
                    │                                         │
                    └──申请退款──> refunding ──退款成功──> refunded
                                    │
                                    └──商家拒绝──> paid（退回原状态）
```

状态值：1=pending 2=paid 3=shipped 4=received 5=completed 6=cancelled 7=refunding 8=refunded

## 创建订单核心流程

1. 计算 request_key（`order:create:{buyer_id}:{items_hash}`）
2. `BloomIdempotencyService.Exists()` 检查，已存在则查 MySQL 确认（解决假阳性）
3. 调用 product-svc BatchGetProducts → 校验价格、获取商品快照
4. Snowflake 生成 order_no
5. 调用 inventory-svc Deduct → 预扣库存
6. MySQL 写入订单 + 订单项（唯一索引兜底）
7. 发 go-delay 延迟消息 → 30min 后投递到 `order_close_delay`
8. 调用 payment-svc CreatePayment → 生成支付单
9. 返回 order_no + pay_url

### 防重方案

- **快速拦截层**：Redis 布隆过滤器（emo/idempotent.BloomIdempotencyService）
- **假阳性处理**：布隆过滤器返回"已存在"时，回查 MySQL UNIQUE(buyer_id, buyer_hash) 确认
- **最终兜底**：MySQL 唯一索引，即使布隆过滤器误放行也能拦住

### 失败补偿

```
bloom.Exists → product.BatchGet → inventory.Deduct → MySQL insert → go-delay → payment.Create
                                       ↓失败              ↓失败           ↓失败
                                    直接返回错误     inventory.Rollback  inventory.Rollback + cancel
```

- go-delay 发送失败 → 只记日志（30min 后库存自动回滚兜底）

## 支付成功流程（消费 order_paid）

1. 收到 Kafka `order_paid` 事件
2. 更新订单状态 pending → paid
3. 调用 inventory-svc Confirm → 确认扣减
4. 写状态变更日志

## 超时关单流程（消费 order_close_delay）

1. 收到 go-delay 投递的超时消息
2. 查订单状态，若仍为 pending → 更新为 cancelled
3. 调用 inventory-svc Rollback → 回滚库存
4. 调用 payment-svc ClosePayment → 关闭支付单

## 数据模型

### orders 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 自增主键 |
| order_no | varchar(64) UNIQUE | Snowflake 订单号 |
| tenant_id | bigint | 租户 ID |
| buyer_id | bigint | 买家用户 ID |
| buyer_hash | varchar(64) | buyer_id + items 哈希，UNIQUE(buyer_id, buyer_hash) |
| status | tinyint | 订单状态 |
| total_amount | bigint | 订单总金额（分） |
| pay_amount | bigint | 实付金额（分） |
| refunded_amount | bigint | 已退款金额（分） |
| payment_no | varchar(64) | 支付单号 |
| receiver_name | varchar(64) | 收货人 |
| receiver_phone | varchar(32) | 收货电话 |
| receiver_address | varchar(512) | 收货地址 |
| remark | varchar(256) | 买家备注 |
| paid_at | bigint | 支付时间 |
| shipped_at | bigint | 发货时间 |
| received_at | bigint | 确认收货时间 |
| ctime | bigint | 创建时间 |
| utime | bigint | 更新时间 |

### order_items 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 自增主键 |
| order_id | bigint | 关联 orders.id |
| tenant_id | bigint | 租户 ID |
| sku_id | bigint | SKU ID |
| product_id | bigint | 商品 ID |
| product_name | varchar(256) | 商品名称快照 |
| sku_spec | varchar(512) | 规格快照 |
| image | varchar(512) | 商品图片快照 |
| price | bigint | 单价（分）快照 |
| quantity | int | 数量 |
| subtotal | bigint | 小计 |
| refunded_quantity | int | 已退款数量 |
| ctime | bigint | 创建时间 |

### order_status_logs 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 自增主键 |
| order_id | bigint | 关联 orders.id |
| old_status | tinyint | 变更前状态 |
| new_status | tinyint | 变更后状态 |
| operator | varchar(64) | 操作人 |
| remark | varchar(256) | 变更原因 |
| ctime | bigint | 变更时间 |

### refund_orders 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 自增主键 |
| refund_no | varchar(64) UNIQUE | Snowflake 退款单号 |
| order_id | bigint | 关联 orders.id |
| tenant_id | bigint | 租户 ID |
| buyer_id | bigint | 买家 ID |
| status | tinyint | 1=申请中 2=已同意 3=已拒绝 4=退款中 5=已退款 6=退款失败 |
| refund_amount | bigint | 退款金额（分） |
| reason | varchar(512) | 退款原因 |
| reject_reason | varchar(512) | 拒绝原因 |
| items | text | 退款商品明细 JSON |
| ctime | bigint | 创建时间 |
| utime | bigint | 更新时间 |

### Redis 缓存

| Key | 类型 | TTL | 说明 |
|-----|------|-----|------|
| `order:info:{order_no}` | String (JSON) | 15min | 订单详情缓存 |
| `order:bloom` | Bloom Filter | - | 创建订单防重 |

## Kafka 事件

### 生产

| Topic | 触发时机 | 消费者 |
|-------|----------|--------|
| `delay_topic` | 创建订单后 | go-delay → 30min 后投递到 order_close_delay |
| `order_cancelled` | 超时关单/手动取消 | 未来扩展 |
| `order_completed` | 订单完成 | product-svc（IncrSales） |

### 消费

| Topic | 生产者 | 处理逻辑 |
|-------|--------|----------|
| `order_paid` | payment-svc | pending → paid + inventory.Confirm |
| `order_close_delay` | go-delay | 若 pending → cancelled + inventory.Rollback + payment.ClosePayment |

## 跨服务 gRPC 调用

| 被调方 | RPC | 时机 |
|--------|-----|------|
| product-svc | BatchGetProducts | 创建订单 |
| inventory-svc | Deduct | 创建订单 |
| inventory-svc | Confirm | 支付成功 |
| inventory-svc | Rollback | 超时关单/取消 |
| payment-svc | CreatePayment | 创建订单 |
| payment-svc | ClosePayment | 超时关单 |
| payment-svc | Refund | 商家同意退款 |
| product-svc | IncrSales | 订单完成 |

## 服务配置

| 配置项 | 值 |
|--------|-----|
| gRPC 端口 | 8085 |
| 数据库 | mall_order |
| Redis DB | 4 |
| Kafka ConsumerGroup | order-svc |
| etcd 服务名 | order |

## 文件清单（21 个文件）

| # | 文件路径 | 说明 |
|---|---------|------|
| 1 | order/domain/order.go | 领域实体 + 状态枚举 |
| 2 | order/repository/dao/order.go | 4 GORM 模型 + OrderDAO |
| 3 | order/repository/dao/init.go | AutoMigrate 4 张表 |
| 4 | order/repository/cache/order.go | 订单详情缓存 |
| 5 | order/repository/order.go | Repository 协调 DAO + Cache |
| 6 | order/events/types.go | 事件 DTO |
| 7 | order/events/producer.go | Producer |
| 8 | order/events/consumer.go | 2 个 Consumer |
| 9 | order/service/order.go | 10 个服务方法 + 补偿逻辑 |
| 10 | order/grpc/order.go | 10 RPC handler |
| 11 | order/ioc/db.go | MySQL |
| 12 | order/ioc/redis.go | Redis |
| 13 | order/ioc/kafka.go | Kafka |
| 14 | order/ioc/logger.go | Logger |
| 15 | order/ioc/grpc.go | gRPC server + 3 clients |
| 16 | order/ioc/idempotent.go | BloomIdempotencyService |
| 17 | order/ioc/snowflake.go | Snowflake Node |
| 18 | order/config/dev.yaml | 配置 |
| 19 | order/app.go | App 聚合 |
| 20 | order/wire.go | Wire DI |
| 21 | order/main.go | 服务入口 |
