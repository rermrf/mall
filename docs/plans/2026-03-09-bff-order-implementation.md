# BFF Order 接口实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在 merchant-bff 和 consumer-bff 中集成 order-svc 的订单管理和查询接口。

**Architecture:** 遵循现有 BFF 模式：ioc/grpc.go 新增 InitOrderClient → handler/order.go 实现 HTTP 处理 → ioc/gin.go 注册路由 → wire.go 注入依赖。consumer-bff 暴露 8 个接口（下单、查单、取消、确认收货、退款），merchant-bff 暴露 5 个接口（查单、发货、退款处理）。

**Tech Stack:** Go, Gin, gRPC, Wire DI, etcd service discovery

---

## 参考文件

| 文件 | 用途 |
|------|------|
| `docs/plans/2026-03-09-bff-order-design.md` | 设计文档 |
| `api/proto/gen/order/v1/order_grpc.pb.go` | gRPC 客户端接口 |
| `consumer-bff/handler/inventory.go` | Handler 模式参考 |
| `consumer-bff/ioc/grpc.go` | gRPC 客户端初始化参考 |
| `consumer-bff/ioc/gin.go` | 路由注册参考 |
| `consumer-bff/wire.go` | Wire DI 参考 |

---

## Task 1: consumer-bff — gRPC 客户端 + Handler + 路由 + Wire

**Files:**
- Create: `consumer-bff/handler/order.go`
- Modify: `consumer-bff/ioc/grpc.go:12-14,60-63`
- Modify: `consumer-bff/ioc/gin.go:1-52`
- Modify: `consumer-bff/wire.go:1-31`
- Regenerate: `consumer-bff/wire_gen.go`

### 1.1 consumer-bff/ioc/grpc.go — 新增 InitOrderClient

在 import 中添加 orderv1，在文件末尾添加函数：

```go
// import 块新增：
orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"

// 文件末尾新增：
func InitOrderClient(etcdClient *clientv3.Client) orderv1.OrderServiceClient {
	conn := initServiceConn(etcdClient, "order")
	return orderv1.NewOrderServiceClient(conn)
}
```

### 1.2 consumer-bff/handler/order.go — 新建

