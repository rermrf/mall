# Alipay Payment Channel & Reconciliation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement Alipay H5 payment channel using smartwalle/alipay SDK, replacing the stub implementation. Then add daily automated reconciliation that compares local payment records with Alipay bill data.

**Architecture:** Replace the existing `AlipayChannel` stub with a real implementation using `github.com/smartwalle/alipay/v3`. Add reconciliation as a new domain within the payment service (same DB, same process). Add admin-bff endpoints for manual reconciliation triggers and result viewing.

**Tech Stack:** Go 1.25, smartwalle/alipay/v3, GORM, Gin, gRPC, Kafka, time.Ticker for cron.

**Reference:** Design doc at `docs/plans/2026-03-23-alipay-reconciliation-design.md`. Current stub at `payment/service/channel/alipay.go`.

---

### Task 1: Add alipay SDK dependency and config

**Files:**
- Modify: `go.mod` (add dependency)
- Modify: `payment/config/dev.yaml` (add alipay config section)

**Step 1: Add SDK dependency**

Run: `go get github.com/smartwalle/alipay/v3`

**Step 2: Add alipay config to dev.yaml**

Append to `payment/config/dev.yaml`:
```yaml
alipay:
  appId: "2021000..."
  privateKey: "MIIEvQ..."
  alipayPublicKey: "MIIBIj..."
  notifyUrl: "https://your-domain.com/api/v1/payment/notify/alipay"
  returnUrl: "https://your-domain.com/payment/result"
  isProd: false
```

**Step 3: Create alipay IoC initializer**

Create `payment/ioc/alipay.go`:
```go
package ioc

import (
	"fmt"

	"github.com/smartwalle/alipay/v3"
	"github.com/spf13/viper"
)

func InitAlipayClient() *alipay.Client {
	type Config struct {
		AppId           string `yaml:"appId"`
		PrivateKey      string `yaml:"privateKey"`
		AlipayPublicKey string `yaml:"alipayPublicKey"`
		IsProd          bool   `yaml:"isProd"`
	}
	var cfg Config
	if err := viper.UnmarshalKey("alipay", &cfg); err != nil {
		panic(fmt.Errorf("读取支付宝配置失败: %w", err))
	}
	if cfg.AppId == "" {
		return nil // alipay not configured, skip
	}
	client, err := alipay.New(cfg.AppId, cfg.PrivateKey, cfg.IsProd)
	if err != nil {
		panic(fmt.Errorf("初始化支付宝客户端失败: %w", err))
	}
	if err = client.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
		panic(fmt.Errorf("加载支付宝公钥失败: %w", err))
	}
	return client
}
```

**Step 4: Commit**

```bash
git add go.mod go.sum payment/config/dev.yaml payment/ioc/alipay.go
git commit -m "feat(payment): add alipay SDK dependency and config"
```

---

### Task 2: Implement AlipayChannel — all 5 Channel interface methods

**Files:**
- Rewrite: `payment/service/channel/alipay.go`

**Step 1: Replace alipay.go stub with real implementation**

