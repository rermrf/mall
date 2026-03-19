# Merchant Account Module Design

**Date**: 2026-03-19
**Status**: Approved

## Overview

Add a new `account` microservice to the mall platform, providing merchant account management, order settlement with platform commission, withdrawal management, and transaction ledger/reconciliation. This fills the critical gap in the current payment system where money "disappears" after user payment — there is no concept of merchant funds, settlement, or payout.

## Problem Statement

Current payment service is a "pass-through" system:
- User pays → PaymentOrder recorded → order_paid event → **nothing happens to merchant funds**
- Refund initiated → channel refund called → **no tracking of fund movement**
- No merchant balance, no settlement, no withdrawal, no reconciliation

## Architecture Decision

**Approach**: Independent `account` microservice (port 8087) with dedicated database `mall_account`, integrated via Kafka events.

**Rationale**: Account/settlement is a distinct bounded context from payment processing. Separating it follows the existing DDD microservice pattern (11 services already), enables independent scaling, and provides proper fund isolation. Event-driven integration with order/payment avoids tight coupling.

## Data Model

### Database: `mall_account`

#### merchant_accounts (商家账户)

| Column | Type | Description |
|--------|------|-------------|
| id | bigint PK | Auto-increment |
| tenant_id | bigint UNIQUE | One account per tenant |
| balance | bigint | Available balance (fen) |
| frozen_balance | bigint | Frozen amount (in withdrawal) |
| total_income | bigint | Cumulative income |
| total_withdrawn | bigint | Cumulative withdrawn |
| total_commission | bigint | Cumulative commission deducted |
| bank_account_name | varchar(128) | Bank account holder name |
| bank_account_no | varchar(64) | Bank card number |
| bank_name | varchar(128) | Bank name |
| status | int32 | 1=active, 2=frozen |
| ctime / utime | bigint | Timestamps |

#### settlement_records (结算记录)

| Column | Type | Description |
|--------|------|-------------|
| id | bigint PK | Auto-increment |
| tenant_id | bigint | Tenant reference |
| settlement_no | varchar(64) UNIQUE | Settlement number |
| order_no | varchar(64) | Associated order |
| payment_no | varchar(64) | Associated payment |
| order_amount | bigint | Order amount (fen) |
| commission_rate | int32 | Rate in basis points (500 = 5%) |
| commission_amount | bigint | Commission amount |
| settlement_amount | bigint | Net settlement = order_amount - commission |
| status | int32 | 1=pending, 2=settled, 3=refund_reversed |
| settled_at | bigint | When settled |
| settle_date | varchar(10) | T+N target date (YYYY-MM-DD) |
| ctime / utime | bigint | Timestamps |

#### withdrawal_records (提现记录)

| Column | Type | Description |
|--------|------|-------------|
| id | bigint PK | Auto-increment |
| tenant_id | bigint | Tenant reference |
| withdrawal_no | varchar(64) UNIQUE | Withdrawal number |
| amount | bigint | Withdrawal amount (fen) |
| status | int32 | 1=pending_review, 2=approved, 3=rejected, 4=paid |
| bank_account_name | varchar(128) | Target bank account name |
| bank_account_no | varchar(64) | Target bank card |
| bank_name | varchar(128) | Target bank |
| reject_reason | varchar(512) | Reason if rejected |
| reviewed_at | bigint | Review timestamp |
| paid_at | bigint | Payment timestamp |
| ctime / utime | bigint | Timestamps |

#### account_transactions (交易流水)

| Column | Type | Description |
|--------|------|-------------|
| id | bigint PK | Auto-increment |
| tenant_id | bigint | Tenant reference |
| transaction_no | varchar(64) UNIQUE | Transaction number |
| type | int32 | 1=settlement_credit, 2=withdrawal_freeze, 3=withdrawal_debit, 4=withdrawal_return, 5=refund_debit |
| amount | bigint | Change amount (positive=credit, negative=debit) |
| balance_after | bigint | Balance after this transaction |
| ref_no | varchar(64) | Reference number (settlement/withdrawal) |
| remark | varchar(256) | Description |
| ctime | bigint | Timestamp |

## gRPC API

### Proto: `api/proto/account/v1/account.proto`

```protobuf
service AccountService {
  // Account management
  rpc GetAccount(GetAccountRequest) returns (GetAccountResponse);
  rpc CreateAccount(CreateAccountRequest) returns (CreateAccountResponse);
  rpc UpdateBankInfo(UpdateBankInfoRequest) returns (UpdateBankInfoResponse);

  // Settlement
  rpc CreateSettlement(CreateSettlementRequest) returns (CreateSettlementResponse);
  rpc ListSettlements(ListSettlementsRequest) returns (ListSettlementsResponse);
  rpc ExecuteSettlement(ExecuteSettlementRequest) returns (ExecuteSettlementResponse);

  // Withdrawal
  rpc RequestWithdrawal(RequestWithdrawalRequest) returns (RequestWithdrawalResponse);
  rpc ReviewWithdrawal(ReviewWithdrawalRequest) returns (ReviewWithdrawalResponse);
  rpc ConfirmWithdrawalPaid(ConfirmWithdrawalPaidRequest) returns (ConfirmWithdrawalPaidResponse);
  rpc ListWithdrawals(ListWithdrawalsRequest) returns (ListWithdrawalsResponse);

  // Transactions & Summary
  rpc ListTransactions(ListTransactionsRequest) returns (ListTransactionsResponse);
  rpc GetAccountSummary(GetAccountSummaryRequest) returns (GetAccountSummaryResponse);
}
```

