# BFF 订单接口设计

## 概述

在 merchant-bff 和 consumer-bff 中集成 order-svc 的订单接口。consumer-bff 提供完整的买家订单操作（下单、查单、取消、确认收货、退款申请），merchant-bff 提供商户订单管理（查单、发货、退款处理）。admin-bff 暂不接入。

## 接口分配

### consumer-bff（买家订单操作，8 路由）

| 方法 | 路由 | RPC | 说明 |
|------|------|-----|------|
| POST | `/api/v1/orders` | CreateOrder | 创建订单 |
| GET | `/api/v1/orders` | ListOrders | 买家订单列表 |
| GET | `/api/v1/orders/:orderNo` | GetOrder | 订单详情 |
| POST | `/api/v1/orders/:orderNo/cancel` | CancelOrder | 取消待支付订单 |
| POST | `/api/v1/orders/:orderNo/confirm` | ConfirmReceive | 确认收货 |
| POST | `/api/v1/orders/:orderNo/refund` | ApplyRefund | 申请退款 |
| GET | `/api/v1/refunds` | ListRefundOrders | 买家退款列表 |
| GET | `/api/v1/refunds/:refundNo` | GetRefundOrder | 退款详情 |

全部在 auth 路由组（JWT + 域名租户解析）。buyer_id 从 JWT uid 获取，tenant_id 从域名解析中间件注入。

### merchant-bff（商户订单管理，5 路由）

| 方法 | 路由 | RPC | 说明 |
|------|------|-----|------|
| GET | `/api/v1/orders` | ListOrders | 租户订单列表 |
| GET | `/api/v1/orders/:orderNo` | GetOrder | 订单详情 |
| POST | `/api/v1/orders/:orderNo/ship` | UpdateOrderStatus(shipped) | 商家发货 |
| POST | `/api/v1/orders/:orderNo/refund/handle` | HandleRefund | 审核退款 |
| GET | `/api/v1/refunds` | ListRefundOrders | 租户退款列表 |

全部在 auth 路由组（JWT + 租户中间件）。tenant_id 从 JWT claims 提取。

### 不暴露的 RPC

- UpdateOrderStatus（通用）— 仅商户发货动作通过 `/ship` 暴露，其余状态变更为系统内部调用
- ConfirmReceive — 仅消费者端暴露

## 文件变更清单

### consumer-bff

| 文件 | 操作 | 说明 |
|------|------|------|
| `consumer-bff/handler/order.go` | 新建 | OrderHandler：8 个方法 |
| `consumer-bff/ioc/grpc.go` | 修改 | 新增 InitOrderClient |
| `consumer-bff/ioc/gin.go` | 修改 | 注入 OrderHandler，注册 8 条路由 |
| `consumer-bff/wire.go` | 修改 | 加 InitOrderClient + NewOrderHandler |
| `consumer-bff/wire_gen.go` | 重新生成 | wire ./consumer-bff/ |

### merchant-bff

| 文件 | 操作 | 说明 |
|------|------|------|
| `merchant-bff/handler/order.go` | 新建 | OrderHandler：5 个方法 |
| `merchant-bff/ioc/grpc.go` | 修改 | 新增 InitOrderClient |
| `merchant-bff/ioc/gin.go` | 修改 | 注入 OrderHandler，注册 5 条路由 |
| `merchant-bff/wire.go` | 修改 | 加 InitOrderClient + NewOrderHandler |
| `merchant-bff/wire_gen.go` | 重新生成 | wire ./merchant-bff/ |

总计：2 个新文件 + 6 个修改文件 + 2 个重新生成。

## Request/Response 结构

### consumer-bff 请求体

```go
// CreateOrder — POST /api/v1/orders
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

// ListOrders — GET /api/v1/orders
type ListOrdersReq struct {
    Status   int32 `form:"status"`
    Page     int32 `form:"page" binding:"required,min=1"`
    PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

// GetOrder — GET /api/v1/orders/:orderNo
// orderNo 从 URL path 提取

// CancelOrder — POST /api/v1/orders/:orderNo/cancel
// orderNo 从 URL path 提取，无请求体

// ConfirmReceive — POST /api/v1/orders/:orderNo/confirm
// orderNo 从 URL path 提取，无请求体

// ApplyRefund — POST /api/v1/orders/:orderNo/refund
type ApplyRefundReq struct {
    Type         int32 `json:"type" binding:"required,oneof=1 2"`
    RefundAmount int64 `json:"refund_amount" binding:"required,min=1"`
    Reason       string `json:"reason" binding:"required"`
}

// ListRefundOrders — GET /api/v1/refunds
type ListRefundsReq struct {
    Status   int32 `form:"status"`
    Page     int32 `form:"page" binding:"required,min=1"`
    PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

// GetRefundOrder — GET /api/v1/refunds/:refundNo
// refundNo 从 URL path 提取
```

### merchant-bff 请求体

```go
// ListOrders — GET /api/v1/orders
type ListOrdersReq struct {
    Status   int32 `form:"status"`
    Page     int32 `form:"page" binding:"required,min=1"`
    PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

// GetOrder — GET /api/v1/orders/:orderNo
// orderNo 从 URL path 提取

// ShipOrder — POST /api/v1/orders/:orderNo/ship
// orderNo 从 URL path 提取，无请求体

// HandleRefund — POST /api/v1/orders/:orderNo/refund/handle
type HandleRefundReq struct {
    RefundNo string `json:"refund_no" binding:"required"`
    Approved bool   `json:"approved"`
    Reason   string `json:"reason"`
}

// ListRefundOrders — GET /api/v1/refunds
type ListRefundsReq struct {
    Status   int32 `form:"status"`
    Page     int32 `form:"page" binding:"required,min=1"`
    PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}
```

### 响应格式

统一 `ginx.Result{Code: 0, Msg: "success", Data: ...}`：

- CreateOrder → Data: {order_no, pay_amount}
- GetOrder → Data: order 对象（含 items）
- ListOrders → Data: {orders, total}
- CancelOrder → Data: nil
- ConfirmReceive → Data: nil
- ApplyRefund → Data: {refund_no}
- HandleRefund → Data: nil
- ShipOrder → Data: nil
- GetRefundOrder → Data: refund 对象
- ListRefundOrders → Data: {refund_orders, total}

### tenant_id / buyer_id 获取方式

- consumer-bff: `buyer_id` = `ctx.Get("uid")`, `tenant_id` = `ctx.Get("tenant_id")`（域名解析中间件）
- merchant-bff: `tenant_id` = `ctx.Get("tenant_id")`（JWT claims）
