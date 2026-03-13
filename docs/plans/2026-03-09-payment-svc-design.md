# Payment Service 设计

## 概述

实现 payment-svc 支付服务，提供 7 个 gRPC RPC：CreatePayment、GetPayment、HandleNotify、ClosePayment、Refund、GetRefund、ListPayments。采用渠道抽象模式，实现 mock 渠道 + 微信/支付宝桩代码。

## 架构决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 支付渠道 | mock 完整实现 + wechat/alipay 桩代码 | Channel 接口抽象，后续可扩展 |
| 回调处理 | 纯 gRPC | payment-svc 不暴露 HTTP 端口，由 BFF/网关层转发回调 |
| 幂等方案 | 布隆过滤器 | 复用 emo/idempotent.BloomIdempotencyService |

## 渠道抽象

### Channel 接口

```go
type Channel interface {
    Pay(ctx context.Context, payment domain.PaymentOrder) (channelTradeNo string, payUrl string, err error)
    QueryPayment(ctx context.Context, paymentNo string) (status int32, channelTradeNo string, err error)
    Refund(ctx context.Context, refund domain.RefundRecord) (channelRefundNo string, err error)
    QueryRefund(ctx context.Context, refundNo string) (status int32, channelRefundNo string, err error)
    VerifyNotify(ctx context.Context, data map[string]string) (paymentNo string, channelTradeNo string, err error)
}
```

### 渠道实现

| 渠道 | 状态 | 行为 |
|------|------|------|
| MockChannel | 完整实现 | Pay 直接返回成功（MOCK_ + snowflake ID），Refund 直接成功 |
| WechatChannel | 桩代码 | 所有方法返回 "not implemented" 错误 |
| AlipayChannel | 桩代码 | 所有方法返回 "not implemented" 错误 |

## 数据模型

### payment_orders 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 | 主键 |
| tenant_id | int64 | 租户 ID |
| payment_no | string | 支付单号（唯一） |
| order_id | int64 | 关联订单 ID |
| order_no | string | 关联订单号 |
| channel | string | 支付渠道：mock/wechat/alipay |
| amount | int64 | 金额（分） |
| status | int32 | 状态 |
| channel_trade_no | string | 第三方交易号 |
| pay_time | int64 | 支付时间（毫秒时间戳） |
| expire_time | int64 | 过期时间（毫秒时间戳） |
| notify_url | string | 回调地址 |
| ctime | int64 | 创建时间 |
| utime | int64 | 更新时间 |

索引：`uk_payment_no`、`idx_order_no`、`idx_tenant_status`

### refund_records 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 | 主键 |
| tenant_id | int64 | 租户 ID |
| payment_no | string | 关联支付单号 |
| refund_no | string | 退款单号（唯一） |
| channel | string | 退款渠道 |
| amount | int64 | 退款金额（分） |
| status | int32 | 状态 |
| channel_refund_no | string | 第三方退款号 |
| ctime | int64 | 创建时间 |
| utime | int64 | 更新时间 |

索引：`uk_refund_no`、`idx_payment_no`

### 状态机

支付单状态：
```
待支付(1) → 支付中(2) → 已支付(3)
待支付(1) → 已关闭(4)
支付中(2) → 已关闭(4)
已支付(3) → 退款中(5) → 已退款(6)
```

退款单状态：
```
退款中(1) → 已退款(2)
退款中(1) → 退款失败(3)
```

## 核心流程

### CreatePayment

1. 生成 payment_no（snowflake）
2. 写入 payment_orders（状态=待支付）
3. 调用 Channel.Pay
4. 返回 payment_no + pay_url

### HandleNotify（幂等）

1. 布隆过滤器检查是否已处理
2. VerifyNotify 验证回调数据
3. 查询支付单，检查状态
4. 更新状态为已支付，写入 channel_trade_no 和 pay_time
5. 发送 `order_paid` Kafka 事件（order_no, payment_no, paid_at）
6. 标记布隆过滤器

### ClosePayment

1. 查询支付单
2. 检查状态为待支付或支付中
3. 更新状态为已关闭

### Refund

1. 查询支付单，校验状态为已支付
2. 生成 refund_no（R + snowflake）
3. 创建退款记录（状态=退款中）
4. 调用 Channel.Refund
5. 更新退款记录（channel_refund_no）
6. 更新支付单状态为退款中

## Kafka 事件

### 生产

| 事件 | Topic | 触发时机 | 消费方 |
|------|-------|---------|--------|
| OrderPaidEvent | order_paid | HandleNotify 支付成功 | order-svc |

### 消费

payment-svc 不消费任何外部事件。

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `payment/domain/payment.go` | 新建 | PaymentOrder, RefundRecord, 状态常量 |
| 2 | `payment/repository/dao/payment.go` | 新建 | GORM 模型 + PaymentDAO 接口 |
| 3 | `payment/repository/dao/init.go` | 新建 | AutoMigrate |
| 4 | `payment/repository/cache/payment.go` | 新建 | Redis 缓存（PaymentCache） |
| 5 | `payment/repository/payment.go` | 新建 | Repository（Cache-Aside） |
| 6 | `payment/service/channel/types.go` | 新建 | Channel 接口定义 |
| 7 | `payment/service/channel/mock.go` | 新建 | MockChannel 实现 |
| 8 | `payment/service/channel/wechat.go` | 新建 | WechatChannel 桩代码 |
| 9 | `payment/service/channel/alipay.go` | 新建 | AlipayChannel 桩代码 |
| 10 | `payment/service/payment.go` | 新建 | PaymentService 业务逻辑 |
| 11 | `payment/events/types.go` | 新建 | OrderPaidEvent 定义 |
| 12 | `payment/events/producer.go` | 新建 | Kafka SaramaProducer |
| 13 | `payment/grpc/payment.go` | 新建 | 7 个 RPC Handler |
| 14 | `payment/ioc/db.go` | 新建 | MySQL 初始化 |
| 15 | `payment/ioc/redis.go` | 新建 | Redis 初始化 |
| 16 | `payment/ioc/kafka.go` | 新建 | Kafka 初始化 + Producer |
| 17 | `payment/ioc/logger.go` | 新建 | Logger 初始化 |
| 18 | `payment/ioc/grpc.go` | 新建 | gRPC Server 初始化 |
| 19 | `payment/ioc/idempotent.go` | 新建 | 布隆过滤器初始化 |
| 20 | `payment/ioc/snowflake.go` | 新建 | Snowflake 节点初始化 |
| 21 | `payment/wire.go` | 新建 | Wire DI |
| 22 | `payment/app.go` | 新建 | App 结构体 |
| 23 | `payment/main.go` | 新建 | 服务入口 |
| 24 | `payment/config/dev.yaml` | 新建 | 开发配置（端口 8086, Redis DB 5, mall_payment） |

共 24 个新建文件。