```go
package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type OrderHandler struct {
	orderClient orderv1.OrderServiceClient
	l           logger.Logger
}

func NewOrderHandler(orderClient orderv1.OrderServiceClient, l logger.Logger) *OrderHandler {
	return &OrderHandler{
		orderClient: orderClient,
		l:           l,
	}
}

type CreateOrderReq struct {
	Items     []CreateOrderItem `json:"items" binding:"required,min=1"`
	AddressID int64             `json:"address_id" binding:"required"`
	CouponID  int64             `json:"coupon_id"`
	Remark    string            `json:"remark"`
}

type CreateOrderItem struct {
	SkuID    int64 `json:"sku_id" binding:"required"`
	Quantity int32 `json:"quantity" binding:"required,min=1"`
}

func (h *OrderHandler) CreateOrder(ctx *gin.Context, req CreateOrderReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	tenantId, _ := ctx.Get("tenant_id")
	items := make([]*orderv1.CreateOrderItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, &orderv1.CreateOrderItem{
			SkuId:    item.SkuID,
			Quantity: item.Quantity,
		})
	}
	resp, err := h.orderClient.CreateOrder(ctx.Request.Context(), &orderv1.CreateOrderRequest{
		BuyerId:   uid.(int64),
		TenantId:  tenantId.(int64),
		Items:     items,
		AddressId: req.AddressID,
		CouponId:  req.CouponID,
		Remark:    req.Remark,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建订单失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"order_no":   resp.GetOrderNo(),
		"pay_amount": resp.GetPayAmount(),
	}}, nil
}

type ListOrdersReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *OrderHandler) ListOrders(ctx *gin.Context, req ListOrdersReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.orderClient.ListOrders(ctx.Request.Context(), &orderv1.ListOrdersRequest{
		BuyerId:  uid.(int64),
		TenantId: tenantId.(int64),
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询订单列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"orders": resp.GetOrders(),
		"total":  resp.GetTotal(),
	}}, nil
}

func (h *OrderHandler) GetOrder(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	if orderNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的订单号"})
		return
	}
	resp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{
		OrderNo: orderNo,
	})
	if err != nil {
		h.l.Error("查询订单详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetOrder()})
}

func (h *OrderHandler) CancelOrder(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	uid, _ := ctx.Get("uid")
	_, err := h.orderClient.CancelOrder(ctx.Request.Context(), &orderv1.CancelOrderRequest{
		OrderNo: orderNo,
		BuyerId: uid.(int64),
	})
	if err != nil {
		h.l.Error("取消订单失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

func (h *OrderHandler) ConfirmReceive(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	uid, _ := ctx.Get("uid")
	_, err := h.orderClient.ConfirmReceive(ctx.Request.Context(), &orderv1.ConfirmReceiveRequest{
		OrderNo: orderNo,
		BuyerId: uid.(int64),
	})
	if err != nil {
		h.l.Error("确认收货失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

type ApplyRefundReq struct {
	Type         int32  `json:"type" binding:"required,oneof=1 2"`
	RefundAmount int64  `json:"refund_amount" binding:"required,min=1"`
	Reason       string `json:"reason" binding:"required"`
}

func (h *OrderHandler) ApplyRefund(ctx *gin.Context, req ApplyRefundReq) (ginx.Result, error) {
	orderNo := ctx.Param("orderNo")
	uid, _ := ctx.Get("uid")
	resp, err := h.orderClient.ApplyRefund(ctx.Request.Context(), &orderv1.ApplyRefundRequest{
		OrderNo:      orderNo,
		BuyerId:      uid.(int64),
		Type:         req.Type,
		RefundAmount: req.RefundAmount,
		Reason:       req.Reason,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("申请退款失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"refund_no": resp.GetRefundNo(),
	}}, nil
}

type ListRefundsReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *OrderHandler) ListRefundOrders(ctx *gin.Context, req ListRefundsReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.orderClient.ListRefundOrders(ctx.Request.Context(), &orderv1.ListRefundOrdersRequest{
		TenantId: tenantId.(int64),
		BuyerId:  uid.(int64),
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询退款列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"refund_orders": resp.GetRefundOrders(),
		"total":         resp.GetTotal(),
	}}, nil
}

func (h *OrderHandler) GetRefundOrder(ctx *gin.Context) {
	refundNo := ctx.Param("refundNo")
	if refundNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的退款单号"})
		return
	}
	resp, err := h.orderClient.GetRefundOrder(ctx.Request.Context(), &orderv1.GetRefundOrderRequest{
		RefundNo: refundNo,
	})
	if err != nil {
		h.l.Error("查询退款详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetRefundOrder()})
}
```

### 1.3 consumer-bff/ioc/gin.go — 注入 OrderHandler + 注册路由

修改 InitGinServer 签名，新增 `orderHandler *handler.OrderHandler` 参数，在 auth 路由组末尾添加订单路由：

```go
func InitGinServer(
	jwtHandler *ijwt.JWTHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	inventoryHandler *handler.InventoryHandler,
	orderHandler *handler.OrderHandler,
	tenantClient tenantv1.TenantServiceClient,
	l logger.Logger,
) *gin.Engine {
```

在 auth 块末尾（`auth.POST("/inventory/stock/batch", ...)` 之后）添加：

