# WeChat Native Payment Channel & Reconciliation Design

**Date**: 2026-03-23
**Status**: Approved

## Overview

Implement WeChat Native (scan-to-pay) payment channel using the official `github.com/wechatpay-apiv3/wechatpay-go` SDK. Extends the existing `Channel` and `Reconciler` interfaces — same architecture pattern as Alipay integration.

## SDK

Official WeChat Pay Go SDK: `github.com/wechatpay-apiv3/wechatpay-go`. Supports V3 API, AEAD-AES-256-GCM callback decryption, certificate auto-update, and bill download.

## Channel Interface Implementation

File: `payment/service/channel/wechat.go`

| Method | WeChat V3 API | Description |
|--------|--------------|-------------|
| `Pay` | `POST /v3/pay/transactions/native` | Native order, returns `code_url` for QR code |
| `QueryPayment` | `GET /v3/pay/transactions/out-trade-no/{no}` | Query payment status |
| `Refund` | `POST /v3/refund/domestic/refunds` | Initiate refund |
| `QueryRefund` | `GET /v3/refund/domestic/refunds/{no}` | Query refund status |
| `VerifyNotify` | V3 callback signature verify + AES decrypt | Parse async notification |
| `DownloadBill` | `GET /v3/bill/tradebill` | Download trade bill CSV |

## Configuration

```yaml
wechat:
  appId: "wx..."
  mchId: "1900..."
  mchApiV3Key: "..."
  privateKeyPath: "config/wechat_apiclient_key.pem"
  serialNo: "..."
  notifyUrl: "https://your-domain.com/api/v1/payment/notify/wechat"
```

## Payment Flow

```
User checkout → CreatePayment(channel="wechat")
    → WechatChannel.Pay() → POST /v3/pay/transactions/native
    → Returns code_url (QR code URL)
    → Frontend generates QR code for user to scan

Payment complete → WeChat async callback to notifyUrl
    → consumer-bff POST /api/v1/payment/notify/wechat
    → SDK verify signature + AES-256-GCM decrypt
    → Extract out_trade_no (payment_no) + transaction_id
    → Update PaymentOrder status to paid
    → Publish order_paid event
```

## Changes

| Area | Change |
|------|--------|
| `payment/service/channel/wechat.go` | New: Channel + Reconciler implementation |
| `payment/ioc/wechat.go` | New: WeChat client initialization |
| `payment/config/dev.yaml` | Add wechat config section |
| `payment/service/payment.go` | Add wechatCh parameter to constructor |
| `payment/service/reconciliation.go` | Add wechatCh parameter to constructor |
| `consumer-bff/handler/payment.go` | Add WechatNotify handler |
| `consumer-bff/ioc/gin.go` | Add /api/v1/payment/notify/wechat route |
| `payment/wire.go` | Add wechat providers |
