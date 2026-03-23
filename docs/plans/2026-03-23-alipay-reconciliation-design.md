# Alipay Payment Channel & Reconciliation Design

**Date**: 2026-03-23
**Status**: Approved

## Overview

Two related features:
1. **Alipay H5 Payment**: Implement the existing `Channel` interface for Alipay H5 (WAP) payments using `github.com/smartwalle/alipay/v3` SDK
2. **Payment Reconciliation**: Daily automated reconciliation between local payment records and Alipay bill data, embedded in the payment service

## Part 1: Alipay H5 Payment

### SDK Choice

Use `github.com/smartwalle/alipay/v3` — the most mature Alipay Go wrapper based on official OpenAPI, supporting RSA2 signing, async callback verification, and bill download.

### Channel Interface Implementation

File: `payment/service/channel/alipay.go`

Implements all 5 methods of the existing `Channel` interface:

| Method | Alipay API | Description |
|--------|-----------|-------------|
| `Pay` | `alipay.trade.wap.pay` | H5 payment, returns payment page URL |
| `QueryPayment` | `alipay.trade.query` | Query payment status |
| `Refund` | `alipay.trade.refund` | Initiate refund (synchronous) |
| `QueryRefund` | `alipay.trade.fastpay.refund.query` | Query refund status |
| `VerifyNotify` | Callback signature verification | RSA2 verify async notification, extract out_trade_no and trade_no |

### Configuration

```yaml
# payment/config/dev.yaml
alipay:
  appId: "2021000..."
  privateKey: "MIIEvQ..."
  alipayPublicKey: "MIIBIj..."
  notifyUrl: "https://your-domain.com/api/v1/payment/notify/alipay"
  returnUrl: "https://your-domain.com/payment/result"
  isProd: false
```

### Payment Flow

```
User checkout → consumer-bff CreatePayment(channel="alipay")
    → payment-svc → AlipayChannel.Pay()
    → Call alipay.trade.wap.pay, get payment page URL
    → Return payUrl to frontend
    → Frontend window.location.href = payUrl (redirect to Alipay H5 cashier)

Payment complete → Alipay async callback to notifyUrl
    → consumer-bff HandleNotify(channel="alipay", body)
    → payment-svc → AlipayChannel.VerifyNotify() verify signature
    → Update PaymentOrder status to paid
    → Publish order_paid event
```

### Consumer BFF Callback Route

Add an **unauthenticated** route for Alipay async notifications:
```
POST /api/v1/payment/notify/alipay → read form body → call HandleNotify
```

### IoC Changes

- `payment/ioc/alipay.go`: Initialize `alipay.Client` from config, inject into `AlipayChannel`
- `payment/ioc/channel.go`: Build channel map `{"mock": MockChannel, "alipay": AlipayChannel}`
- Update `wire.go` to include alipay initialization

## Part 2: Payment Reconciliation

### Data Model

Two new tables in `mall_payment` database:

#### reconciliation_batches

| Column | Type | Description |
|--------|------|-------------|
| id | bigint PK | Auto-increment |
| batch_no | varchar(64) UNIQUE | Batch number |
| channel | varchar(32) | alipay / wechat |
| bill_date | varchar(10) | Bill date YYYY-MM-DD |
| status | int32 | 1=processing, 2=completed, 3=failed |
| total_channel | int32 | Channel record count |
| total_local | int32 | Local record count |
| total_match | int32 | Matched count |
| total_mismatch | int32 | Mismatch count |
| channel_amount | int64 | Channel total amount (fen) |
| local_amount | int64 | Local total amount (fen) |
| error_msg | varchar(512) | Error message if failed |
| ctime / utime | bigint | Timestamps |

#### reconciliation_details

| Column | Type | Description |
|--------|------|-------------|
| id | bigint PK | Auto-increment |
| batch_id | bigint | FK to batch |
| payment_no | varchar(64) | Local payment number |
| channel_trade_no | varchar(128) | Channel trade number |
| type | int32 | 1=local_extra, 2=channel_extra, 3=amount_mismatch, 4=status_mismatch |
| local_amount | int64 | Local amount (fen) |
| channel_amount | int64 | Channel amount (fen) |
| local_status | int32 | Local payment status |
| channel_status | varchar(32) | Channel status string |
| handled | bool | Whether resolved |
| remark | varchar(512) | Notes |
| ctime | bigint | Timestamp |

### Reconciliation Flow

```
Daily cron at 01:00 (reconcile previous day)
    │
    ├── 1. Download channel bill
    │      Call alipay.data.dataservice.bill.downloadurl.query
    │      Download CSV, parse into structured BillItems
    │
    ├── 2. Query local payment records
    │      SELECT * FROM payment_orders
    │      WHERE channel='alipay' AND pay_time within target date
    │
    ├── 3. Compare record by record
    │      Key: channel_trade_no
    │      - Local has, channel has → compare amount and status
    │      - Local has, channel missing → type=1 (local_extra)
    │      - Channel has, local missing → type=2 (channel_extra)
    │      - Amount mismatch → type=3
    │      - Status mismatch → type=4
    │
    ├── 4. Write results
    │      Create reconciliation_batch + reconciliation_details
    │
    └── 5. Alert (if mismatches > 0, log warning)
```

### Reconciler Interface

New optional interface (not all channels need reconciliation):

```go
type Reconciler interface {
    DownloadBill(ctx context.Context, billDate string) ([]BillItem, error)
}

type BillItem struct {
    ChannelTradeNo string
    OutTradeNo     string  // payment_no
    Amount         int64   // fen
    Status         string  // TRADE_SUCCESS / TRADE_CLOSED
    PayTime        string
}
```

Only AlipayChannel implements `Reconciler`. The reconciliation service checks via type assertion.

### New Proto RPCs

Add to `payment.proto`:

```protobuf
rpc RunReconciliation(RunReconciliationRequest) returns (RunReconciliationResponse);
rpc ListReconciliationBatches(ListBatchesRequest) returns (ListBatchesResponse);
rpc GetReconciliationDetail(GetDetailRequest) returns (GetDetailResponse);
```

### Admin BFF Endpoints

```
POST /api/v1/reconciliation/run           # Manual trigger (date + channel)
GET  /api/v1/reconciliation/batches       # Batch list (paginated)
GET  /api/v1/reconciliation/batches/:id   # Batch detail with mismatch items
```

### Reconciliation Service Location

Embedded in payment service (not a separate microservice), because:
- Data source is `payment_orders` table (same DB)
- Channel clients are already initialized in payment service
- No cross-service calls needed

New files:
- `payment/service/reconciliation.go` — reconciliation business logic
- `payment/service/reconciliation_job.go` — daily cron trigger
- `payment/repository/dao/reconciliation.go` — GORM models
- `payment/repository/reconciliation.go` — repository layer

## Summary of Changes

| Area | Changes |
|------|---------|
| Dependencies | Add `github.com/smartwalle/alipay/v3` |
| payment/service/channel/ | New `alipay.go`, extend `types.go` with Reconciler |
| payment/service/ | New `reconciliation.go`, `reconciliation_job.go` |
| payment/repository/ | New reconciliation DAO + repository |
| payment/ioc/ | New `alipay.go`, update channel initialization |
| payment/config/ | Add alipay config section |
| payment.proto | Add 3 reconciliation RPCs |
| payment/grpc/ | Add reconciliation RPC handlers |
| consumer-bff | Add `/api/v1/payment/notify/alipay` callback route |
| admin-bff | Add 3 reconciliation endpoints |