```go
package channel

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/smartwalle/alipay/v3"
	"github.com/spf13/viper"

	"github.com/rermrf/mall/payment/domain"
)

type AlipayChannel struct {
	client *alipay.Client
}

func NewAlipayChannel(client *alipay.Client) *AlipayChannel {
	return &AlipayChannel{client: client}
}

func (c *AlipayChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	if c.client == nil {
		return "", "", fmt.Errorf("支付宝客户端未初始化")
	}

	// 金额从分转为元（字符串，保留两位小数）
	amount := fmt.Sprintf("%.2f", float64(payment.Amount)/100.0)

	param := alipay.TradeWapPay{
		Trade: alipay.Trade{
			NotifyURL:   viper.GetString("alipay.notifyUrl"),
			ReturnURL:   viper.GetString("alipay.returnUrl"),
			Subject:     fmt.Sprintf("订单 %s", payment.OrderNo),
			OutTradeNo:  payment.PaymentNo,
			TotalAmount: amount,
			ProductCode: "QUICK_WAP_WAY",
		},
	}

	payURL, err := c.client.TradeWapPay(param)
	if err != nil {
		return "", "", fmt.Errorf("创建支付宝H5支付失败: %w", err)
	}

	return "", payURL.String(), nil
}

func (c *AlipayChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	if c.client == nil {
		return 0, "", fmt.Errorf("支付宝客户端未初始化")
	}

	param := alipay.TradeQuery{
		OutTradeNo: paymentNo,
	}

	var result alipay.TradeQueryRsp
	if err := c.client.Request(ctx, param, &result); err != nil {
		return 0, "", fmt.Errorf("查询支付宝支付状态失败: %w", err)
	}

	if result.Code != alipay.CodeSuccess {
		return 0, "", fmt.Errorf("支付宝查询失败: %s - %s", result.SubCode, result.SubMsg)
	}

	status := mapAlipayTradeStatus(result.TradeStatus)
	return int32(status), result.TradeNo, nil
}

func (c *AlipayChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("支付宝客户端未初始化")
	}

	amount := fmt.Sprintf("%.2f", float64(refund.Amount)/100.0)

	param := alipay.TradeRefund{
		OutTradeNo:   refund.PaymentNo,
		RefundAmount: amount,
		RefundReason: "用户退款",
		OutRequestNo: refund.RefundNo,
	}

	var result alipay.TradeRefundRsp
	if err := c.client.Request(ctx, param, &result); err != nil {
		return "", fmt.Errorf("支付宝退款失败: %w", err)
	}

	if result.Code != alipay.CodeSuccess {
		return "", fmt.Errorf("支付宝退款失败: %s - %s", result.SubCode, result.SubMsg)
	}

	return result.TradeNo, nil
}

func (c *AlipayChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	if c.client == nil {
		return 0, "", fmt.Errorf("支付宝客户端未初始化")
	}

	param := alipay.TradeFastpayRefundQuery{
		OutRequestNo: refundNo,
	}

	var result alipay.TradeFastpayRefundQueryRsp
	if err := c.client.Request(ctx, param, &result); err != nil {
		return 0, "", fmt.Errorf("查询支付宝退款状态失败: %w", err)
	}

	if result.Code != alipay.CodeSuccess {
		return 0, "", fmt.Errorf("支付宝退款查询失败: %s - %s", result.SubCode, result.SubMsg)
	}

	if result.RefundStatus == "REFUND_SUCCESS" {
		return int32(domain.RefundStatusRefunded), result.TradeNo, nil
	}
	return int32(domain.RefundStatusRefunding), result.TradeNo, nil
}

func (c *AlipayChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	if c.client == nil {
		return "", "", fmt.Errorf("支付宝客户端未初始化")
	}

	// Convert map to url.Values for SDK verification
	values := url.Values{}
	for k, v := range data {
		values.Set(k, v)
	}

	notification, err := c.client.DecodeNotification(values)
	if err != nil {
		return "", "", fmt.Errorf("支付宝回调验签失败: %w", err)
	}

	if notification.TradeStatus != "TRADE_SUCCESS" && notification.TradeStatus != "TRADE_FINISHED" {
		return "", "", fmt.Errorf("支付宝交易状态非成功: %s", notification.TradeStatus)
	}

	return notification.OutTradeNo, notification.TradeNo, nil
}

// DownloadBill implements Reconciler interface
func (c *AlipayChannel) DownloadBill(ctx context.Context, billDate string) ([]BillItem, error) {
	if c.client == nil {
		return nil, fmt.Errorf("支付宝客户端未初始化")
	}

	param := alipay.BillDownloadURLQuery{
		BillType: "trade",
		BillDate: billDate,
	}

	var result alipay.BillDownloadURLQueryRsp
	if err := c.client.Request(ctx, param, &result); err != nil {
		return nil, fmt.Errorf("查询支付宝对账单下载地址失败: %w", err)
	}

	if result.Code != alipay.CodeSuccess {
		return nil, fmt.Errorf("获取对账单失败: %s - %s", result.SubCode, result.SubMsg)
	}

	// Download and parse the CSV bill file
	return downloadAndParseBill(result.BillDownloadUrl)
}

// downloadAndParseBill downloads the zip file from Alipay, extracts CSV, and parses bill items
func downloadAndParseBill(downloadURL string) ([]BillItem, error) {
	// TODO: implement CSV download and parsing
	// 1. HTTP GET downloadURL → zip file
	// 2. Unzip → find the detail CSV (not summary)
	// 3. Parse each row into BillItem
	// For now, return empty - will be implemented in reconciliation task
	return nil, fmt.Errorf("对账单解析暂未实现")
}

func mapAlipayTradeStatus(status alipay.TradeStatus) domain.PaymentStatus {
	switch status {
	case alipay.TradeStatusWaitBuyerPay:
		return domain.PaymentStatusPending
	case alipay.TradeStatusClosed:
		return domain.PaymentStatusClosed
	case alipay.TradeStatusSuccess:
		return domain.PaymentStatusPaid
	case alipay.TradeStatusFinished:
		return domain.PaymentStatusPaid
	default:
		return domain.PaymentStatusPending
	}
}

// amountToFen converts yuan string (e.g. "99.00") to fen int64 (9900)
func amountToFen(yuan string) int64 {
	f, err := strconv.ParseFloat(yuan, 64)
	if err != nil {
		return 0
	}
	return int64(f * 100)
}
```

