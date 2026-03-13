# BFF Inventory 接口实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在 merchant-bff 和 consumer-bff 中集成 inventory-svc 的库存管理和查询接口。

**Architecture:** 遵循现有 BFF 模式：ioc/grpc.go 新增 InitInventoryClient → handler/inventory.go 实现 HTTP 处理 → ioc/gin.go 注册路由 → wire.go 注入依赖。merchant-bff 暴露 4 个接口（SetStock, GetStock, BatchGetStock, ListLogs），consumer-bff 暴露 2 个接口（GetStock, BatchGetStock）。

**Tech Stack:** Go, Gin, gRPC, Wire DI, etcd service discovery

---

## 参考文件

| 文件 | 用途 |
|------|------|
| `docs/plans/2026-03-08-bff-inventory-design.md` | 设计文档 |
| `api/proto/gen/inventory/v1/inventory_grpc.pb.go` | gRPC 客户端接口 |
| `merchant-bff/handler/tenant.go` | Handler 模式参考 |
| `merchant-bff/ioc/grpc.go` | gRPC 客户端初始化参考 |
| `merchant-bff/ioc/gin.go` | 路由注册参考 |
| `merchant-bff/wire.go` | Wire DI 参考 |

---

## Task 1: merchant-bff — gRPC 客户端 + Handler + 路由 + Wire

**Files:**
- Create: `merchant-bff/handler/inventory.go`
- Modify: `merchant-bff/ioc/grpc.go:12-14,57-60`
- Modify: `merchant-bff/ioc/gin.go:1-60`
- Modify: `merchant-bff/wire.go:1-29`
- Regenerate: `merchant-bff/wire_gen.go`

### 1.1 merchant-bff/ioc/grpc.go — 新增 InitInventoryClient

在 import 中添加 inventoryv1，在文件末尾添加函数：

```go
// import 块新增：
inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"

// 文件末尾新增：
func InitInventoryClient(etcdClient *clientv3.Client) inventoryv1.InventoryServiceClient {
	conn := initServiceConn(etcdClient, "inventory")
	return inventoryv1.NewInventoryServiceClient(conn)
}
```

### 1.2 merchant-bff/handler/inventory.go — 新建

