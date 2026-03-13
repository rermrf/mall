# Consumer-BFF 域名→租户 Redis 缓存设计

## 背景

Consumer-BFF 的 `TenantResolve` 中间件在每个 HTTP 请求上都通过 gRPC 调用 tenant-service 的 `GetShopByDomain` 来解析域名对应的租户。在高并发场景下，这带来不必要的延迟和对 tenant-service 的压力。

域名→租户的映射变更频率极低（商户基本不换域名），非常适合缓存。

## 方案

在 `TenantResolve` 中间件中增加 Redis 缓存层。

### 缓存结构

```
Key:    tenant:domain:{domain}
Value:  JSON {"tenant_id": 123, "shop_id": 456, "shop_name": "..."}
TTL:    10 分钟（正常） / 1 分钟（空标记，防穿透）
```

### 请求流程

```
HTTP 请求 → 提取 domain (Host 头 / X-Tenant-Domain)
  ↓
查 Redis GET tenant:domain:{domain}
  ├─ 命中且 tenant_id > 0 → 反序列化，注入 ctx，继续
  ├─ 命中且 tenant_id == 0 → 返回 404 店铺不存在
  └─ 未命中 (nil) → gRPC GetShopByDomain
       ├─ 成功 → SET tenant:domain:{domain} (TTL 10min)，注入 ctx
       └─ 失败 → SET 空标记 {"tenant_id":0} (TTL 1min)，返回 404
```

### 缓存失效

采用 TTL 自然过期，不做主动失效。域名映射变更后最多 10 分钟生效。

## 改动范围

| 文件 | 改动说明 |
|------|---------|
| `consumer-bff/handler/middleware/tenant_resolve.go` | 注入 `redis.Cmdable`，构造函数增加参数；`Build()` 内增加 Redis 查询/回写逻辑 |
| `consumer-bff/ioc/gin.go` | `NewTenantResolve` 调用增加 redis 参数传递 |
| `consumer-bff/wire_gen.go` | 更新 wire 生成代码，传递 redis 实例 |

### 不需要改动

- tenant-service（不做主动失效）
- 其他 BFF（merchant-bff / admin-bff 不用域名解析）
- proto 定义

## 缓存数据结构

```go
type domainCacheEntry struct {
    TenantID int64  `json:"tenant_id"`
    ShopID   int64  `json:"shop_id"`
    ShopName string `json:"shop_name"`
    Logo     string `json:"logo"`
    Domain   string `json:"domain"`
}
```

只缓存必要字段，不缓存整个 proto 对象（避免 proto 字段变更导致反序列化失败）。