**Step 2: Add Reconciler interface and BillItem to types.go**

Append to `payment/service/channel/types.go`:
```go
// Reconciler is an optional interface for channels that support bill download
type Reconciler interface {
	DownloadBill(ctx context.Context, billDate string) ([]BillItem, error)
}

type BillItem struct {
	ChannelTradeNo string
	OutTradeNo     string // payment_no
	Amount         int64  // fen
	Status         string // TRADE_SUCCESS / TRADE_CLOSED etc.
	PayTime        string
}
```

**Step 3: Update NewPaymentService to accept AlipayChannel via DI**

Modify `payment/service/payment.go` constructor to accept `*alipay.Client` and construct `AlipayChannel` properly:
```go
func NewPaymentService(
	repo repository.PaymentRepository,
	producer events.Producer,
	idempotencySvc idempotent.IdempotencyService,
	node *snowflake.Node,
	mockCh *channel.MockChannel,
	alipayCh *channel.AlipayChannel,
	l logger.Logger,
) PaymentService {
	channels := map[string]channel.Channel{
		"mock":   mockCh,
		"alipay": alipayCh,
	}
	// ...
}
```

Update `payment/wire.go` to include `ioc.InitAlipayClient` and `channel.NewAlipayChannel`.

**Step 4: Commit**

```bash
git add payment/service/channel/ payment/service/payment.go payment/wire.go payment/wire_gen.go payment/ioc/alipay.go
git commit -m "feat(payment): implement Alipay H5 payment channel with smartwalle/alipay SDK"
```

---

### Task 3: Consumer BFF — Alipay notify callback route

**Files:**
- Modify: `consumer-bff/handler/payment.go` (add notify handler)
- Modify: `consumer-bff/ioc/gin.go` (add notify route)

**Step 1: Add Alipay notify handler**

The handler reads form data from Alipay's POST callback, converts it to the map format expected by `HandleNotify`, and responds with "success" on success.

