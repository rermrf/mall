# Tenant Service (tenant-svc) 设计文档

## 日期

2026-03-07

## 概述

SaaS 多租户商城项目的租户/商家服务。负责商家入驻审核、店铺管理、SaaS 套餐管理和配额控制四大功能域，共 17 个 gRPC RPC。

## 架构决策

- 完全对齐 user 服务的 DDD 分层：`domain → dao → cache → repository → service → grpc → events → ioc → wire → main`
- 独立数据库 `mall_tenant`，gRPC 端口 8082，etcd 注册名 `tenant`
- Proto 已定义并生成（`api/proto/gen/tenant/v1/`）
- Logger 使用 `github.com/rermrf/emo/logger`
- Kafka 只做 Producer（2 个 topic），不实现 consumer

## 数据模型

### 数据库表

| 表 | DAO | 关键索引 |
|---|-----|---------|
| `tenants` | `TenantDAO` | PK `id` |
| `tenant_plans` | `PlanDAO` | PK `id` |
| `tenant_quota_usage` | `QuotaDAO` | `uniqueIndex:uk_tenant_type(tenant_id, quota_type)` |
| `shops` | `ShopDAO` | `uniqueIndex:uk_tenant(tenant_id)`, `uniqueIndex:uk_subdomain(subdomain)`, `uniqueIndex:uk_custom_domain(custom_domain)` |

shops 表以 proto 为准，包含 `subdomain` 和 `custom_domain` 字段（支持 GetShopByDomain 域名解析）。

### 缓存策略 (Cache-Aside)

| Key | TTL | 说明 |
|-----|-----|------|
| `tenant:info:{id}` | 30min | 租户信息 |
| `tenant:quota:{tid}:{type}` | 10min | 配额使用量 |
| `shop:info:{tenant_id}` | 15min | 店铺信息 |
| `shop:domain:{domain}` | 15min | 域名→店铺映射（高频查询） |

## RPC 接口（17 个）

### 租户管理（6）

- `CreateTenant` — 商家入驻，状态设为"待审核"
- `GetTenant` — 查询商家信息（走缓存）
- `UpdateTenant` — 更新商家信息（清缓存）
- `ListTenants` — 商家列表（分页+状态过滤，不走缓存）
- `ApproveTenant` — 审核商家（通过/拒绝），通过时发送 `tenant_approved` Kafka 事件
- `FreezeTenant` — 冻结/解冻商家

### 套餐管理（4）

- `GetPlan` — 查询套餐详情
- `ListPlans` — 套餐列表
- `CreatePlan` — 创建套餐
- `UpdatePlan` — 更新套餐

### 配额控制（3）

- `CheckQuota` — 检查配额是否允许（如商品数量是否超限），返回 allowed + 当前用量
- `IncrQuota` — 增加配额使用量（创建商品时调用）
- `DecrQuota` — 减少配额使用量（删除商品时调用）

### 店铺管理（3+1）

- `GetShop` — 按 tenant_id 查店铺
- `UpdateShop` — 更新店铺信息
- `GetShopByDomain` — 域名解析，consumer-bff TenantResolve 中间件调用

## Kafka 事件

只做 Producer：

| Topic | 触发时机 | 消息体 | 下游消费者 |
|-------|---------|--------|-----------|
| `tenant_approved` | ApproveTenant 审核通过 | `{TenantId, Name, PlanId}` | user-svc, notification-svc |
| `tenant_plan_changed` | 套餐变更 | `{TenantId, OldPlanId, NewPlanId}` | product-svc |

## Domain 层

3 个文件：

- `tenant.go` — Tenant 实体 + TenantStatus 枚举（1-待审核 2-正常 3-冻结 4-注销）
- `plan.go` — TenantPlan 实体 + PlanStatus 枚举
- `shop.go` — Shop 实体 + ShopStatus 枚举 + QuotaUsage 值对象

## 文件清单

| # | 文件路径 | 说明 |
|---|---------|------|
| 1 | `tenant/domain/tenant.go` | Tenant + TenantStatus |
| 2 | `tenant/domain/plan.go` | TenantPlan + PlanStatus |
| 3 | `tenant/domain/shop.go` | Shop + ShopStatus + QuotaUsage |
| 4 | `tenant/repository/dao/tenant.go` | 4 GORM 模型 + 4 DAO 接口及实现 |
| 5 | `tenant/repository/dao/init.go` | AutoMigrate 4 张表 |
| 6 | `tenant/repository/cache/tenant.go` | Redis 缓存 |
| 7 | `tenant/repository/tenant.go` | CachedRepository |
| 8 | `tenant/service/tenant.go` | 业务逻辑 |
| 9 | `tenant/events/types.go` | 事件定义 |
| 10 | `tenant/events/producer.go` | Kafka Producer |
| 11 | `tenant/grpc/tenant.go` | 17 RPC Handler |
| 12 | `tenant/ioc/db.go` | MySQL 初始化 |
| 13 | `tenant/ioc/redis.go` | Redis 初始化 |
| 14 | `tenant/ioc/kafka.go` | Kafka 初始化 |
| 15 | `tenant/ioc/logger.go` | Logger 初始化 |
| 16 | `tenant/ioc/grpc.go` | gRPC Server 初始化 |
| 17 | `tenant/app.go` | App 聚合 |
| 18 | `tenant/wire.go` | Wire DI |
| 19 | `tenant/config/dev.yaml` | 开发配置 |
| 20 | `tenant/main.go` | 服务入口 |
