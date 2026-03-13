# Notification Service + BFF 设计

## 概述

实现 notification-svc 通知微服务（10 个 gRPC RPC）+ Kafka Consumer（5 个事件）+ 3 种通知渠道（Aliyun SMS、SMTP Email、站内信）+ consumer-bff 通知接口（4 个端点）+ merchant-bff 通知接口（4 个端点）。涵盖通知发送、通知查询、模板管理三大功能域。

## 架构决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 数据库 | MySQL (`mall_notification`) | 通知模板、通知记录持久化 |
| 缓存 | Redis | 未读数量缓存（热路径） |
| Kafka 角色 | Consumer only | 消费 5 个事件 topic |
| SMS 渠道 | Aliyun SMS SDK | `github.com/alibabacloud-go/dysmsapi-20170525/v4` |
| Email 渠道 | Go `net/smtp` | 标准 SMTP 发送 |
| 模板引擎 | Go `text/template` | 支持 `{{.OrderNo}}` 等占位符 |
| BFF 分布 | consumer-bff + merchant-bff | 消费者/商家查看通知 |
| gRPC 端口 | 8091 | |
| Redis DB | 10 | |
| 服务名 | notification | |

## 数据模型

### MySQL 表（2 张）

| 表 | 关键字段 | 索引 |
|----|---------|------|
| `notification_templates` | id, tenant_id(0=平台模板), code, channel(1=SMS/2=Email/3=站内信), title, content(支持模板占位符), status(1=启用/2=停用), ctime, utime | `uk_tenant_code_channel(tenant_id,code,channel)` |
| `notifications` | id, user_id, tenant_id, channel(1=SMS/2=Email/3=站内信), title, content, is_read, status(1=待发送/2=已发送/3=发送失败), ctime, utime | `idx_user_read(user_id,is_read)`, `idx_tenant` |

### Redis 缓存策略

- 未读数量：`notification:unread:{userId}` — String(int64)，TTL 10min
- 新通知入库后清除缓存（下次查询重新计算）
- MarkRead / MarkAllRead 清除缓存

## Kafka Consumers（5 个 topic）

| Topic | 来源 | 通知动作 | 模板编码 |
|-------|------|---------|---------|
| `user_registered` | user-svc | 欢迎 SMS + Email 发给用户 | `welcome_sms`, `welcome_email` |
| `order_paid` | payment-svc | 站内信通知商家有新订单 | `order_paid_merchant` |
| `order_shipped` | logistics-svc | SMS + 站内信通知买家已发货 | `ship_notify_sms`, `ship_notify_inapp` |
| `inventory_alert` | inventory-svc | 站内信 + Email 通知商家库存不足 | `inventory_alert_inapp`, `inventory_alert_email` |
| `tenant_approved` | tenant-svc | 站内信 + Email 通知商家审核通过 | `tenant_approved_inapp`, `tenant_approved_email` |

每个 Consumer：解析事件 → 按 template_code+channel 查模板 → text/template 渲染 → 渠道分发 → 持久化通知记录。

## notification-svc RPC（10 个）

| RPC | 说明 | 实现方式 |
|-----|------|---------|
| SendNotification | 按模板发送单条通知 | 查模板 → 渲染 → 分发 → 入库 |
| BatchSendNotification | 批量发送 | 循环调用 SendNotification |
| ListNotifications | 分页查询用户通知 | MySQL 分页查询 |
| MarkRead | 标记单条已读 | MySQL UPDATE + 清除未读缓存 |
| MarkAllRead | 全部标记已读 | MySQL UPDATE + 清除未读缓存 |
| GetUnreadCount | 获取未读数量 | Redis/MySQL |
| CreateTemplate | 创建通知模板 | MySQL INSERT |
| UpdateTemplate | 更新通知模板 | MySQL UPDATE |
| DeleteTemplate | 删除通知模板 | MySQL DELETE |
| ListTemplates | 模板列表 | MySQL 查询 |

## 通知渠道提供者

| 渠道 | 接口 | 实现 |
|------|------|------|
| SMS | `SmsProvider.Send(ctx, phone, templateCode, params)` | `AliyunSmsProvider` — 阿里云 SDK |
| Email | `EmailProvider.Send(ctx, to, subject, body)` | `SmtpEmailProvider` — net/smtp |
| 站内信 | 直接 DB INSERT | 通过 NotificationDAO |