Add to `consumer-bff/handler/payment.go`:
```go
func (h *PaymentHandler) AlipayNotify(ctx *gin.Context) {
	if err := ctx.Request.ParseForm(); err != nil {
		ctx.String(200, "FAIL")
		return
	}

	data := make(map[string]string)
	for k, v := range ctx.Request.Form {
		if len(v) > 0 {
			data[k] = v[0]
		}
	}
	bodyBytes, _ := json.Marshal(data)

	_, err := h.paymentClient.HandleNotify(ctx.Request.Context(), &paymentv1.HandleNotifyRequest{
		Channel:    "alipay",
		NotifyBody: string(bodyBytes),
	})
	if err != nil {
		ctx.String(200, "FAIL")
		return
	}
	ctx.String(200, "success")
}
```

**Step 2: Add route (no auth required)**

In `consumer-bff/ioc/gin.go`, add BEFORE the auth middleware group:
```go
server.POST("/api/v1/payment/notify/alipay", paymentHandler.AlipayNotify)
```

**Step 3: Commit**

```bash
git add consumer-bff/handler/payment.go consumer-bff/ioc/gin.go
git commit -m "feat(consumer-bff): add Alipay async notification callback endpoint"
```

---

### Task 4: Reconciliation — DAO models and repository

**Files:**
- Create: `payment/repository/dao/reconciliation.go`
- Create: `payment/repository/reconciliation.go`

**Step 1: Create reconciliation DAO models**

`payment/repository/dao/reconciliation.go`:
```go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ReconciliationBatchModel struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	BatchNo        string `gorm:"type:varchar(64);uniqueIndex:uk_batch_no"`
	Channel        string `gorm:"type:varchar(32)"`
	BillDate       string `gorm:"type:varchar(10);index:idx_bill_date"`
	Status         int32  // 1=处理中 2=已完成 3=失败
	TotalChannel   int32
	TotalLocal     int32
	TotalMatch     int32
	TotalMismatch  int32
	ChannelAmount  int64
	LocalAmount    int64
	ErrorMsg       string `gorm:"type:varchar(512)"`
	Ctime          int64
	Utime          int64
}

func (ReconciliationBatchModel) TableName() string { return "reconciliation_batches" }

type ReconciliationDetailModel struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	BatchId        int64  `gorm:"index:idx_batch_id"`
	PaymentNo      string `gorm:"type:varchar(64)"`
	ChannelTradeNo string `gorm:"type:varchar(128)"`
	Type           int32  // 1=本地多 2=渠道多 3=金额不一致 4=状态不一致
	LocalAmount    int64
	ChannelAmount  int64
	LocalStatus    int32
	ChannelStatus  string `gorm:"type:varchar(32)"`
	Handled        bool
	Remark         string `gorm:"type:varchar(512)"`
	Ctime          int64
}

func (ReconciliationDetailModel) TableName() string { return "reconciliation_details" }

type ReconciliationDAO interface {
	CreateBatch(ctx context.Context, batch ReconciliationBatchModel) (ReconciliationBatchModel, error)
	UpdateBatch(ctx context.Context, id int64, updates map[string]any) error
	ListBatches(ctx context.Context, offset, limit int) ([]ReconciliationBatchModel, int64, error)
	GetBatch(ctx context.Context, id int64) (ReconciliationBatchModel, error)
	CreateDetails(ctx context.Context, details []ReconciliationDetailModel) error
	ListDetails(ctx context.Context, batchId int64, offset, limit int) ([]ReconciliationDetailModel, int64, error)
}

type GORMReconciliationDAO struct {
	db *gorm.DB
}

func NewReconciliationDAO(db *gorm.DB) ReconciliationDAO {
	return &GORMReconciliationDAO{db: db}
}

func (d *GORMReconciliationDAO) CreateBatch(ctx context.Context, batch ReconciliationBatchModel) (ReconciliationBatchModel, error) {
	now := time.Now().UnixMilli()
	batch.Ctime = now
	batch.Utime = now
	err := d.db.WithContext(ctx).Create(&batch).Error
	return batch, err
}

func (d *GORMReconciliationDAO) UpdateBatch(ctx context.Context, id int64, updates map[string]any) error {
	updates["utime"] = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Model(&ReconciliationBatchModel{}).Where("id = ?", id).Updates(updates).Error
}

func (d *GORMReconciliationDAO) ListBatches(ctx context.Context, offset, limit int) ([]ReconciliationBatchModel, int64, error) {
	var batches []ReconciliationBatchModel
	var total int64
	query := d.db.WithContext(ctx).Model(&ReconciliationBatchModel{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&batches).Error
	return batches, total, err
}

func (d *GORMReconciliationDAO) GetBatch(ctx context.Context, id int64) (ReconciliationBatchModel, error) {
	var batch ReconciliationBatchModel
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&batch).Error
	return batch, err
}

func (d *GORMReconciliationDAO) CreateDetails(ctx context.Context, details []ReconciliationDetailModel) error {
	if len(details) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	for i := range details {
		details[i].Ctime = now
	}
	return d.db.WithContext(ctx).CreateInBatches(details, 100).Error
}

func (d *GORMReconciliationDAO) ListDetails(ctx context.Context, batchId int64, offset, limit int) ([]ReconciliationDetailModel, int64, error) {
	var details []ReconciliationDetailModel
	var total int64
	query := d.db.WithContext(ctx).Model(&ReconciliationDetailModel{}).Where("batch_id = ?", batchId)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id ASC").Offset(offset).Limit(limit).Find(&details).Error
	return details, total, err
}
```