```go
		// 订单
		auth.POST("/orders", ginx.WrapBody[handler.CreateOrderReq](l, orderHandler.CreateOrder))
		auth.GET("/orders", ginx.WrapQuery[handler.ListOrdersReq](l, orderHandler.ListOrders))
		auth.GET("/orders/:orderNo", orderHandler.GetOrder)
		auth.POST("/orders/:orderNo/cancel", orderHandler.CancelOrder)
		auth.POST("/orders/:orderNo/confirm", orderHandler.ConfirmReceive)
		auth.POST("/orders/:orderNo/refund", ginx.WrapBody[handler.ApplyRefundReq](l, orderHandler.ApplyRefund))
		auth.GET("/refunds", ginx.WrapQuery[handler.ListRefundsReq](l, orderHandler.ListRefundOrders))
		auth.GET("/refunds/:refundNo", orderHandler.GetRefundOrder)
```

### 1.4 consumer-bff/wire.go — 添加依赖

```go
var thirdPartySet = wire.NewSet(
	ioc.InitEtcdClient,
	ioc.InitLogger,
	ioc.InitUserClient,
	ioc.InitTenantClient,
	ioc.InitInventoryClient,
	ioc.InitOrderClient,
)

var handlerSet = wire.NewSet(
	ijwt.NewJWTHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	handler.NewInventoryHandler,
	handler.NewOrderHandler,
	ioc.InitGinServer,
)
```

### 1.5 验证

```bash
wire ./consumer-bff/
go build ./consumer-bff/...
go vet ./consumer-bff/...
```

---

## Task 2: merchant-bff — gRPC 客户端 + Handler + 路由 + Wire

**Files:**
- Create: `merchant-bff/handler/order.go`
- Modify: `merchant-bff/ioc/grpc.go:12-15,63-66`
- Modify: `merchant-bff/ioc/gin.go:1-67`
- Modify: `merchant-bff/wire.go:1-31`
- Regenerate: `merchant-bff/wire_gen.go`

### 2.1 merchant-bff/ioc/grpc.go — 新增 InitOrderClient

在 import 中添加 orderv1，在文件末尾添加函数：

```go
// import 块新增：
orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"

// 文件末尾新增：
func InitOrderClient(etcdClient *clientv3.Client) orderv1.OrderServiceClient {
	conn := initServiceConn(etcdClient, "order")
	return orderv1.NewOrderServiceClient(conn)
}
```

### 2.2 merchant-bff/handler/order.go — 新建

```go
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type OrderHandler struct {
	orderClient orderv1.OrderServiceClient
	l           logger.Logger
}

func NewOrderHandler(orderClient orderv1.OrderServiceClient, l logger.Logger) *OrderHandler {
	return &OrderHandler{
		orderClient: orderClient,
		l:           l,
	}
}

type ListOrdersReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *OrderHandler) ListOrders(ctx *gin.Context, req ListOrdersReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.orderClient.ListOrders(ctx.Request.Context(), &orderv1.ListOrdersRequest{
		TenantId: tenantId.(int64),
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询订单列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"orders": resp.GetOrders(),
		"total":  resp.GetTotal(),
	}}, nil
}

func (h *OrderHandler) GetOrder(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	if orderNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的订单号"})
		return
	}
	resp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{
		OrderNo: orderNo,
	})
	if err != nil {
		h.l.Error("查询订单详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetOrder()})
}

func (h *OrderHandler) ShipOrder(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	tenantId, _ := ctx.Get("tenant_id")
	uid, _ := ctx.Get("uid")
	_, err := h.orderClient.UpdateOrderStatus(ctx.Request.Context(), &orderv1.UpdateOrderStatusRequest{
		OrderNo:      orderNo,
		Status:       3, // shipped
		OperatorId:   uid.(int64),
		OperatorType: 2, // 商家
		Remark:       "商家发货",
	})
	_ = tenantId // tenant_id 通过 gRPC interceptor 传递
	if err != nil {
		h.l.Error("发货失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

type HandleRefundReq struct {
	RefundNo string `json:"refund_no" binding:"required"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

