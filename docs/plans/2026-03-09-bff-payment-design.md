# BFF Payment 接口设计

## 概述

在 consumer-bff 和 merchant-bff 中添加 payment 相关 HTTP 接口，连接 payment-svc gRPC 服务。遵循现有 BFF 模式（handler → gRPC client → Wire DI）。

## 接口清单

### Consumer BFF（3 个接口）

| HTTP 方法 | 路由 | Handler | gRPC 调用 | 说明 |
|-----------|------|---------|-----------|------|
| POST | `/api/v1/payments` | CreatePayment | CreatePayment | 创建支付单 |
| GET | `/api/v1/payments/:paymentNo` | GetPayment | GetPayment | 查询支付状态 |
| POST | `/api/v1/payments/notify` | HandleNotify | HandleNotify | mock 回调模拟 |

- 均在 auth 路由组（需登录）
- CreatePayment 请求体：`{orderNo, channel, amount, orderId}`
- HandleNotify 请求体：`{channel, notifyBody}`

### Merchant BFF（4 个接口）

| HTTP 方法 | 路由 | Handler | gRPC 调用 | 说明 |
|-----------|------|---------|-----------|------|
| GET | `/api/v1/payments` | ListPayments | ListPayments | 店铺支付单列表 |
| GET | `/api/v1/payments/:paymentNo` | GetPayment | GetPayment | 支付详情 |
| POST | `/api/v1/payments/:paymentNo/refund` | Refund | Refund | 发起退款 |
| GET | `/api/v1/refunds/:refundNo` | GetRefund | GetRefund | 退款状态 |

- 均在 auth 路由组（需登录 + tenant 提取）
- ListPayments 使用 tenant_id 过滤
- Refund 请求体：`{amount, reason}`

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `consumer-bff/handler/payment.go` | 新建 | PaymentHandler + 3 方法 |
| 2 | `consumer-bff/ioc/grpc.go` | 修改 | +InitPaymentClient |
| 3 | `consumer-bff/ioc/gin.go` | 修改 | +paymentHandler 参数 + 3 路由 |
| 4 | `consumer-bff/wire.go` | 修改 | +InitPaymentClient + NewPaymentHandler |
| 5 | `merchant-bff/handler/payment.go` | 新建 | PaymentHandler + 4 方法 |
| 6 | `merchant-bff/ioc/grpc.go` | 修改 | +InitPaymentClient |
| 7 | `merchant-bff/ioc/gin.go` | 修改 | +paymentHandler 参数 + 4 路由 |
| 8 | `merchant-bff/wire.go` | 修改 | +InitPaymentClient + NewPaymentHandler |

共 8 个文件（2 新建 + 6 修改） + 2 个 wire_gen.go 重新生成。