## Event-Driven Data Flow

### Incoming Events (consumed by account service)

**Topic: `order_completed`**
```json
{
  "orderNo": "string",
  "tenantId": 123,
  "paymentNo": "string",
  "amount": 10000,
  "completedAt": 1710000000000
}
```

Triggered when order reaches "completed" status. Account service creates a settlement_record with status=pending and settle_date = completedAt + N days.

**Topic: `refund_completed`**
```json
{
  "orderNo": "string",
  "tenantId": 123,
  "refundNo": "string",
  "paymentNo": "string",
  "amount": 5000,
  "refundedAt": 1710000000000
}
```

Account service handles:
- If settlement already settled: creates refund debit transaction, deducts from balance
- If settlement still pending: updates settlement status to refund_reversed

### Settlement Flow

```
Order Completed → order_completed event
    → account: create settlement_record (pending, settle_date=T+N)

Daily Cron (00:30) → ExecuteSettlement
    → query settle_date <= today AND status=pending
    → for each record:
        → fetch commission_rate from tenant service (via plan)
        → calculate commission_amount and settlement_amount
        → credit merchant balance
        → record account_transaction (type=settlement_credit)
        → update settlement_record status=settled
```

### Withdrawal Flow

```
Merchant requests withdrawal
    → check balance >= amount
    → freeze: balance -= amount, frozen_balance += amount
    → create withdrawal_record (pending_review)
    → record transaction (type=withdrawal_freeze)

Admin reviews:
    → Approve: status=approved (wait for manual bank transfer)
    → Reject: status=rejected, unfreeze: balance += amount, frozen_balance -= amount
              record transaction (type=withdrawal_return)

Admin confirms paid:
    → status=paid, frozen_balance -= amount, total_withdrawn += amount
    → record transaction (type=withdrawal_debit)
```

## Commission Configuration

Commission rate is stored in the tenant service's subscription plan. Account service calls tenant service via gRPC to get the current commission rate when creating settlement records.

Default commission rate configurable in account service config (e.g., `default_commission_rate: 500` for 5%).

## Service Structure

```
account/
├── config/
│   └── dev.yaml
├── domain/
│   └── account.go
├── repository/
│   ├── dao/account.go
│   ├── cache/account.go
│   └── account.go
├── service/
│   ├── account.go
│   └── settlement_job.go
├── grpc/
│   └── account.go
├── events/
│   ├── consumer.go
│   └── types.go
├── integration/
│   └── tenant.go
├── ioc/
├── main.go
└── wire.go
```

## BFF Integration

### merchant-bff (new endpoints)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/account | Get my account (balance, bank info) |
| PUT | /api/v1/account/bank-info | Update bank account info |
| GET | /api/v1/settlements | List my settlement records |
| POST | /api/v1/withdrawals | Request withdrawal |
| GET | /api/v1/withdrawals | List my withdrawal records |
| GET | /api/v1/transactions | List my transaction ledger |
| GET | /api/v1/account/summary | Account summary (monthly income, pending, withdrawn) |

### admin-bff (new endpoints)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/accounts | List all merchant accounts |
| GET | /api/v1/accounts/:tenantId | Get specific merchant account |
| GET | /api/v1/settlements | List all settlements (filter by tenant/date) |
| GET | /api/v1/withdrawals | List all withdrawal requests |
| POST | /api/v1/withdrawals/:id/review | Review withdrawal (approve/reject) |
| POST | /api/v1/withdrawals/:id/confirm | Confirm withdrawal paid |
| GET | /api/v1/transactions | List all platform transactions |

## Frontend Changes

### merchant-frontend (new pages)

- **Account Overview**: Balance card, bank info, quick actions (withdraw, view settlements)
- **Settlement List**: ProTable with date filter, showing order→commission→net amount
- **Withdrawal Management**: Withdrawal form + history list with status tracking
- **Transaction Ledger**: ProTable with type/date filters, running balance display

### admin-frontend (new pages)

- **Merchant Accounts**: ProTable listing all merchant accounts with balances
- **Withdrawal Review**: ProTable with pending review filter, approve/reject/confirm actions
- **Platform Settlements**: ProTable with tenant/date filters for settlement monitoring
- **Platform Transactions**: ProTable with tenant/type filters for transaction monitoring

## Order Service Changes

Order service needs to publish `order_completed` event when order status transitions to "completed" (after buyer confirms receipt or auto-confirm timeout). Currently it only publishes `order_paid`.

## Key Design Decisions

1. **T+N settlement** (configurable, default T+7): Protects against refunds after settlement
2. **Commission in basis points** (万分比): Enables fine-grained rate control (e.g., 350 = 3.5%)
3. **Manual withdrawal + admin review**: Safe for initial launch, automated payout can be added later
4. **Balance + frozen_balance**: Proper fund isolation during withdrawal processing
5. **Transaction ledger**: Complete audit trail, every balance change has a transaction record
6. **Idempotency**: settlement_no and withdrawal_no as unique keys prevent duplicate operations
