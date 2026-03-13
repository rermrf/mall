# BFF Payment 接口实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在 consumer-bff 和 merchant-bff 中添加 payment 相关 HTTP 接口，连接 payment-svc gRPC 服务。

**Architecture:** 遵循现有 BFF 模式：ioc/grpc.go 初始化 gRPC client → handler/payment.go 实现 HTTP handler → ioc/gin.go 注册路由 → wire.go 更新 DI。每个 BFF 独立修改，互不影响。

**Tech Stack:** Go, Gin, gRPC, Wire DI, etcd service discovery

---

## 参考文件

| 文件 | 用途 |
|------|------|
| `docs/plans/2026-03-09-bff-payment-design.md` | 设计文档 |
| `api/proto/payment/v1/payment.proto` | Proto 定义（7 RPC） |
| `api/proto/gen/payment/v1/payment_grpc.pb.go` | gRPC 生成代码 |
| `consumer-bff/handler/order.go` | Consumer handler 模式参考 |
| `consumer-bff/ioc/grpc.go` | Consumer gRPC client 初始化参考 |
| `consumer-bff/ioc/gin.go` | Consumer 路由注册参考 |
| `consumer-bff/wire.go` | Consumer Wire DI 参考 |
| `merchant-bff/handler/order.go` | Merchant handler 模式参考 |
| `merchant-bff/ioc/grpc.go` | Merchant gRPC client 初始化参考 |
| `merchant-bff/ioc/gin.go` | Merchant 路由注册参考 |
| `merchant-bff/wire.go` | Merchant Wire DI 参考 |

---

## Task 1: Consumer BFF — Payment Handler + gRPC Client + 路由 + Wire

**Files:**
- Create: `consumer-bff/handler/payment.go`
- Modify: `consumer-bff/ioc/grpc.go`
- Modify: `consumer-bff/ioc/gin.go`
- Modify: `consumer-bff/wire.go`
- Regenerate: `consumer-bff/wire_gen.go`

### 1.1 consumer-bff/handler/payment.go

```go
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type PaymentHandler struct {
	paymentClient paymentv1.PaymentServiceClient
	l             logger.Logger
}

func NewPaymentHandler(paymentClient paymentv1.PaymentServiceClient, l logger.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentClient: paymentClient,
		l:             l,
	}
}

type CreatePaymentReq struct {
	OrderID int64  `json:"order_id" binding:"required"`
	OrderNo string `json:"order_no" binding:"required"`
	Channel string `json:"channel" binding:"required,oneof=mock wechat alipay"`
	Amount  int64  `json:"amount" binding:"required,min=1"`
}

func (h *PaymentHandler) CreatePayment(ctx *gin.Context, req CreatePaymentReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.paymentClient.CreatePayment(ctx.Request.Context(), &paymentv1.CreatePaymentRequest{
		TenantId: tenantId.(int64),
		OrderId:  req.OrderID,
		OrderNo:  req.OrderNo,
		Channel:  req.Channel,
		Amount:   req.Amount,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建支付单失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"payment_no": resp.GetPaymentNo(),
		"pay_url":    resp.GetPayUrl(),
	}}, nil
}

func (h *PaymentHandler) GetPayment(ctx *gin.Context) {
	paymentNo := ctx.Param("paymentNo")
	if paymentNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的支付单号"})
		return
	}
	resp, err := h.paymentClient.GetPayment(ctx.Request.Context(), &paymentv1.GetPaymentRequest{
		PaymentNo: paymentNo,
	})
	if err != nil {
		h.l.Error("查询支付状态失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetPayment()})
}

type HandleNotifyReq struct {
	Channel    string `json:"channel" binding:"required"`
	NotifyBody string `json:"notify_body" binding:"required"`
}

func (h *PaymentHandler) HandleNotify(ctx *gin.Context, req HandleNotifyReq) (ginx.Result, error) {
	resp, err := h.paymentClient.HandleNotify(ctx.Request.Context(), &paymentv1.HandleNotifyRequest{
		Channel:    req.Channel,
		NotifyBody: req.NotifyBody,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("处理支付回调失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"success": resp.GetSuccess(),
	}}, nil
}
```

### 1.2 consumer-bff/ioc/grpc.go — 添加 InitPaymentClient

在文件末尾（`InitOrderClient` 函数之后）追加：

```go
paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"  // 添加到 import

func InitPaymentClient(etcdClient *clientv3.Client) paymentv1.PaymentServiceClient {
	conn := initServiceConn(etcdClient, "payment")
	return paymentv1.NewPaymentServiceClient(conn)
}
```

### 1.3 consumer-bff/ioc/gin.go — 添加 paymentHandler 参数 + 3 路由

函数签名添加 `paymentHandler *handler.PaymentHandler` 参数。

在 auth 路由组的订单路由之后追加：

```go
// 支付
auth.POST("/payments", ginx.WrapBody[handler.CreatePaymentReq](l, paymentHandler.CreatePayment))
auth.GET("/payments/:paymentNo", paymentHandler.GetPayment)
auth.POST("/payments/notify", ginx.WrapBody[handler.HandleNotifyReq](l, paymentHandler.HandleNotify))
```

### 1.4 consumer-bff/wire.go — 添加 InitPaymentClient + NewPaymentHandler

```go
// thirdPartySet 添加:
ioc.InitPaymentClient,

// handlerSet 添加:
handler.NewPaymentHandler,
```

### 1.5 验证

```bash
go build ./consumer-bff/...
go vet ./consumer-bff/...
```

---

## Task 2: Merchant BFF — Payment Handler + gRPC Client + 路由 + Wire

