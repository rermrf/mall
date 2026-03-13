# Logistics Service + BFF 设计

## 概述

实现 logistics-svc 物流微服务（10 个 gRPC RPC）+ Kafka Producer（发货通知）+ merchant-bff 物流管理接口（7 个端点）+ consumer-bff 物流查询接口（1 个端点）。涵盖运费模板、运费计算、物流追踪三大功能域。

## 架构决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 数据库 | MySQL (`mall_logistics`) | 运费模板、运费规则、物流单、物流轨迹持久化 |
| 缓存 | Redis | 运费模板+规则缓存（CalculateFreight 热路径） |
| Kafka 角色 | Producer only | 生产 `order_shipped` 事件 |
| BFF 分布 | merchant-bff + consumer-bff | 商家管理运费模板+发货，消费者查物流 |
| gRPC 端口 | 8090 | |
| Redis DB | 9 | |
| 服务名 | logistics | |

## 数据模型

### MySQL 表（4 张）

| 表 | 关键字段 | 索引 |
|----|---------|------|
| `freight_templates` | id, tenant_id, name, charge_type(1按件/2按重量), free_threshold(分,0=不包邮), ctime, utime | `idx_tenant` |
| `freight_rules` | id, template_id, regions(JSON省编码列表), first_unit, first_price(分), additional_unit, additional_price(分), ctime, utime | `idx_template` |
| `shipments` | id, tenant_id, order_id, carrier_code, carrier_name, tracking_no, status(1已发货/2运输中/3已签收), ctime, utime | `uk_order(order_id)`, `idx_tenant_status` |
| `shipment_tracks` | id, shipment_id, description, location, track_time, ctime | `idx_shipment` |

### Redis 缓存策略

- 运费模板：`logistics:templates:{tenantId}` — String(JSON)，缓存租户所有运费模板+规则，TTL 30min
- CalculateFreight 先查 Redis，miss 则查 MySQL 并回填
- 模板 CUD 操作清除对应租户缓存

## logistics-svc RPC（10 个）

| RPC | 说明 | 实现方式 |
|-----|------|---------|
| CreateFreightTemplate | 创建运费模板+规则 | MySQL INSERT(template+rules) + 清除缓存 |
| UpdateFreightTemplate | 更新运费模板+规则 | MySQL UPDATE + 删旧规则+INSERT新规则 + 清除缓存 |
| GetFreightTemplate | 获取模板详情 | MySQL 查询（含 rules） |
| ListFreightTemplates | 模板列表 | MySQL 查询（按 tenant_id） |
| DeleteFreightTemplate | 删除模板 | MySQL DELETE(template+rules) + 清除缓存 |
| CalculateFreight | 运费计算 | Redis/MySQL 获取模板 → 匹配省份规则 → 按计费方式计算 |
| CreateShipment | 创建物流单（发货） | MySQL INSERT + Kafka `order_shipped` |
| GetShipment | 物流单详情 | MySQL 查询（含 tracks） |
| GetShipmentByOrder | 按订单查物流 | MySQL 查询（含 tracks） |
| AddTrack | 添加物流轨迹 | MySQL INSERT + 可更新 shipment status |

## Kafka 事件

| Topic | 方向 | 事件 | 说明 |
|-------|------|------|------|
| `order_shipped` | 生产 | OrderShippedEvent{OrderId, TenantId, CarrierCode, CarrierName, TrackingNo} | 发货通知 → order-svc/notification-svc 消费 |

## Merchant BFF 接口（7 个）