```go
package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type InventoryHandler struct {
	inventoryClient inventoryv1.InventoryServiceClient
	l               logger.Logger
}

func NewInventoryHandler(inventoryClient inventoryv1.InventoryServiceClient, l logger.Logger) *InventoryHandler {
	return &InventoryHandler{
		inventoryClient: inventoryClient,
		l:               l,
	}
}

type SetStockReq struct {
	SkuId          int64 `json:"sku_id" binding:"required"`
	Total          int32 `json:"total" binding:"required,min=0"`
	AlertThreshold int32 `json:"alert_threshold" binding:"min=0"`
}

func (h *InventoryHandler) SetStock(ctx *gin.Context, req SetStockReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	_, err := h.inventoryClient.SetStock(ctx.Request.Context(), &inventoryv1.SetStockRequest{
		TenantId:       tenantId.(int64),
		SkuId:          req.SkuId,
		Total:          req.Total,
		AlertThreshold: req.AlertThreshold,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("设置库存失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

func (h *InventoryHandler) GetStock(ctx *gin.Context) {
	skuIdStr := ctx.Param("skuId")
	skuId, err := strconv.ParseInt(skuIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的 skuId"})
		return
	}
	resp, err := h.inventoryClient.GetStock(ctx.Request.Context(), &inventoryv1.GetStockRequest{
		SkuId: skuId,
	})
	if err != nil {
		h.l.Error("查询库存失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetInventory()})
}

type BatchGetStockReq struct {
	SkuIds []int64 `json:"sku_ids" binding:"required,min=1"`
}

func (h *InventoryHandler) BatchGetStock(ctx *gin.Context, req BatchGetStockReq) (ginx.Result, error) {
	resp, err := h.inventoryClient.BatchGetStock(ctx.Request.Context(), &inventoryv1.BatchGetStockRequest{
		SkuIds: req.SkuIds,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("批量查询库存失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetInventories()}, nil
}

type ListLogsReq struct {
	SkuId    int64 `form:"sku_id"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *InventoryHandler) ListLogs(ctx *gin.Context, req ListLogsReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.inventoryClient.ListLogs(ctx.Request.Context(), &inventoryv1.ListLogsRequest{
		TenantId: tenantId.(int64),
		SkuId:    req.SkuId,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询库存日志失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"logs":  resp.GetLogs(),
		"total": resp.GetTotal(),
	}}, nil
}
```

### 1.3 merchant-bff/ioc/gin.go — 注入 InventoryHandler + 注册路由

修改 InitGinServer 签名，新增 `inventoryHandler *handler.InventoryHandler` 参数，在 auth 路由组末尾添加库存路由：

```go
func InitGinServer(
	jwtHandler *ijwt.JWTHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	inventoryHandler *handler.InventoryHandler,
	l logger.Logger,
) *gin.Engine {
```

在 auth 块末尾（`auth.GET("/quotas/:type", ...)` 之后）添加：

```go
		// 库存管理
		auth.POST("/inventory/stock", ginx.WrapBody[handler.SetStockReq](l, inventoryHandler.SetStock))
		auth.GET("/inventory/stock/:skuId", inventoryHandler.GetStock)
		auth.POST("/inventory/stock/batch", ginx.WrapBody[handler.BatchGetStockReq](l, inventoryHandler.BatchGetStock))
		auth.GET("/inventory/logs", ginx.WrapQuery[handler.ListLogsReq](l, inventoryHandler.ListLogs))
```

### 1.4 merchant-bff/wire.go — 添加依赖

```go
var thirdPartySet = wire.NewSet(
	ioc.InitEtcdClient,
	ioc.InitLogger,
	ioc.InitUserClient,
	ioc.InitTenantClient,
	ioc.InitInventoryClient,
)

var handlerSet = wire.NewSet(
	ijwt.NewJWTHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	handler.NewInventoryHandler,
	ioc.InitGinServer,
)
```

### 1.5 验证

```bash
wire ./merchant-bff/
go build ./merchant-bff/...
go vet ./merchant-bff/...
```

---

## Task 2: consumer-bff — gRPC 客户端 + Handler + 路由 + Wire

**Files:**
- Create: `consumer-bff/handler/inventory.go`
- Modify: `consumer-bff/ioc/grpc.go:12-14,54-57`
- Modify: `consumer-bff/ioc/gin.go:1-48`
- Modify: `consumer-bff/wire.go:1-29`
- Regenerate: `consumer-bff/wire_gen.go`

### 2.1 consumer-bff/ioc/grpc.go — 新增 InitInventoryClient

在 import 中添加 inventoryv1，在文件末尾添加函数：

```go
// import 块新增：
inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"

// 文件末尾新增：
func InitInventoryClient(etcdClient *clientv3.Client) inventoryv1.InventoryServiceClient {
	conn := initServiceConn(etcdClient, "inventory")
	return inventoryv1.NewInventoryServiceClient(conn)
}
```

### 2.2 consumer-bff/handler/inventory.go — 新建

```go
package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type InventoryHandler struct {
	inventoryClient inventoryv1.InventoryServiceClient
	l               logger.Logger
}

func NewInventoryHandler(inventoryClient inventoryv1.InventoryServiceClient, l logger.Logger) *InventoryHandler {
	return &InventoryHandler{
		inventoryClient: inventoryClient,
		l:               l,
	}
}

func (h *InventoryHandler) GetStock(ctx *gin.Context) {
	skuIdStr := ctx.Param("skuId")
	skuId, err := strconv.ParseInt(skuIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的 skuId"})
		return
	}
	resp, err := h.inventoryClient.GetStock(ctx.Request.Context(), &inventoryv1.GetStockRequest{
		SkuId: skuId,
	})
	if err != nil {
		h.l.Error("查询库存失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetInventory()})
}

type BatchGetStockReq struct {
	SkuIds []int64 `json:"sku_ids" binding:"required,min=1"`
}

func (h *InventoryHandler) BatchGetStock(ctx *gin.Context, req BatchGetStockReq) (ginx.Result, error) {
	resp, err := h.inventoryClient.BatchGetStock(ctx.Request.Context(), &inventoryv1.BatchGetStockRequest{
		SkuIds: req.SkuIds,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("批量查询库存失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetInventories()}, nil
}
```

### 2.3 consumer-bff/ioc/gin.go — 注入 InventoryHandler + 注册路由

修改 InitGinServer 签名，新增 `inventoryHandler *handler.InventoryHandler` 参数：

```go
func InitGinServer(
	jwtHandler *ijwt.JWTHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	inventoryHandler *handler.InventoryHandler,
	tenantClient tenantv1.TenantServiceClient,
	l logger.Logger,
) *gin.Engine {
```

在 auth 块末尾（`auth.DELETE("/addresses/:id", ...)` 之后）添加：

```go
		// 库存查询
		auth.GET("/inventory/stock/:skuId", inventoryHandler.GetStock)
		auth.POST("/inventory/stock/batch", ginx.WrapBody[handler.BatchGetStockReq](l, inventoryHandler.BatchGetStock))
```

### 2.4 consumer-bff/wire.go — 添加依赖

```go
var thirdPartySet = wire.NewSet(
	ioc.InitEtcdClient,
	ioc.InitLogger,
	ioc.InitUserClient,
	ioc.InitTenantClient,
	ioc.InitInventoryClient,
)

var handlerSet = wire.NewSet(
	ijwt.NewJWTHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	handler.NewInventoryHandler,
	ioc.InitGinServer,
)
```

### 2.5 验证

```bash
wire ./consumer-bff/
go build ./consumer-bff/...
go vet ./consumer-bff/...
```

---

## 验证步骤

1. `wire ./merchant-bff/ && wire ./consumer-bff/` — Wire 生成成功
2. `go build ./merchant-bff/... ./consumer-bff/...` — 编译通过
3. `go vet ./merchant-bff/... ./consumer-bff/...` — 无警告

---

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `merchant-bff/handler/inventory.go` | 新建 | 4 方法：SetStock, GetStock, BatchGetStock, ListLogs |
| 2 | `merchant-bff/ioc/grpc.go` | 修改 | +InitInventoryClient |
| 3 | `merchant-bff/ioc/gin.go` | 修改 | +inventoryHandler 参数 + 4 路由 |
| 4 | `merchant-bff/wire.go` | 修改 | +InitInventoryClient + NewInventoryHandler |
| 5 | `merchant-bff/wire_gen.go` | 重新生成 | wire ./merchant-bff/ |
| 6 | `consumer-bff/handler/inventory.go` | 新建 | 2 方法：GetStock, BatchGetStock |
| 7 | `consumer-bff/ioc/grpc.go` | 修改 | +InitInventoryClient |
| 8 | `consumer-bff/ioc/gin.go` | 修改 | +inventoryHandler 参数 + 2 路由 |
| 9 | `consumer-bff/wire.go` | 修改 | +InitInventoryClient + NewInventoryHandler |
| 10 | `consumer-bff/wire_gen.go` | 重新生成 | wire ./consumer-bff/ |
