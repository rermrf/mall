# BFF 库存接口设计

## 概述

在 merchant-bff 和 consumer-bff 中集成 inventory-svc 的库存接口。merchant-bff 提供完整的库存管理能力（设置、查询、日志），consumer-bff 提供只读库存查询。Deduct/Confirm/Rollback 不在 BFF 暴露，由未来的 order-svc 内部 gRPC 调用。

## 接口分配

### merchant-bff（商户库存管理）

| 方法 | 路由 | RPC | 说明 |
|------|------|-----|------|
| POST | `/api/v1/inventory/stock` | SetStock | 设置/更新 SKU 库存 |
| GET | `/api/v1/inventory/stock/:skuId` | GetStock | 查询单个 SKU 库存 |
| POST | `/api/v1/inventory/stock/batch` | BatchGetStock | 批量查询库存 |
| GET | `/api/v1/inventory/logs` | ListLogs | 查询库存变更日志 |

全部在 auth 路由组（JWT + 租户中间件）。

### consumer-bff（消费者查询库存）

| 方法 | 路由 | RPC | 说明 |
|------|------|-----|------|
| GET | `/api/v1/inventory/stock/:skuId` | GetStock | 查看商品库存 |
| POST | `/api/v1/inventory/stock/batch` | BatchGetStock | 批量查看库存（列表页） |

全部在 auth 路由组（JWT + 域名租户解析）。

### 不暴露的 RPC

Deduct / Confirm / Rollback — 由未来的 order-svc 内部 gRPC 直接调用 inventory-svc。

## 文件变更清单

### merchant-bff

| 文件 | 操作 | 说明 |
|------|------|------|
| `merchant-bff/handler/inventory.go` | 新建 | InventoryHandler：4 个方法 |
| `merchant-bff/ioc/grpc.go` | 修改 | 新增 InitInventoryClient |
| `merchant-bff/ioc/gin.go` | 修改 | 注入 InventoryHandler，注册 4 条路由 |
| `merchant-bff/wire.go` | 修改 | 加 InitInventoryClient + NewInventoryHandler |
| `merchant-bff/wire_gen.go` | 重新生成 | wire ./merchant-bff/ |

### consumer-bff

| 文件 | 操作 | 说明 |
|------|------|------|
| `consumer-bff/handler/inventory.go` | 新建 | InventoryHandler：2 个方法 |
| `consumer-bff/ioc/grpc.go` | 修改 | 新增 InitInventoryClient |
| `consumer-bff/ioc/gin.go` | 修改 | 注入 InventoryHandler，注册 2 条路由 |
| `consumer-bff/wire.go` | 修改 | 加 InitInventoryClient + NewInventoryHandler |
| `consumer-bff/wire_gen.go` | 重新生成 | wire ./consumer-bff/ |

总计：2 个新文件 + 6 个修改文件 + 2 个重新生成。

## Request/Response 结构

### merchant-bff 请求体

```go
// SetStock — POST /api/v1/inventory/stock
type SetStockReq struct {
    SkuId          int64 `json:"sku_id" binding:"required"`
    Total          int32 `json:"total" binding:"required,min=0"`
    AlertThreshold int32 `json:"alert_threshold" binding:"min=0"`
}

// GetStock — GET /api/v1/inventory/stock/:skuId
// skuId 从 URL path 提取，无请求体

// BatchGetStock — POST /api/v1/inventory/stock/batch
type BatchGetStockReq struct {
    SkuIds []int64 `json:"sku_ids" binding:"required,min=1"`
}

// ListLogs — GET /api/v1/inventory/logs
type ListLogsReq struct {
    SkuId    int64 `form:"sku_id"`
    Page     int32 `form:"page" binding:"required,min=1"`
    PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}
```

### consumer-bff 请求体

```go
// GetStock — GET /api/v1/inventory/stock/:skuId
// skuId 从 URL path 提取

// BatchGetStock — POST /api/v1/inventory/stock/batch
type BatchGetStockReq struct {
    SkuIds []int64 `json:"sku_ids" binding:"required,min=1"`
}
```

### 响应格式

统一 `ginx.Result{Code: 0, Msg: "success", Data: ...}`：

- SetStock → Data: nil
- GetStock → Data: inventory 对象
- BatchGetStock → Data: inventories 数组
- ListLogs → Data: {logs 数组, total}

### tenant_id 获取方式

- merchant-bff: `ctx.Get("tenant_id")` — JWT claims 提取
- consumer-bff: `ctx.Get("tenant_id")` — 域名解析中间件注入
