# Search Service + BFF 设计

## 概述

实现 search-svc 搜索微服务（7 个 gRPC RPC）+ Kafka Consumer 商品同步 + consumer-bff 搜索 HTTP 接口（5 个端点）。基于 Elasticsearch 实现全文搜索、搜索建议、热搜词、搜索历史。

## 架构决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 搜索引擎 | Elasticsearch + IK 分词 | 支持中文全文搜索、suggest、聚合 |
| 数据同步 | Kafka Consumer | 监听 product_updated / product_status_changed 事件 |
| 热搜词 | Redis Sorted Set | ZINCRBY 计数，ZREVRANGE 取 Top N |
| 搜索历史 | Redis List | LPUSH + LTRIM 保留最近 20 条 |
| 搜索建议 | ES completion suggester | 无需额外存储 |
| BFF | 仅 consumer-bff | 搜索是消费者功能 |
| MySQL | 无 | 搜索服务无需关系数据库 |

## 数据模型

### ES 索引（`mall_product`）

| 字段 | ES 类型 | 说明 |
|------|---------|------|
| id | long | 商品 ID |
| tenant_id | long | 租户 ID（filter） |
| name | text (ik_max_word) | 商品名，全文搜索 |
| subtitle | text (ik_smart) | 副标题 |
| category_id | long | 分类 ID |
| category_name | keyword | 分类名 |
| brand_id | long | 品牌 ID |
| brand_name | keyword | 品牌名 |
| price | long | 最低 SKU 价格（分） |
| sales | long | 销量 |
| main_image | keyword | 主图 URL |
| status | integer | 状态（仅索引 status=2 上架） |
| shop_id | long | 店铺 ID |
| shop_name | keyword | 店铺名 |

### Redis 缓存策略

- 热搜词：`search:hot` Sorted Set，搜索时 ZINCRBY，定时截取 Top N
- 搜索历史：`search:history:{userId}` List，LPUSH + LTRIM 保留最近 20 条
- 搜索建议：直接用 ES completion suggester，不额外缓存

## 数据同步流程

1. product-svc 发 Kafka 事件（`product_updated` / `product_status_changed`）
2. search-svc Kafka Consumer 消费事件
3. Consumer 调用 product-svc gRPC `GetProduct` 获取完整商品数据
4. 将 Product 转换为 ProductDocument 写入 ES（status=2 同步，非 2 则从 ES 删除）

## search-svc RPC（7 个）

| RPC | 说明 | 实现方式 |
|-----|------|---------|
| SearchProducts | 全文搜索 + 筛选 + 排序 + 分页 | ES bool query + sort |
| GetSuggestions | 输入联想 | ES completion suggester |
| GetHotWords | 热搜词 Top N | Redis ZREVRANGE |
| GetSearchHistory | 用户搜索历史 | Redis LRANGE |
| ClearSearchHistory | 清空搜索历史 | Redis DEL |
| SyncProduct | 同步商品到 ES | ES Index |
| DeleteProduct | 从 ES 删除商品 | ES Delete |

## Kafka Consumer（2 个 topic）

| Topic | 处理逻辑 |
|-------|---------|
| product_updated | 调 product-svc GetProduct → SyncProduct 写入 ES |
| product_status_changed | status=2 则 Sync，否则 Delete |

## Consumer BFF 接口（5 个）

| HTTP 方法 | 路由 | 说明 | 认证 |
|-----------|------|------|------|
| GET | `/api/v1/search` | 搜索商品 | 公开（登录时记录历史） |
| GET | `/api/v1/search/suggestions` | 搜索建议 | 公开 |
| GET | `/api/v1/search/hot` | 热搜词 | 公开 |
| GET | `/api/v1/search/history` | 搜索历史 | 需登录 |
| DELETE | `/api/v1/search/history` | 清空历史 | 需登录 |

> 搜索和建议/热搜放 pub 路由组（无需登录），历史放 auth 路由组。搜索时如果用户已登录则异步记录搜索历史。

## 基础设施

- gRPC 端口：8088
- 服务名：search
- Redis DB：7
- ES 索引：mall_product

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `search/domain/search.go` | 新建 | 域模型 |
| 2 | `search/repository/dao/search.go` | 新建 | ES DAO |
| 3 | `search/repository/dao/init.go` | 新建 | ES 索引初始化 |
| 4 | `search/repository/cache/search.go` | 新建 | Redis（热搜词+历史） |
| 5 | `search/repository/search.go` | 新建 | Repository |
| 6 | `search/service/search.go` | 新建 | SearchService |
| 7 | `search/grpc/search.go` | 新建 | 7 RPC Handler |
| 8 | `search/events/consumer.go` | 新建 | Kafka Consumer |
| 9 | `search/events/types.go` | 新建 | 事件类型 |
| 10 | `search/ioc/es.go` | 新建 | ES 客户端初始化 |
| 11 | `search/ioc/redis.go` | 新建 | Redis |
| 12 | `search/ioc/logger.go` | 新建 | Logger |
| 13 | `search/ioc/grpc.go` | 新建 | etcd + gRPC Server |
| 14 | `search/ioc/kafka.go` | 新建 | Kafka Consumer |
| 15 | `search/ioc/product_client.go` | 新建 | product-svc gRPC 客户端 |
| 16 | `search/wire.go` | 新建 | Wire DI |
| 17 | `search/app.go` | 新建 | App |
| 18 | `search/main.go` | 新建 | 入口 |
| 19 | `search/config/dev.yaml` | 新建 | 配置 |
| 20 | `search/wire_gen.go` | 生成 | Wire |
| 21 | `consumer-bff/handler/search.go` | 新建 | SearchHandler + 5 方法 |
| 22 | `consumer-bff/ioc/grpc.go` | 修改 | +InitSearchClient |
| 23 | `consumer-bff/ioc/gin.go` | 修改 | +searchHandler + 5 路由 |
| 24 | `consumer-bff/wire.go` | 修改 | +InitSearchClient + NewSearchHandler |
| 25 | `consumer-bff/wire_gen.go` | 重新生成 | Wire |

共 25 个文件（19 新建 + 3 修改 + 1 生成 + 2 重新生成）。