## Consumer BFF 接口（4 个）

| HTTP 方法 | 路由 | 说明 | 认证 |
|-----------|------|------|------|
| GET | `/api/v1/notifications` | 分页查询通知 | 需登录 |
| GET | `/api/v1/notifications/unread-count` | 未读数量 | 需登录 |
| PUT | `/api/v1/notifications/:id/read` | 标记已读 | 需登录 |
| PUT | `/api/v1/notifications/read-all` | 全部已读 | 需登录 |

## Merchant BFF 接口（4 个）

| HTTP 方法 | 路由 | 说明 | 认证 |
|-----------|------|------|------|
| GET | `/api/v1/notifications` | 分页查询通知 | 需登录 |
| GET | `/api/v1/notifications/unread-count` | 未读数量 | 需登录 |
| PUT | `/api/v1/notifications/:id/read` | 标记已读 | 需登录 |
| PUT | `/api/v1/notifications/read-all` | 全部已读 | 需登录 |

> 两个 BFF 的通知接口逻辑完全一致，都通过 gRPC 调用 notification-svc。区别在于 uid 来源不同（consumer-bff 从 JWT 获取消费者 uid，merchant-bff 从 JWT 获取商家员工 uid）。

## 基础设施

- gRPC 端口：8091
- 服务名：notification
- MySQL 库：mall_notification
- Redis DB：10

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `notification/domain/notification.go` | 新建 | 域模型 |
| 2 | `notification/repository/dao/notification.go` | 新建 | 2 GORM 模型 + 2 DAO |
| 3 | `notification/repository/dao/init.go` | 新建 | AutoMigrate |
| 4 | `notification/repository/cache/notification.go` | 新建 | Redis（未读数量缓存） |
| 5 | `notification/repository/notification.go` | 新建 | Repository |
| 6 | `notification/service/notification.go` | 新建 | NotificationService |
| 7 | `notification/service/provider/sms.go` | 新建 | SmsProvider 接口 + AliyunSmsProvider |
| 8 | `notification/service/provider/email.go` | 新建 | EmailProvider 接口 + SmtpEmailProvider |
| 9 | `notification/grpc/notification.go` | 新建 | 10 RPC Handler |
| 10 | `notification/events/types.go` | 新建 | 事件类型（消费的 5 种事件） |
| 11 | `notification/events/consumer.go` | 新建 | 5 个 Kafka Consumer |
| 12 | `notification/ioc/db.go` | 新建 | MySQL |
| 13 | `notification/ioc/redis.go` | 新建 | Redis |
| 14 | `notification/ioc/logger.go` | 新建 | Logger |
| 15 | `notification/ioc/grpc.go` | 新建 | etcd + gRPC Server |
| 16 | `notification/ioc/kafka.go` | 新建 | Kafka Consumer Group + Consumers |
| 17 | `notification/ioc/provider.go` | 新建 | SMS + Email provider 初始化 |
| 18 | `notification/wire.go` | 新建 | Wire DI |
| 19 | `notification/app.go` | 新建 | App（含 Server + Consumers） |
| 20 | `notification/main.go` | 新建 | 入口 |
| 21 | `notification/config/dev.yaml` | 新建 | 配置 |
| 22 | `notification/wire_gen.go` | 生成 | Wire |
| 23 | `consumer-bff/handler/notification.go` | 新建 | NotificationHandler + 4 方法 |
| 24 | `consumer-bff/ioc/grpc.go` | 修改 | +InitNotificationClient |
| 25 | `consumer-bff/ioc/gin.go` | 修改 | +notificationHandler + 4 路由 |
| 26 | `consumer-bff/wire.go` | 修改 | +providers |
| 27 | `consumer-bff/wire_gen.go` | 重新生成 | Wire |
| 28 | `merchant-bff/handler/notification.go` | 新建 | NotificationHandler + 4 方法 |
| 29 | `merchant-bff/ioc/grpc.go` | 修改 | +InitNotificationClient |
| 30 | `merchant-bff/ioc/gin.go` | 修改 | +notificationHandler + 4 路由 |
| 31 | `merchant-bff/wire.go` | 修改 | +providers |
| 32 | `merchant-bff/wire_gen.go` | 重新生成 | Wire |

共 32 个文件（22 新建 + 6 修改 + 1 生成 + 3 重新生成）。
