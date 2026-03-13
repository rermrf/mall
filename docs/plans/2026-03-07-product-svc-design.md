# Product Service 设计文档

## 日期

2026-03-07

## 概述

SaaS 多租户商城的商品微服务（product-svc），负责 SPU/SKU 管理、三级分类树、品牌 CRUD、动态规格属性和发布/下架状态控制。Proto 已定义 17 个 RPC。

## 架构决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 分层架构 | DDD 分层（沿用 tenant-svc） | 项目统一风格 |
| SKU 策略 | SPU+SKU 整体操作 | 一致性好，事务内创建/更新 |
| 分类树 | 邻接表 + 内存建树 | 数量少（百级），简单高效 |
| 配额检查 | product-svc 直接 gRPC 调 tenant-svc | 原子性好 |
| 缓存 | Cache-Aside | 沿用已有模式 |
| 事件 | Kafka 异步 | 解耦 search/cart 服务 |
| 端口 | 8083 | user(8081), tenant(8082) 之后 |
| 数据库 | mall_product | 独立库 |

## 数据模型

### Domain 层

**product.go — SPU + SKU + Spec：**

```
Product:
  ID, TenantID, CategoryID, BrandID int64
  Name, Subtitle, Images(JSON), Description string
  Status: Draft(1) / Published(2) / Unpublished(3)
  Sales int64
  SKUs []SKU
  Specs []ProductSpec
  Ctime, Utime time.Time

SKU:
  ID, TenantID, ProductID int64
  SpecValues string (JSON, e.g. {"颜色":"红色","尺码":"XL"})
  Price, OriginalPrice, CostPrice int64 (单位: 分)
  SKUCode, BarCode string
  Status: Active(1) / Inactive(2)
  Stock int32
  Ctime, Utime time.Time

ProductSpec:
  ID, ProductID int64
  Name string (e.g. "颜色")
  Values string (JSON array, e.g. ["红色","蓝色","黑色"])
  TenantID int64
```

**category.go：**

```
Category:
  ID, TenantID, ParentID int64
  Name string
  Level: 1/2/3
  Sort int32
  Icon string
  Status: Active(1) / Hidden(2)
  Children []Category (内存构建)
```

**brand.go：**

```
Brand:
  ID, TenantID int64
  Name, Logo string
  Status: Active(1) / Inactive(2)
```

### DAO 层（5 张表）

| 表名 | 关键索引 |
|------|---------|
| products | `idx_tenant_status(tenant_id, status)`, `idx_tenant_category(tenant_id, category_id)` |
| product_skus | `idx_product(product_id)`, `uniqueIndex:uk_sku_code(tenant_id, sku_code)` |
| product_specs | `idx_product(product_id)` |
| categories | `idx_tenant_parent(tenant_id, parent_id)` |
| brands | `idx_tenant(tenant_id)`, `uniqueIndex:uk_tenant_name(tenant_id, name)` |

SKU 和 Spec 在 SPU 创建/更新时同一事务处理（先删后插）。

### Cache 层

| Key | TTL | 说明 |
|-----|-----|------|
| `product:info:{id}` | 15min | Product + SKUs + Specs |
| `product:category:tree:{tenant_id}` | 30min | 完整分类树 |

## Service 层

单文件单接口（`ProductService`），包含：

### Product (SPU+SKU)

- `CreateProduct(ctx, product) → (Product, error)` — 检查配额 → 事务创建 SPU+SKU+Spec → IncrQuota → 发 product_updated 事件
- `GetProduct(ctx, id) → (Product, error)` — 含 SKUs + Specs
- `UpdateProduct(ctx, product) → error` — 事务更新 → 发 product_updated 事件
- `UpdateProductStatus(ctx, id, status) → error` — 发布/下架 → 发 product_status_changed 事件
- `ListProducts(ctx, tenantId, categoryId, status, page, pageSize) → ([]Product, int64, error)` — 不含 SKUs/Specs
- `BatchGetProducts(ctx, ids) → ([]Product, error)` — 含 SKUs
- `DeleteProduct(ctx, id) → error` — 删除 SPU+SKU+Spec → DecrQuota

### Category

- `CreateCategory(ctx, category) → (Category, error)` — 校验层级 ≤ 3
- `UpdateCategory(ctx, category) → error`
- `ListCategories(ctx, tenantId) → ([]Category, error)` — 返回树形结构（邻接表查全部 → 内存建树）
- `DeleteCategory(ctx, id) → error` — 校验无子分类且无商品引用

### Brand

- `CreateBrand(ctx, brand) → (Brand, error)`
- `UpdateBrand(ctx, brand) → error`
- `ListBrands(ctx, tenantId) → ([]Brand, error)`
- `DeleteBrand(ctx, id) → error` — 校验无商品引用

### Sales

- `IncrSales(ctx, productId, delta) → error`

## 跨服务依赖

product-svc → tenant-svc（gRPC client via etcd）：
- `CheckQuota(tenantId, "product_count")` — 创建前检查
- `IncrQuota(tenantId, "product_count", 1)` — 创建成功后递增
- `DecrQuota(tenantId, "product_count", 1)` — 删除成功后递减

## Kafka 事件

| 事件 | Topic | 触发时机 | 消费者 |
|------|-------|---------|--------|
| ProductStatusChanged | product_status_changed | 发布/下架 | search-svc |
| ProductUpdated | product_updated | 更新 SPU/SKU | search-svc, cart-svc |

## gRPC 层

`ProductGRPCServer` 实现 proto 中定义的 17 个 RPC。

关键点：
- tenant_id 从 gRPC metadata 获取（tenantx 拦截器）或从 request 参数获取
- ListProducts 返回不含 SKUs/Specs（性能优化）
- GetProduct 返回完整数据（含 SKUs + Specs）
- BatchGetProducts 含 SKUs，供 order/cart 服务批量查询

## 文件清单

| # | 文件 | 说明 |
|---|------|------|
| 1 | `domain/product.go` | SPU/SKU/Spec 实体 + 状态枚举 |
| 2 | `domain/category.go` | 分类实体 + 状态枚举 |
| 3 | `domain/brand.go` | 品牌实体 + 状态枚举 |
| 4 | `repository/dao/product.go` | Product + SKU + Spec DAO |
| 5 | `repository/dao/category.go` | Category DAO |
| 6 | `repository/dao/brand.go` | Brand DAO |
| 7 | `repository/dao/init.go` | AutoMigrate 5 张表 |
| 8 | `repository/cache/product.go` | Redis 缓存 |
| 9 | `repository/product.go` | Product CachedRepository |
| 10 | `repository/category.go` | Category Repository |
| 11 | `repository/brand.go` | Brand Repository |
| 12 | `service/product.go` | 业务逻辑（单文件单接口） |
| 13 | `events/types.go` | 事件 DTO |
| 14 | `events/producer.go` | Kafka Producer |
| 15 | `grpc/product.go` | 17 RPC handler |
| 16 | `ioc/db.go` | MySQL 初始化 |
| 17 | `ioc/redis.go` | Redis 初始化 |
| 18 | `ioc/kafka.go` | Kafka 初始化 |
| 19 | `ioc/logger.go` | Logger 初始化 |
| 20 | `ioc/grpc.go` | gRPC server + tenant-svc client |
| 21 | `config/dev.yaml` | 配置（port 8083, db mall_product） |
| 22 | `app.go` | App 聚合 |
| 23 | `wire.go` | Wire DI |
| 24 | `main.go` | 入口 |

共 24 个文件。