**Files:**
- Create: `merchant-bff/handler/payment.go`
- Modify: `merchant-bff/ioc/grpc.go`
- Modify: `merchant-bff/ioc/gin.go`
- Modify: `merchant-bff/wire.go`
- Regenerate: `merchant-bff/wire_gen.go`

### 2.1 merchant-bff/handler/payment.go

```go
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type PaymentHandler struct {
	paymentClient paymentv1.PaymentServiceClient
	l             logger.Logger
}

func NewPaymentHandler(paymentClient paymentv1.PaymentServiceClient, l logger.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentClient: paymentClient,
		l:             l,
	}
}

type ListPaymentsReq struct {
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *PaymentHandler) ListPayments(ctx *gin.Context, req ListPaymentsReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.paymentClient.ListPayments(ctx.Request.Context(), &paymentv1.ListPaymentsRequest{
		TenantId: tenantId.(int64),
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询支付列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"payments": resp.GetPayments(),
		"total":    resp.GetTotal(),
	}}, nil
}

func (h *PaymentHandler) GetPayment(ctx *gin.Context) {
	paymentNo := ctx.Param("paymentNo")
	if paymentNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的支付单号"})
		return
	}
	resp, err := h.paymentClient.GetPayment(ctx.Request.Context(), &paymentv1.GetPaymentRequest{
		PaymentNo: paymentNo,
	})
	if err != nil {
		h.l.Error("查询支付详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetPayment()})
}

type RefundReq struct {
	Amount int64  `json:"amount" binding:"required,min=1"`
	Reason string `json:"reason" binding:"required"`
}

func (h *PaymentHandler) Refund(ctx *gin.Context, req RefundReq) (ginx.Result, error) {
	paymentNo := ctx.Param("paymentNo")
	resp, err := h.paymentClient.Refund(ctx.Request.Context(), &paymentv1.RefundRequest{
		PaymentNo: paymentNo,
		Amount:    req.Amount,
		Reason:    req.Reason,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("发起退款失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"refund_no": resp.GetRefundNo(),
	}}, nil
}

func (h *PaymentHandler) GetRefund(ctx *gin.Context) {
	refundNo := ctx.Param("refundNo")
	if refundNo == "" {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的退款单号"})
		return
	}
	resp, err := h.paymentClient.GetRefund(ctx.Request.Context(), &paymentv1.GetRefundRequest{
		RefundNo: refundNo,
	})
	if err != nil {
		h.l.Error("查询退款详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetRefund()})
}
```

### 2.2 merchant-bff/ioc/grpc.go — 添加 InitPaymentClient

在文件末尾（`InitOrderClient` 函数之后）追加：

```go
paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"  // 添加到 import

func InitPaymentClient(etcdClient *clientv3.Client) paymentv1.PaymentServiceClient {
	conn := initServiceConn(etcdClient, "payment")
	return paymentv1.NewPaymentServiceClient(conn)
}
```

### 2.3 merchant-bff/ioc/gin.go — 添加 paymentHandler 参数 + 4 路由

函数签名添加 `paymentHandler *handler.PaymentHandler` 参数。

在 auth 路由组的订单/退款路由之后追加：

```go
// 支付管理
auth.GET("/payments", ginx.WrapQuery[handler.ListPaymentsReq](l, paymentHandler.ListPayments))
auth.GET("/payments/:paymentNo", paymentHandler.GetPayment)
auth.POST("/payments/:paymentNo/refund", ginx.WrapBody[handler.RefundReq](l, paymentHandler.Refund))
auth.GET("/refunds/:refundNo/payment", paymentHandler.GetRefund)
```

注意：merchant-bff 的 order handler 已有 `GET /refunds` 路由（用于订单退款列表），支付退款查询使用 `GET /refunds/:refundNo/payment` 避免冲突。

### 2.4 merchant-bff/wire.go — 添加 InitPaymentClient + NewPaymentHandler

```go
// thirdPartySet 添加:
ioc.InitPaymentClient,

// handlerSet 添加:
handler.NewPaymentHandler,
```

### 2.5 验证

```bash
go build ./merchant-bff/...
go vet ./merchant-bff/...
```

---

## 验证步骤

```bash
go build ./consumer-bff/... ./merchant-bff/...
go vet ./consumer-bff/... ./merchant-bff/...
```

---

## 文件清单

| # | 文件路径 | 操作 | 说明 |
|---|---------|------|------|
| 1 | `consumer-bff/handler/payment.go` | 新建 | PaymentHandler + CreatePayment, GetPayment, HandleNotify |
| 2 | `consumer-bff/ioc/grpc.go` | 修改 | +InitPaymentClient |
| 3 | `consumer-bff/ioc/gin.go` | 修改 | +paymentHandler 参数 + 3 路由 |
| 4 | `consumer-bff/wire.go` | 修改 | +InitPaymentClient + NewPaymentHandler |
| 5 | `consumer-bff/wire_gen.go` | 重新生成 | Wire DI |
| 6 | `merchant-bff/handler/payment.go` | 新建 | PaymentHandler + ListPayments, GetPayment, Refund, GetRefund |
| 7 | `merchant-bff/ioc/grpc.go` | 修改 | +InitPaymentClient |
| 8 | `merchant-bff/ioc/gin.go` | 修改 | +paymentHandler 参数 + 4 路由 |
| 9 | `merchant-bff/wire.go` | 修改 | +InitPaymentClient + NewPaymentHandler |
| 10 | `merchant-bff/wire_gen.go` | 重新生成 | Wire DI |

共 10 个文件（2 新建 + 6 修改 + 2 生成）。