func (h *OrderHandler) HandleRefund(ctx *gin.Context, req HandleRefundReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	_, err := h.orderClient.HandleRefund(ctx.Request.Context(), &orderv1.HandleRefundRequest{
		RefundNo: req.RefundNo,
		TenantId: tenantId.(int64),
		Approved: req.Approved,
		Reason:   req.Reason,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("处理退款失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListRefundsReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *OrderHandler) ListRefundOrders(ctx *gin.Context, req ListRefundsReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.orderClient.ListRefundOrders(ctx.Request.Context(), &orderv1.ListRefundOrdersRequest{
		TenantId: tenantId.(int64),
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询退款列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"refund_orders": resp.GetRefundOrders(),
		"total":         resp.GetTotal(),
	}}, nil
}
```

### 2.3 merchant-bff/ioc/gin.go — 注入 OrderHandler + 注册路由

修改 InitGinServer 签名，新增 `orderHandler *handler.OrderHandler` 参数：

```go
func InitGinServer(
	jwtHandler *ijwt.JWTHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	inventoryHandler *handler.InventoryHandler,
	orderHandler *handler.OrderHandler,
	l logger.Logger,
) *gin.Engine {
```

在 auth 块末尾（`auth.GET("/inventory/logs", ...)` 之后）添加：

```go
		// 订单管理
		auth.GET("/orders", ginx.WrapQuery[handler.ListOrdersReq](l, orderHandler.ListOrders))
		auth.GET("/orders/:orderNo", orderHandler.GetOrder)
		auth.POST("/orders/:orderNo/ship", orderHandler.ShipOrder)
		auth.POST("/orders/:orderNo/refund/handle", ginx.WrapBody[handler.HandleRefundReq](l, orderHandler.HandleRefund))
		auth.GET("/refunds", ginx.WrapQuery[handler.ListRefundsReq](l, orderHandler.ListRefundOrders))
```

### 2.4 merchant-bff/wire.go — 添加依赖

```go
var thirdPartySet = wire.NewSet(
	ioc.InitEtcdClient,
	ioc.InitLogger,
	ioc.InitUserClient,
	ioc.InitTenantClient,
	ioc.InitInventoryClient,
	ioc.InitOrderClient,
)

var handlerSet = wire.NewSet(
	ijwt.NewJWTHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	handler.NewInventoryHandler,
	handler.NewOrderHandler,
	ioc.InitGinServer,
)
```

### 2.5 验证

```bash
wire ./merchant-bff/
go build ./merchant-bff/...
go vet ./merchant-bff/...
```

---

## 验证步骤

1. `wire ./consumer-bff/ && wire ./merchant-bff/` — Wire 生成成功
2. `go build ./consumer-bff/... ./merchant-bff/...` — 编译通过
3. `go vet ./consumer-bff/... ./merchant-bff/...` — 无警告

---

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `consumer-bff/handler/order.go` | 新建 | 8 方法：CreateOrder, ListOrders, GetOrder, CancelOrder, ConfirmReceive, ApplyRefund, ListRefundOrders, GetRefundOrder |
| 2 | `consumer-bff/ioc/grpc.go` | 修改 | +InitOrderClient |
| 3 | `consumer-bff/ioc/gin.go` | 修改 | +orderHandler 参数 + 8 路由 |
| 4 | `consumer-bff/wire.go` | 修改 | +InitOrderClient + NewOrderHandler |
| 5 | `consumer-bff/wire_gen.go` | 重新生成 | wire ./consumer-bff/ |
| 6 | `merchant-bff/handler/order.go` | 新建 | 5 方法：ListOrders, GetOrder, ShipOrder, HandleRefund, ListRefundOrders |
| 7 | `merchant-bff/ioc/grpc.go` | 修改 | +InitOrderClient |
| 8 | `merchant-bff/ioc/gin.go` | 修改 | +orderHandler 参数 + 5 路由 |
| 9 | `merchant-bff/wire.go` | 修改 | +InitOrderClient + NewOrderHandler |
| 10 | `merchant-bff/wire_gen.go` | 重新生成 | wire ./merchant-bff/ |