| HTTP 方法 | 路由 | 说明 | 认证 |
|-----------|------|------|------|
| POST | `/api/v1/freight-templates` | 创建运费模板 | 需登录 |
| PUT | `/api/v1/freight-templates/:id` | 更新运费模板 | 需登录 |
| GET | `/api/v1/freight-templates/:id` | 获取模板详情 | 需登录 |
| GET | `/api/v1/freight-templates` | 模板列表 | 需登录 |
| DELETE | `/api/v1/freight-templates/:id` | 删除模板 | 需登录 |
| POST | `/api/v1/orders/:order_no/ship` | **聚合**: CreateShipment + UpdateOrderStatus | 需登录 |
| GET | `/api/v1/orders/:order_no/logistics` | 查物流 | 需登录 |

> merchant-bff 所有路由都在 auth 路由组中。`POST /orders/:order_no/ship` 是聚合端点：先通过 order-svc 获取订单信息（order_id），再调用 logistics-svc.CreateShipment，最后调用 order-svc.UpdateOrderStatus 标记已发货。

## Consumer BFF 接口（1 个）

| HTTP 方法 | 路由 | 说明 | 认证 |
|-----------|------|------|------|
| GET | `/api/v1/orders/:order_no/logistics` | 查询订单物流 | 需登录 |

> consumer-bff auth 路由组。通过 order-svc 查 order_id，再调 logistics-svc.GetShipmentByOrder。

## 基础设施

- gRPC 端口：8090
- 服务名：logistics
- MySQL 库：mall_logistics
- Redis DB：9

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `logistics/domain/logistics.go` | 新建 | 域模型 |
| 2 | `logistics/repository/dao/logistics.go` | 新建 | 4 GORM 模型 + 3 DAO |
| 3 | `logistics/repository/dao/init.go` | 新建 | AutoMigrate |
| 4 | `logistics/repository/cache/logistics.go` | 新建 | Redis（运费模板缓存） |
| 5 | `logistics/repository/logistics.go` | 新建 | Repository |
| 6 | `logistics/service/logistics.go` | 新建 | LogisticsService |
| 7 | `logistics/grpc/logistics.go` | 新建 | 10 RPC Handler |
| 8 | `logistics/events/types.go` | 新建 | 事件类型 |
| 9 | `logistics/events/producer.go` | 新建 | Kafka Producer |
| 10 | `logistics/ioc/db.go` | 新建 | MySQL |
| 11 | `logistics/ioc/redis.go` | 新建 | Redis |
| 12 | `logistics/ioc/logger.go` | 新建 | Logger |
| 13 | `logistics/ioc/grpc.go` | 新建 | etcd + gRPC Server |
| 14 | `logistics/ioc/kafka.go` | 新建 | Kafka |
| 15 | `logistics/wire.go` | 新建 | Wire DI |
| 16 | `logistics/app.go` | 新建 | App |
| 17 | `logistics/main.go` | 新建 | 入口 |
| 18 | `logistics/config/dev.yaml` | 新建 | 配置 |
| 19 | `logistics/wire_gen.go` | 生成 | Wire |
| 20 | `merchant-bff/handler/logistics.go` | 新建 | LogisticsHandler + 7 方法 |
| 21 | `merchant-bff/ioc/grpc.go` | 修改 | +InitLogisticsClient |
| 22 | `merchant-bff/ioc/gin.go` | 修改 | +logisticsHandler + 7 路由 |
| 23 | `merchant-bff/wire.go` | 修改 | +InitLogisticsClient + NewLogisticsHandler |
| 24 | `merchant-bff/wire_gen.go` | 重新生成 | Wire |
| 25 | `consumer-bff/handler/logistics.go` | 新建 | LogisticsHandler + 1 方法 |
| 26 | `consumer-bff/ioc/grpc.go` | 修改 | +InitLogisticsClient |
| 27 | `consumer-bff/ioc/gin.go` | 修改 | +logisticsHandler + 1 路由 |
| 28 | `consumer-bff/wire.go` | 修改 | +InitLogisticsClient + NewLogisticsHandler |
| 29 | `consumer-bff/wire_gen.go` | 重新生成 | Wire |

共 29 个文件（19 新建 + 6 修改 + 1 生成 + 3 重新生成）。