Update `InitTables` in `payment/repository/dao/payment.go` to add the two new models.

**Step 2: Commit**

```bash
git add payment/repository/dao/reconciliation.go payment/repository/dao/payment.go
git commit -m "feat(payment): add reconciliation DAO models and tables"
```

---

### Task 5: Reconciliation — service layer with bill parsing and comparison

**Files:**
- Create: `payment/service/reconciliation.go`
- Create: `payment/service/reconciliation_job.go`

**Step 1: Create reconciliation service**

`payment/service/reconciliation.go` — implements:
1. `RunReconciliation(channel, billDate)` — orchestrates the full flow
2. Downloads bill via `Reconciler` interface
3. Queries local payment records for the same date
4. Compares records by channel_trade_no
5. Creates batch + detail records
6. `ListBatches(page, pageSize)` and `GetBatchDetail(batchId, page, pageSize)` for admin queries

Complete the `downloadAndParseBill` function in `alipay.go` to:
1. HTTP GET the download URL → get zip file
2. Unzip → find the detail CSV file (filename contains "业务明细")
3. Parse CSV rows (skip header/summary rows) into `[]BillItem`

**Step 2: Create reconciliation cron job**

`payment/service/reconciliation_job.go` — daily ticker at 01:00, reconciles previous day for all configured channels.

**Step 3: Commit**

```bash
git add payment/service/reconciliation.go payment/service/reconciliation_job.go payment/service/channel/alipay.go
git commit -m "feat(payment): add reconciliation service with bill download and comparison"
```

---

### Task 6: Proto extension and gRPC handlers for reconciliation

**Files:**
- Modify: `api/proto/payment/v1/payment.proto` (add 3 RPCs)
- Modify: `payment/grpc/payment.go` (add RPC implementations)

**Step 1: Add reconciliation RPCs to payment.proto**

