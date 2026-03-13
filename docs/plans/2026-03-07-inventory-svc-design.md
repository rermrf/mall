# Inventory Service 设计文档

## 日期

2026-03-07

## 概述

SaaS 多租户商城的库存微服务（inventory-svc），负责库存设置、查询和三阶段扣减（预扣→确认→回滚）。核心亮点：**Redis Lua 原子预扣** + **go-delay 延迟消息超时回滚**。Proto 已定义 7 个 RPC。

## 架构决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 分层架构 | DDD 分层（沿用项目统一风格） | 项目统一风格 |
| 三阶段模式 | TCC: Deduct/Confirm/Rollback | Proto 已定义，业界成熟模式 |
| 预扣实现 | Redis Lua 原子操作 | 高并发下原子性保证 |
| 预扣记录 | Redis Hash（不入 MySQL） | Confirm/Rollback 时才落库 |
| 超时回滚 | go-delay 延迟消息服务 | 自研亮点，MySQL+Kafka 实现 |
| SetStock 策略 | MySQL + Redis 同步写 | 保证两者一致 |
| 端口 | 8084 | user(8081), tenant(8082), product(8083) 之后 |
| 数据库 | mall_inventory | 独立库 |
| Redis DB | 3 | tenant(1), product(2) 之后 |

## 架构流程

```
下单 → Deduct (Redis Lua 原子预扣)
            ↓
      go-delay 发 30min 延迟消息
            ↓
    ┌───────┴───────┐
    │               │
  支付成功        30min 超时
    │               │
  Confirm         Rollback
  (MySQL 落库)    (Redis Lua 回滚)
```

## 数据模型

### Domain 层

**inventory.go — Inventory + DeductRecord + InventoryLog：**

```
Inventory:
  ID, TenantID, SKUID int64
  Total, Available, Locked, Sold int32
  AlertThreshold int32
  Ctime, Utime time.Time

DeductRecord:
  OrderID int64
  Items []DeductItem
  TenantID int64

DeductItem:
  SKUID int64
  Quantity int32

InventoryLog:
  ID, SKUID, OrderID int64
  Type: Deduct(1) / Confirm(2) / Rollback(3) / Manual(4)
  Quantity int32
  BeforeAvailable, AfterAvailable int32
  TenantID int64
  Ctime time.Time
```

### DAO 层（2 张表）

| 表名 | 关键索引 |
|------|---------|
| inventories | `uniqueIndex:uk_tenant_sku(tenant_id, sku_id)` |
| inventory_logs | `idx_sku(sku_id)`, `idx_order(order_id)` |

预扣记录不入 MySQL，只存 Redis Hash，Confirm/Rollback 时才写 inventory_logs。

### Cache/Redis 层

| Key | 类型 | 说明 |
|-----|------|------|
| `inventory:stock:{sku_id}` | Hash | `{total, available, locked, sold}` — Lua 原子操作 |
| `inventory:deduct:{order_id}` | Hash | `{sku_id: quantity, ...}` — 预扣记录，TTL 35min |

Lua 脚本两个：
- **deduct.lua**: 多 SKU 原子预扣，available -= qty, locked += qty，任一不足全部失败
- **rollback.lua**: 多 SKU 原子回滚，available += qty, locked -= qty

## Service 层

单文件单接口（`InventoryService`），包含：

- `SetStock(ctx, tenantId, skuId, total, alertThreshold) → error` — MySQL upsert + Redis Hash 同步写
- `GetStock(ctx, skuId) → (Inventory, error)` — 优先读 Redis，miss 则读 MySQL 回填
- `BatchGetStock(ctx, skuIds) → ([]Inventory, error)` — 批量查询
- `Deduct(ctx, orderId, tenantId, items) → (bool, string, error)` — Redis Lua 原子预扣 → 存 Redis Hash → 发 go-delay 延迟消息
- `Confirm(ctx, orderId) → error` — 读 Redis Hash → MySQL 事务更新 → 写 log → 删 Redis Hash
- `Rollback(ctx, orderId) → error` — 读 Redis Hash → Redis Lua 回滚 → MySQL 写 log → 删 Redis Hash
- `ListLogs(ctx, tenantId, skuId, page, pageSize) → ([]InventoryLog, int64, error)`

### 关键业务逻辑

**Deduct 流程：**
1. Redis Lua 原子执行：检查所有 SKU available >= quantity，全部满足才扣减
2. 任一 SKU 库存不足 → 全部不扣，返回 success=false, message="SKU xxx 库存不足"
3. 成功后写 Redis Hash `inventory:deduct:{order_id}` (TTL 35min)
4. 发延迟消息到 delay_topic：`{biz: "inventory", key: order_id, biz_topic: "inventory_deduct_expire", execute_at: now+30min}`

**Confirm 幂等：** 读 Redis Hash，不存在 → 已处理，直接返回 nil

**Rollback 幂等：** 读 Redis Hash，不存在 → 已处理，直接返回 nil

## Events

| 事件 | Topic | 触发时机 | 消费者 |
|------|-------|---------|--------|
| 延迟回滚 | inventory_deduct_expire | Deduct 后 30min（go-delay 投递） | inventory-svc 自身 |
| 库存预警 | inventory_alert | SetStock/Confirm 后 available < threshold | notification-svc |

**Consumer**：inventory-svc 消费 `inventory_deduct_expire`，检查 Redis Hash 是否存在，存在则调 Rollback。

## 跨服务依赖

- inventory-svc → go-delay（Kafka: delay_topic → inventory_deduct_expire）
- order-svc → inventory-svc（gRPC: Deduct / Confirm / Rollback）

## gRPC 层

`InventoryGRPCServer` 实现 proto 中定义的 7 个 RPC。

关键点：
- tenant_id 从 gRPC metadata 获取（tenantx 拦截器）或从 request 参数获取
- Deduct 返回 success + message，不使用 gRPC error 表示库存不足（业务失败 ≠ 系统错误）
- Confirm/Rollback 幂等，重复调用不报错

## 文件清单

| # | 文件 | 说明 |
|---|------|------|
| 1 | `domain/inventory.go` | Inventory + DeductRecord + DeductItem + InventoryLog + 枚举 |
| 2 | `repository/dao/inventory.go` | Inventory + InventoryLog DAO |
| 3 | `repository/dao/init.go` | AutoMigrate 2 张表 |
| 4 | `repository/cache/inventory.go` | Redis Hash 读写 + Lua 脚本（deduct/rollback） |
| 5 | `repository/inventory.go` | InventoryRepository（MySQL + Redis 协调） |
| 6 | `service/inventory.go` | 业务逻辑（三阶段核心） |
| 7 | `events/types.go` | 延迟消息 DTO + 库存预警事件 |
| 8 | `events/producer.go` | Kafka Producer（delay_topic + inventory_alert） |
| 9 | `events/consumer.go` | 消费 inventory_deduct_expire → 触发 Rollback |
| 10 | `grpc/inventory.go` | 7 RPC handler |
| 11 | `ioc/db.go` | MySQL 初始化 |
| 12 | `ioc/redis.go` | Redis 初始化 |
| 13 | `ioc/kafka.go` | Kafka 初始化 + Producer + Consumer |
| 14 | `ioc/logger.go` | Logger 初始化 |
| 15 | `ioc/grpc.go` | gRPC server 初始化 |
| 16 | `config/dev.yaml` | 配置（port 8084, db mall_inventory, redis db 3） |
| 17 | `app.go` | App 聚合（Server + Consumer） |
| 18 | `wire.go` | Wire DI |
| 19 | `main.go` | 入口 |

共 19 个文件。
