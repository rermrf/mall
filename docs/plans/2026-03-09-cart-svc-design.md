# Cart Service + BFF 设计

## 概述

实现 cart-svc 购物车微服务（6 个 gRPC RPC）+ consumer-bff 购物车 HTTP 接口（6 个端点）。BFF 聚合 product-svc 商品信息返回给前端。

## 架构决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 存储 | MySQL + Redis Cache-Aside | 与项目其他服务模式一致 |
| Kafka | 无 | 购物车是简单 CRUD，不需要事件 |
| ID 生成 | MySQL 自增 | 无需 Snowflake |
| 幂等 | 无需 | AddItem 已存在则累加数量，天然幂等 |
| BFF | 仅 consumer-bff | 购物车是消费者功能 |

## 数据模型

### cart_items 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 | 主键自增 |
| user_id | int64 | 用户 ID |
| sku_id | int64 | SKU ID |
| product_id | int64 | 商品 ID |
| tenant_id | int64 | 租户 ID |
| quantity | int32 | 数量 |
| selected | bool | 是否勾选 |
| ctime | int64 | 创建时间（毫秒） |
| utime | int64 | 更新时间（毫秒） |

索引：`uniqueIndex:uk_user_sku(UserId, SkuId)`、`index:idx_user(UserId)`

### Cache 策略

- Key：`cart:items:{userId}`
- 内容：整个购物车列表 JSON
- TTL：30 分钟
- 写操作删除缓存，读操作重建

## cart-svc RPC（6 个）

| RPC | 说明 |
|-----|------|
| AddItem | 添加商品，已存在同 SKU 则累加数量 |
| UpdateItem | 更新数量或勾选状态 |
| RemoveItem | 删除单个 SKU |
| GetCart | 获取用户购物车所有 item |
| ClearCart | 清空购物车 |
| BatchRemoveItems | 批量删除（下单后清除已购 SKU） |

## Consumer BFF 接口（6 个）

| HTTP 方法 | 路由 | 说明 |
|-----------|------|------|
| POST | `/api/v1/cart/items` | 加入购物车 |
| PUT | `/api/v1/cart/items/:skuId` | 更新数量/勾选 |
| DELETE | `/api/v1/cart/items/:skuId` | 删除单个 |
| GET | `/api/v1/cart` | 获取购物车（BFF 聚合商品信息） |
| DELETE | `/api/v1/cart` | 清空 |
| POST | `/api/v1/cart/batch-remove` | 批量删除 |

### BFF GetCart 聚合

1. cartClient.GetCart(userId) → 基础 item 列表
2. productClient.BatchGetProducts(productIds) → 商品+SKU 信息
3. 填充 product_name、product_image、sku_spec、price、stock
4. 返回聚合后的完整数据

## 基础设施

- gRPC 端口：8087
- 服务名：cart
- Redis DB：6
- 数据库：mall_cart

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `cart/domain/cart.go` | 新建 | CartItem 域模型 |
| 2 | `cart/repository/dao/cart.go` | 新建 | GORM 模型 + CartDAO |
| 3 | `cart/repository/dao/init.go` | 新建 | AutoMigrate |
| 4 | `cart/repository/cache/cart.go` | 新建 | Redis 缓存 |
| 5 | `cart/repository/cart.go` | 新建 | Repository |
| 6 | `cart/service/cart.go` | 新建 | CartService |
| 7 | `cart/grpc/cart.go` | 新建 | 6 RPC Handler |
| 8 | `cart/ioc/db.go` | 新建 | MySQL |
| 9 | `cart/ioc/redis.go` | 新建 | Redis |
| 10 | `cart/ioc/logger.go` | 新建 | Logger |
| 11 | `cart/ioc/grpc.go` | 新建 | etcd + gRPC Server |
| 12 | `cart/wire.go` | 新建 | Wire DI |
| 13 | `cart/app.go` | 新建 | App |
| 14 | `cart/main.go` | 新建 | 入口 |
| 15 | `cart/config/dev.yaml` | 新建 | 配置 |
| 16 | `cart/wire_gen.go` | 生成 | Wire |
| 17 | `consumer-bff/handler/cart.go` | 新建 | CartHandler + 6 方法 |
| 18 | `consumer-bff/ioc/grpc.go` | 修改 | +InitCartClient + InitProductClient |
| 19 | `consumer-bff/ioc/gin.go` | 修改 | +cartHandler + 6 路由 |
| 20 | `consumer-bff/wire.go` | 修改 | +InitCartClient + InitProductClient + NewCartHandler |

共 20 个文件（17 新建 + 3 修改）+ 2 个 wire_gen.go 重新生成。