```protobuf
// Reconciliation
rpc RunReconciliation(RunReconciliationRequest) returns (RunReconciliationResponse);
rpc ListReconciliationBatches(ListReconciliationBatchesRequest) returns (ListReconciliationBatchesResponse);
rpc GetReconciliationBatchDetail(GetReconciliationBatchDetailRequest) returns (GetReconciliationBatchDetailResponse);

// Messages
message RunReconciliationRequest {
  string channel = 1;
  string bill_date = 2;
}
message RunReconciliationResponse {
  int64 batch_id = 1;
}

message ReconciliationBatch {
  int64 id = 1;
  string batch_no = 2;
  string channel = 3;
  string bill_date = 4;
  int32 status = 5;
  int32 total_channel = 6;
  int32 total_local = 7;
  int32 total_match = 8;
  int32 total_mismatch = 9;
  int64 channel_amount = 10;
  int64 local_amount = 11;
  string error_msg = 12;
  google.protobuf.Timestamp ctime = 13;
}

message ReconciliationDetail {
  int64 id = 1;
  int64 batch_id = 2;
  string payment_no = 3;
  string channel_trade_no = 4;
  int32 type = 5;
  int64 local_amount = 6;
  int64 channel_amount = 7;
  int32 local_status = 8;
  string channel_status = 9;
  bool handled = 10;
  string remark = 11;
}

message ListReconciliationBatchesRequest {
  int32 page = 1;
  int32 page_size = 2;
}
message ListReconciliationBatchesResponse {
  repeated ReconciliationBatch batches = 1;
  int64 total = 2;
}

message GetReconciliationBatchDetailRequest {
  int64 batch_id = 1;
  int32 page = 2;
  int32 page_size = 3;
}
message GetReconciliationBatchDetailResponse {
  ReconciliationBatch batch = 1;
  repeated ReconciliationDetail details = 2;
  int64 total = 3;
}
```

**Step 2: Run proto generation**

Run: `make grpc`

**Step 3: Add gRPC handlers**

**Step 4: Commit**

```bash
git add api/proto/payment/ api/proto/gen/payment/ payment/grpc/
git commit -m "feat(payment): add reconciliation gRPC RPCs and handlers"
```

---

### Task 7: Admin BFF — reconciliation endpoints

**Files:**
- Create or modify: `admin-bff/handler/reconciliation.go`
- Modify: `admin-bff/ioc/gin.go`

**Step 1: Create reconciliation handler**

```
POST /api/v1/reconciliation/run           → { channel, bill_date }
GET  /api/v1/reconciliation/batches       → { page, page_size }
GET  /api/v1/reconciliation/batches/:id   → { page, page_size } (detail with mismatch items)
```

**Step 2: Add routes**

**Step 3: Commit**

```bash
git add admin-bff/handler/reconciliation.go admin-bff/ioc/gin.go admin-bff/wire.go admin-bff/wire_gen.go
git commit -m "feat(admin-bff): add reconciliation management endpoints"
```

---

### Task 8: Wire updates and build verification

**Files:**
- Modify: `payment/wire.go` (add reconciliation providers)
- Modify: `payment/ioc/` (update DB init for new tables)

**Step 1: Update wire.go**

Add `ReconciliationDAO`, `ReconciliationService`, `InitAlipayClient` to provider sets.

**Step 2: Regenerate wire**

Run: `cd payment && wire`

**Step 3: Build all affected services**

```bash
go build ./payment/...
go build ./consumer-bff/...
go build ./admin-bff/...
```

**Step 4: Commit**

```bash
git add payment/ consumer-bff/ admin-bff/
git commit -m "feat(payment): complete Alipay channel and reconciliation integration"
```

---

## Summary

| Task | Description | Scope |
|------|-------------|-------|
| 1 | SDK dependency + config + IoC | go.mod, config, ioc/alipay.go |
| 2 | AlipayChannel implementation | 5 Channel methods + Reconciler + DownloadBill |
| 3 | Consumer BFF notify callback | POST /api/v1/payment/notify/alipay |
| 4 | Reconciliation DAO | 2 new tables (batches + details) |
| 5 | Reconciliation service | Bill download, CSV parse, comparison, cron job |
| 6 | Proto + gRPC for reconciliation | 3 new RPCs |
| 7 | Admin BFF reconciliation endpoints | 3 new admin endpoints |
| 8 | Wire + build verification | Final integration and compilation |
