# Merchant Account Module Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a new `account` microservice providing merchant settlement (T+N with commission), withdrawal with admin review, and transaction ledger — filling the critical gap where merchant funds are invisible after payment.

**Architecture:** Independent Go microservice (`account/`, port 8087, DB `mall_account`) following existing DDD patterns. Event-driven integration via Kafka (`order_completed` → settlement creation). Cron-based T+N batch settlement. BFF endpoints for merchant and admin operations.

**Tech Stack:** Go 1.25, Gin (BFF only), gRPC + Protocol Buffers, GORM + MySQL, Redis, Kafka (Sarama), Google Wire, etcd service discovery, Snowflake IDs.

**Reference patterns:** All code follows `/payment/` service structure exactly. See design doc at `docs/plans/2026-03-19-merchant-account-design.md`.

---

### Task 1: Proto definition and code generation

**Files:**
- Create: `api/proto/account/v1/account.proto`
- Modify: `api/proto/tenant/v1/tenant.proto` (add commission_rate to TenantPlan)
- Modify: `order/events/types.go` (add PaymentNo, Amount to OrderCompletedEvent)

**Step 1: Create account.proto**

```protobuf
syntax = "proto3";

package account.v1;

import "google/protobuf/timestamp.proto";

option go_package = "/account/v1;accountv1";

// ==================== Messages ====================

message MerchantAccount {
  int64 id = 1;
  int64 tenant_id = 2;
  int64 balance = 3;           // 可用余额（分）
  int64 frozen_balance = 4;    // 冻结金额（提现中）
  int64 total_income = 5;      // 累计收入
  int64 total_withdrawn = 6;   // 累计提现
  int64 total_commission = 7;  // 累计被扣佣金
  string bank_account_name = 8;
  string bank_account_no = 9;
  string bank_name = 10;
  int32 status = 11; // 1=正常 2=冻结
  google.protobuf.Timestamp ctime = 12;
  google.protobuf.Timestamp utime = 13;
}

message SettlementRecord {
  int64 id = 1;
  int64 tenant_id = 2;
  string settlement_no = 3;
  string order_no = 4;
  string payment_no = 5;
  int64 order_amount = 6;     // 订单金额（分）
  int32 commission_rate = 7;  // 佣金比例（万分比，500=5%）
  int64 commission_amount = 8;
  int64 settlement_amount = 9;
  int32 status = 10; // 1=待结算 2=已结算 3=已退款冲销
  int64 settled_at = 11;
  string settle_date = 12;    // YYYY-MM-DD
  google.protobuf.Timestamp ctime = 13;
  google.protobuf.Timestamp utime = 14;
}

message WithdrawalRecord {
  int64 id = 1;
  int64 tenant_id = 2;
  string withdrawal_no = 3;
  int64 amount = 4;
  int32 status = 5; // 1=待审核 2=已通过 3=已拒绝 4=已打款
  string bank_account_name = 6;
  string bank_account_no = 7;
  string bank_name = 8;
  string reject_reason = 9;
  int64 reviewed_at = 10;
  int64 paid_at = 11;
  google.protobuf.Timestamp ctime = 12;
  google.protobuf.Timestamp utime = 13;
}

message AccountTransaction {
  int64 id = 1;
  int64 tenant_id = 2;
  string transaction_no = 3;
  int32 type = 4; // 1=结算入账 2=提现冻结 3=提现扣款 4=提现退回 5=退款扣款
  int64 amount = 5;         // 变动金额（正=入账，负=扣款）
  int64 balance_after = 6;
  string ref_no = 7;        // 关联单号
  string remark = 8;
  google.protobuf.Timestamp ctime = 9;
}

message AccountSummary {
  int64 balance = 1;
  int64 frozen_balance = 2;
  int64 pending_settlement = 3;
  int64 month_income = 4;
  int64 month_commission = 5;
  int64 month_withdrawn = 6;
}

// ==================== AccountService ====================

service AccountService {
  // 账户管理
  rpc GetAccount(GetAccountRequest) returns (GetAccountResponse);
  rpc CreateAccount(CreateAccountRequest) returns (CreateAccountResponse);
  rpc UpdateBankInfo(UpdateBankInfoRequest) returns (UpdateBankInfoResponse);

  // 结算
  rpc CreateSettlement(CreateSettlementRequest) returns (CreateSettlementResponse);
  rpc ListSettlements(ListSettlementsRequest) returns (ListSettlementsResponse);
  rpc ExecuteSettlement(ExecuteSettlementRequest) returns (ExecuteSettlementResponse);

  // 提现
  rpc RequestWithdrawal(RequestWithdrawalRequest) returns (RequestWithdrawalResponse);
  rpc ReviewWithdrawal(ReviewWithdrawalRequest) returns (ReviewWithdrawalResponse);
  rpc ConfirmWithdrawalPaid(ConfirmWithdrawalPaidRequest) returns (ConfirmWithdrawalPaidResponse);
  rpc ListWithdrawals(ListWithdrawalsRequest) returns (ListWithdrawalsResponse);

  // 流水 & 对账
  rpc ListTransactions(ListTransactionsRequest) returns (ListTransactionsResponse);
  rpc GetAccountSummary(GetAccountSummaryRequest) returns (GetAccountSummaryResponse);

  // 管理端
  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse);
}

// ==================== Requests & Responses ====================

message GetAccountRequest { int64 tenant_id = 1; }
message GetAccountResponse { MerchantAccount account = 1; }

message CreateAccountRequest { int64 tenant_id = 1; }
message CreateAccountResponse { int64 id = 1; }

message UpdateBankInfoRequest {
  int64 tenant_id = 1;
  string bank_account_name = 2;
  string bank_account_no = 3;
  string bank_name = 4;
}
message UpdateBankInfoResponse {}

message CreateSettlementRequest {
  int64 tenant_id = 1;
  string order_no = 2;
  string payment_no = 3;
  int64 order_amount = 4;
}
message CreateSettlementResponse { string settlement_no = 1; }

message ListSettlementsRequest {
  int64 tenant_id = 1;
  int32 status = 2;
  int32 page = 3;
  int32 page_size = 4;
}
message ListSettlementsResponse {
  repeated SettlementRecord settlements = 1;
  int64 total = 2;
}

message ExecuteSettlementRequest {
  string settle_date = 1; // YYYY-MM-DD, 结算所有 <= 此日期的待结算记录
}
message ExecuteSettlementResponse {
  int32 settled_count = 1;
  int64 settled_amount = 2;
}

message RequestWithdrawalRequest {
  int64 tenant_id = 1;
  int64 amount = 2;
}
message RequestWithdrawalResponse { string withdrawal_no = 1; }

message ReviewWithdrawalRequest {
  int64 id = 1;
  bool approved = 2;
  string reject_reason = 3;
}
message ReviewWithdrawalResponse {}

message ConfirmWithdrawalPaidRequest { int64 id = 1; }
message ConfirmWithdrawalPaidResponse {}

message ListWithdrawalsRequest {
  int64 tenant_id = 1;
  int32 status = 2;
  int32 page = 3;
  int32 page_size = 4;
}
message ListWithdrawalsResponse {
  repeated WithdrawalRecord withdrawals = 1;
  int64 total = 2;
}

message ListTransactionsRequest {
  int64 tenant_id = 1;
  int32 type = 2;
  int32 page = 3;
  int32 page_size = 4;
}
message ListTransactionsResponse {
  repeated AccountTransaction transactions = 1;
  int64 total = 2;
}

message GetAccountSummaryRequest { int64 tenant_id = 1; }
message GetAccountSummaryResponse { AccountSummary summary = 1; }

message ListAccountsRequest {
  int32 page = 1;
  int32 page_size = 2;
}
message ListAccountsResponse {
  repeated MerchantAccount accounts = 1;
  int64 total = 2;
}
```

**Step 2: Add commission_rate to TenantPlan in tenant.proto**

In `api/proto/tenant/v1/tenant.proto`, add field 11 to `TenantPlan`:
```protobuf
message TenantPlan {
  // ... existing fields 1-10 ...
  int32 commission_rate = 11; // 佣金比例（万分比，500=5%）
}
```

**Step 3: Add PaymentNo and Amount to OrderCompletedEvent**

In `order/events/types.go`, update `OrderCompletedEvent`:
```go
type OrderCompletedEvent struct {
	OrderNo   string             `json:"order_no"`
	TenantID  int64              `json:"tenant_id"`
	PaymentNo string             `json:"payment_no"`
	Amount    int64              `json:"amount"`
	Items     []CompletedItemInfo `json:"items"`
}
```

**Step 4: Generate proto code**

Run: `make grpc`
Expected: Generated code in `api/proto/gen/account/v1/`

**Step 5: Update order service to include PaymentNo and Amount in completed event**

In `order/service/order.go`, find `ProduceCompleted` call and add `PaymentNo` and `Amount` fields. (The order domain model should already have these from the payment creation step.)

**Step 6: Commit**

```bash
git add api/proto/account/ api/proto/gen/account/ api/proto/tenant/v1/tenant.proto api/proto/gen/tenant/ order/events/types.go
git commit -m "feat: add account proto, commission_rate to tenant plan, and enrich order_completed event"
```

---

### Task 2: Account service — domain, DAO, repository

**Files:**
- Create: `account/domain/account.go`
- Create: `account/repository/dao/account.go`
- Create: `account/repository/cache/account.go`
- Create: `account/repository/account.go`

**Step 1: Create domain models**

`account/domain/account.go`:
```go
package domain

import "time"

type MerchantAccount struct {
	ID              int64
	TenantID        int64
	Balance         int64 // 可用余额（分）
	FrozenBalance   int64 // 冻结金额
	TotalIncome     int64
	TotalWithdrawn  int64
	TotalCommission int64
	BankAccountName string
	BankAccountNo   string
	BankName        string
	Status          AccountStatus
	Ctime           time.Time
	Utime           time.Time
}

type AccountStatus int32

const (
	AccountStatusActive AccountStatus = 1
	AccountStatusFrozen AccountStatus = 2
)

type SettlementRecord struct {
	ID               int64
	TenantID         int64
	SettlementNo     string
	OrderNo          string
	PaymentNo        string
	OrderAmount      int64
	CommissionRate   int32 // 万分比
	CommissionAmount int64
	SettlementAmount int64
	Status           SettlementStatus
	SettledAt        int64
	SettleDate       string // YYYY-MM-DD
	Ctime            time.Time
	Utime            time.Time
}

type SettlementStatus int32

const (
	SettlementStatusPending  SettlementStatus = 1
	SettlementStatusSettled  SettlementStatus = 2
	SettlementStatusReversed SettlementStatus = 3
)

type WithdrawalRecord struct {
	ID              int64
	TenantID        int64
	WithdrawalNo    string
	Amount          int64
	Status          WithdrawalStatus
	BankAccountName string
	BankAccountNo   string
	BankName        string
	RejectReason    string
	ReviewedAt      int64
	PaidAt          int64
	Ctime           time.Time
	Utime           time.Time
}

type WithdrawalStatus int32

const (
	WithdrawalStatusPendingReview WithdrawalStatus = 1
	WithdrawalStatusApproved     WithdrawalStatus = 2
	WithdrawalStatusRejected     WithdrawalStatus = 3
	WithdrawalStatusPaid         WithdrawalStatus = 4
)

type AccountTransaction struct {
	ID            int64
	TenantID      int64
	TransactionNo string
	Type          TransactionType
	Amount        int64 // 正=入账，负=扣款
	BalanceAfter  int64
	RefNo         string
	Remark        string
	Ctime         time.Time
}

type TransactionType int32

const (
	TxTypeSettlementCredit TransactionType = 1
	TxTypeWithdrawFreeze   TransactionType = 2
	TxTypeWithdrawDebit    TransactionType = 3
	TxTypeWithdrawReturn   TransactionType = 4
	TxTypeRefundDebit      TransactionType = 5
)
```

**Step 2: Create DAO layer**

`account/repository/dao/account.go`:
```go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ==================== GORM Models ====================

type MerchantAccountModel struct {
	ID              int64  `gorm:"primaryKey;autoIncrement"`
	TenantId        int64  `gorm:"uniqueIndex:uk_tenant_id"`
	Balance         int64
	FrozenBalance   int64
	TotalIncome     int64
	TotalWithdrawn  int64
	TotalCommission int64
	BankAccountName string `gorm:"type:varchar(128)"`
	BankAccountNo   string `gorm:"type:varchar(64)"`
	BankName        string `gorm:"type:varchar(128)"`
	Status          int32
	Ctime           int64
	Utime           int64
}

func (MerchantAccountModel) TableName() string { return "merchant_accounts" }

type SettlementRecordModel struct {
	ID               int64  `gorm:"primaryKey;autoIncrement"`
	TenantId         int64  `gorm:"index:idx_tenant_status"`
	SettlementNo     string `gorm:"type:varchar(64);uniqueIndex:uk_settlement_no"`
	OrderNo          string `gorm:"type:varchar(64);index:idx_order_no"`
	PaymentNo        string `gorm:"type:varchar(64)"`
	OrderAmount      int64
	CommissionRate   int32
	CommissionAmount int64
	SettlementAmount int64
	Status           int32 `gorm:"index:idx_tenant_status;index:idx_settle_date_status"`
	SettledAt        int64
	SettleDate       string `gorm:"type:varchar(10);index:idx_settle_date_status"`
	Ctime            int64
	Utime            int64
}

func (SettlementRecordModel) TableName() string { return "settlement_records" }

type WithdrawalRecordModel struct {
	ID              int64  `gorm:"primaryKey;autoIncrement"`
	TenantId        int64  `gorm:"index:idx_tenant_id"`
	WithdrawalNo    string `gorm:"type:varchar(64);uniqueIndex:uk_withdrawal_no"`
	Amount          int64
	Status          int32 `gorm:"index:idx_status"`
	BankAccountName string `gorm:"type:varchar(128)"`
	BankAccountNo   string `gorm:"type:varchar(64)"`
	BankName        string `gorm:"type:varchar(128)"`
	RejectReason    string `gorm:"type:varchar(512)"`
	ReviewedAt      int64
	PaidAt          int64
	Ctime           int64
	Utime           int64
}

func (WithdrawalRecordModel) TableName() string { return "withdrawal_records" }

type AccountTransactionModel struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	TenantId      int64  `gorm:"index:idx_tenant_type"`
	TransactionNo string `gorm:"type:varchar(64);uniqueIndex:uk_transaction_no"`
	Type          int32  `gorm:"index:idx_tenant_type"`
	Amount        int64
	BalanceAfter  int64
	RefNo         string `gorm:"type:varchar(64)"`
	Remark        string `gorm:"type:varchar(256)"`
	Ctime         int64
}

func (AccountTransactionModel) TableName() string { return "account_transactions" }

// ==================== InitTables ====================

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&MerchantAccountModel{},
		&SettlementRecordModel{},
		&WithdrawalRecordModel{},
		&AccountTransactionModel{},
	)
}

// ==================== DAO Interface ====================

type AccountDAO interface {
	// Account
	CreateAccount(ctx context.Context, account MerchantAccountModel) (MerchantAccountModel, error)
	FindAccountByTenantId(ctx context.Context, tenantId int64) (MerchantAccountModel, error)
	UpdateBankInfo(ctx context.Context, tenantId int64, name, no, bank string) error
	UpdateBalance(ctx context.Context, tenantId int64, balanceDelta, frozenDelta int64, extraUpdates map[string]any) error
	ListAccounts(ctx context.Context, offset, limit int) ([]MerchantAccountModel, int64, error)

	// Settlement
	CreateSettlement(ctx context.Context, record SettlementRecordModel) error
	FindPendingSettlements(ctx context.Context, settleDate string, limit int) ([]SettlementRecordModel, error)
	UpdateSettlementStatus(ctx context.Context, id int64, oldStatus, newStatus int32, updates map[string]any) error
	ListSettlements(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]SettlementRecordModel, int64, error)
	FindSettlementByOrderNo(ctx context.Context, orderNo string) (SettlementRecordModel, error)

	// Withdrawal
	CreateWithdrawal(ctx context.Context, record WithdrawalRecordModel) error
	FindWithdrawalById(ctx context.Context, id int64) (WithdrawalRecordModel, error)
	UpdateWithdrawalStatus(ctx context.Context, id int64, oldStatus, newStatus int32, updates map[string]any) error
	ListWithdrawals(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]WithdrawalRecordModel, int64, error)

	// Transaction
	CreateTransaction(ctx context.Context, tx AccountTransactionModel) error
	ListTransactions(ctx context.Context, tenantId int64, txType int32, offset, limit int) ([]AccountTransactionModel, int64, error)

	GetDB() *gorm.DB
}

// ==================== GORM Implementation ====================

type GORMAccountDAO struct {
	db *gorm.DB
}

func NewAccountDAO(db *gorm.DB) AccountDAO {
	return &GORMAccountDAO{db: db}
}

func (d *GORMAccountDAO) GetDB() *gorm.DB { return d.db }

func (d *GORMAccountDAO) CreateAccount(ctx context.Context, account MerchantAccountModel) (MerchantAccountModel, error) {
	now := time.Now().UnixMilli()
	account.Ctime = now
	account.Utime = now
	account.Status = 1
	err := d.db.WithContext(ctx).Create(&account).Error
	return account, err
}

func (d *GORMAccountDAO) FindAccountByTenantId(ctx context.Context, tenantId int64) (MerchantAccountModel, error) {
	var account MerchantAccountModel
	err := d.db.WithContext(ctx).Where("tenant_id = ?", tenantId).First(&account).Error
	return account, err
}

func (d *GORMAccountDAO) UpdateBankInfo(ctx context.Context, tenantId int64, name, no, bank string) error {
	return d.db.WithContext(ctx).Model(&MerchantAccountModel{}).
		Where("tenant_id = ?", tenantId).
		Updates(map[string]any{
			"bank_account_name": name,
			"bank_account_no":   no,
			"bank_name":         bank,
			"utime":             time.Now().UnixMilli(),
		}).Error
}

func (d *GORMAccountDAO) UpdateBalance(ctx context.Context, tenantId int64, balanceDelta, frozenDelta int64, extraUpdates map[string]any) error {
	updates := map[string]any{"utime": time.Now().UnixMilli()}
	for k, v := range extraUpdates {
		updates[k] = v
	}
	query := d.db.WithContext(ctx).Model(&MerchantAccountModel{}).Where("tenant_id = ?", tenantId)
	if balanceDelta != 0 {
		query = query.Update("balance", gorm.Expr("balance + ?", balanceDelta))
	}
	if frozenDelta != 0 {
		query = query.Update("frozen_balance", gorm.Expr("frozen_balance + ?", frozenDelta))
	}
	return query.Updates(updates).Error
}

func (d *GORMAccountDAO) ListAccounts(ctx context.Context, offset, limit int) ([]MerchantAccountModel, int64, error) {
	var accounts []MerchantAccountModel
	var total int64
	query := d.db.WithContext(ctx).Model(&MerchantAccountModel{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&accounts).Error
	return accounts, total, err
}

func (d *GORMAccountDAO) CreateSettlement(ctx context.Context, record SettlementRecordModel) error {
	now := time.Now().UnixMilli()
	record.Ctime = now
	record.Utime = now
	return d.db.WithContext(ctx).Create(&record).Error
}

func (d *GORMAccountDAO) FindPendingSettlements(ctx context.Context, settleDate string, limit int) ([]SettlementRecordModel, error) {
	var records []SettlementRecordModel
	err := d.db.WithContext(ctx).
		Where("settle_date <= ? AND status = ?", settleDate, 1).
		Order("id ASC").Limit(limit).Find(&records).Error
	return records, err
}

func (d *GORMAccountDAO) UpdateSettlementStatus(ctx context.Context, id int64, oldStatus, newStatus int32, updates map[string]any) error {
	updates["status"] = newStatus
	updates["utime"] = time.Now().UnixMilli()
	result := d.db.WithContext(ctx).Model(&SettlementRecordModel{}).
		Where("id = ? AND status = ?", id, oldStatus).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *GORMAccountDAO) ListSettlements(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]SettlementRecordModel, int64, error) {
	var records []SettlementRecordModel
	var total int64
	query := d.db.WithContext(ctx).Model(&SettlementRecordModel{})
	if tenantId > 0 {
		query = query.Where("tenant_id = ?", tenantId)
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

func (d *GORMAccountDAO) FindSettlementByOrderNo(ctx context.Context, orderNo string) (SettlementRecordModel, error) {
	var record SettlementRecordModel
	err := d.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&record).Error
	return record, err
}

func (d *GORMAccountDAO) CreateWithdrawal(ctx context.Context, record WithdrawalRecordModel) error {
	now := time.Now().UnixMilli()
	record.Ctime = now
	record.Utime = now
	return d.db.WithContext(ctx).Create(&record).Error
}

func (d *GORMAccountDAO) FindWithdrawalById(ctx context.Context, id int64) (WithdrawalRecordModel, error) {
	var record WithdrawalRecordModel
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	return record, err
}

func (d *GORMAccountDAO) UpdateWithdrawalStatus(ctx context.Context, id int64, oldStatus, newStatus int32, updates map[string]any) error {
	updates["status"] = newStatus
	updates["utime"] = time.Now().UnixMilli()
	result := d.db.WithContext(ctx).Model(&WithdrawalRecordModel{}).
		Where("id = ? AND status = ?", id, oldStatus).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *GORMAccountDAO) ListWithdrawals(ctx context.Context, tenantId int64, status int32, offset, limit int) ([]WithdrawalRecordModel, int64, error) {
	var records []WithdrawalRecordModel
	var total int64
	query := d.db.WithContext(ctx).Model(&WithdrawalRecordModel{})
	if tenantId > 0 {
		query = query.Where("tenant_id = ?", tenantId)
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

func (d *GORMAccountDAO) CreateTransaction(ctx context.Context, tx AccountTransactionModel) error {
	tx.Ctime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Create(&tx).Error
}

func (d *GORMAccountDAO) ListTransactions(ctx context.Context, tenantId int64, txType int32, offset, limit int) ([]AccountTransactionModel, int64, error) {
	var txs []AccountTransactionModel
	var total int64
	query := d.db.WithContext(ctx).Model(&AccountTransactionModel{})
	if tenantId > 0 {
		query = query.Where("tenant_id = ?", tenantId)
	}
	if txType > 0 {
		query = query.Where("type = ?", txType)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&txs).Error
	return txs, total, err
}
```

**Step 3: Create cache layer**

`account/repository/cache/account.go`:
```go
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rermrf/mall/account/repository/dao"
)

type AccountCache interface {
	GetAccount(ctx context.Context, tenantId int64) (dao.MerchantAccountModel, error)
	SetAccount(ctx context.Context, account dao.MerchantAccountModel) error
	DeleteAccount(ctx context.Context, tenantId int64) error
}

type RedisAccountCache struct {
	rdb redis.Cmdable
}

func NewAccountCache(rdb redis.Cmdable) AccountCache {
	return &RedisAccountCache{rdb: rdb}
}

func (c *RedisAccountCache) key(tenantId int64) string {
	return fmt.Sprintf("account:info:%d", tenantId)
}

func (c *RedisAccountCache) GetAccount(ctx context.Context, tenantId int64) (dao.MerchantAccountModel, error) {
	var account dao.MerchantAccountModel
	data, err := c.rdb.Get(ctx, c.key(tenantId)).Bytes()
	if err != nil {
		return account, err
	}
	err = json.Unmarshal(data, &account)
	return account, err
}

func (c *RedisAccountCache) SetAccount(ctx context.Context, account dao.MerchantAccountModel) error {
	data, err := json.Marshal(account)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.key(account.TenantId), data, 15*time.Minute).Err()
}

func (c *RedisAccountCache) DeleteAccount(ctx context.Context, tenantId int64) error {
	return c.rdb.Del(ctx, c.key(tenantId)).Err()
}
```

**Step 4: Create repository aggregation layer**

`account/repository/account.go`:
```go
package repository

import (
	"context"
	"time"

	"github.com/rermrf/mall/account/domain"
	"github.com/rermrf/mall/account/repository/cache"
	"github.com/rermrf/mall/account/repository/dao"
)

type AccountRepository interface {
	// Account
	CreateAccount(ctx context.Context, account domain.MerchantAccount) (domain.MerchantAccount, error)
	FindAccountByTenantId(ctx context.Context, tenantId int64) (domain.MerchantAccount, error)
	UpdateBankInfo(ctx context.Context, tenantId int64, name, no, bank string) error
	UpdateBalance(ctx context.Context, tenantId int64, balanceDelta, frozenDelta int64, extraUpdates map[string]any) error
	ListAccounts(ctx context.Context, page, pageSize int32) ([]domain.MerchantAccount, int64, error)

	// Settlement
	CreateSettlement(ctx context.Context, record domain.SettlementRecord) error
	FindPendingSettlements(ctx context.Context, settleDate string, limit int) ([]domain.SettlementRecord, error)
	UpdateSettlementStatus(ctx context.Context, id int64, oldStatus, newStatus domain.SettlementStatus, updates map[string]any) error
	ListSettlements(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SettlementRecord, int64, error)
	FindSettlementByOrderNo(ctx context.Context, orderNo string) (domain.SettlementRecord, error)

	// Withdrawal
	CreateWithdrawal(ctx context.Context, record domain.WithdrawalRecord) error
	FindWithdrawalById(ctx context.Context, id int64) (domain.WithdrawalRecord, error)
	UpdateWithdrawalStatus(ctx context.Context, id int64, oldStatus, newStatus domain.WithdrawalStatus, updates map[string]any) error
	ListWithdrawals(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.WithdrawalRecord, int64, error)

	// Transaction
	CreateTransaction(ctx context.Context, tx domain.AccountTransaction) error
	ListTransactions(ctx context.Context, tenantId int64, txType int32, page, pageSize int32) ([]domain.AccountTransaction, int64, error)
}

type accountRepository struct {
	dao   dao.AccountDAO
	cache cache.AccountCache
}

func NewAccountRepository(dao dao.AccountDAO, cache cache.AccountCache) AccountRepository {
	return &accountRepository{dao: dao, cache: cache}
}

// ========== Account ==========

func (r *accountRepository) CreateAccount(ctx context.Context, account domain.MerchantAccount) (domain.MerchantAccount, error) {
	model, err := r.dao.CreateAccount(ctx, r.toAccountModel(account))
	if err != nil {
		return domain.MerchantAccount{}, err
	}
	return r.toAccountDomain(model), nil
}

func (r *accountRepository) FindAccountByTenantId(ctx context.Context, tenantId int64) (domain.MerchantAccount, error) {
	cached, err := r.cache.GetAccount(ctx, tenantId)
	if err == nil {
		return r.toAccountDomain(cached), nil
	}
	model, err := r.dao.FindAccountByTenantId(ctx, tenantId)
	if err != nil {
		return domain.MerchantAccount{}, err
	}
	_ = r.cache.SetAccount(ctx, model)
	return r.toAccountDomain(model), nil
}

func (r *accountRepository) UpdateBankInfo(ctx context.Context, tenantId int64, name, no, bank string) error {
	err := r.dao.UpdateBankInfo(ctx, tenantId, name, no, bank)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteAccount(ctx, tenantId)
	return nil
}

func (r *accountRepository) UpdateBalance(ctx context.Context, tenantId int64, balanceDelta, frozenDelta int64, extraUpdates map[string]any) error {
	err := r.dao.UpdateBalance(ctx, tenantId, balanceDelta, frozenDelta, extraUpdates)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteAccount(ctx, tenantId)
	return nil
}

func (r *accountRepository) ListAccounts(ctx context.Context, page, pageSize int32) ([]domain.MerchantAccount, int64, error) {
	offset := int((page - 1) * pageSize)
	models, total, err := r.dao.ListAccounts(ctx, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	accounts := make([]domain.MerchantAccount, 0, len(models))
	for _, m := range models {
		accounts = append(accounts, r.toAccountDomain(m))
	}
	return accounts, total, nil
}

// ========== Settlement ==========

func (r *accountRepository) CreateSettlement(ctx context.Context, record domain.SettlementRecord) error {
	return r.dao.CreateSettlement(ctx, r.toSettlementModel(record))
}

func (r *accountRepository) FindPendingSettlements(ctx context.Context, settleDate string, limit int) ([]domain.SettlementRecord, error) {
	models, err := r.dao.FindPendingSettlements(ctx, settleDate, limit)
	if err != nil {
		return nil, err
	}
	records := make([]domain.SettlementRecord, 0, len(models))
	for _, m := range models {
		records = append(records, r.toSettlementDomain(m))
	}
	return records, nil
}

func (r *accountRepository) UpdateSettlementStatus(ctx context.Context, id int64, oldStatus, newStatus domain.SettlementStatus, updates map[string]any) error {
	return r.dao.UpdateSettlementStatus(ctx, id, int32(oldStatus), int32(newStatus), updates)
}

func (r *accountRepository) ListSettlements(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SettlementRecord, int64, error) {
	offset := int((page - 1) * pageSize)
	models, total, err := r.dao.ListSettlements(ctx, tenantId, status, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	records := make([]domain.SettlementRecord, 0, len(models))
	for _, m := range models {
		records = append(records, r.toSettlementDomain(m))
	}
	return records, total, nil
}

func (r *accountRepository) FindSettlementByOrderNo(ctx context.Context, orderNo string) (domain.SettlementRecord, error) {
	model, err := r.dao.FindSettlementByOrderNo(ctx, orderNo)
	if err != nil {
		return domain.SettlementRecord{}, err
	}
	return r.toSettlementDomain(model), nil
}

// ========== Withdrawal ==========

func (r *accountRepository) CreateWithdrawal(ctx context.Context, record domain.WithdrawalRecord) error {
	return r.dao.CreateWithdrawal(ctx, r.toWithdrawalModel(record))
}

func (r *accountRepository) FindWithdrawalById(ctx context.Context, id int64) (domain.WithdrawalRecord, error) {
	model, err := r.dao.FindWithdrawalById(ctx, id)
	if err != nil {
		return domain.WithdrawalRecord{}, err
	}
	return r.toWithdrawalDomain(model), nil
}

func (r *accountRepository) UpdateWithdrawalStatus(ctx context.Context, id int64, oldStatus, newStatus domain.WithdrawalStatus, updates map[string]any) error {
	return r.dao.UpdateWithdrawalStatus(ctx, id, int32(oldStatus), int32(newStatus), updates)
}

func (r *accountRepository) ListWithdrawals(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.WithdrawalRecord, int64, error) {
	offset := int((page - 1) * pageSize)
	models, total, err := r.dao.ListWithdrawals(ctx, tenantId, status, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	records := make([]domain.WithdrawalRecord, 0, len(models))
	for _, m := range models {
		records = append(records, r.toWithdrawalDomain(m))
	}
	return records, total, nil
}

// ========== Transaction ==========

func (r *accountRepository) CreateTransaction(ctx context.Context, tx domain.AccountTransaction) error {
	return r.dao.CreateTransaction(ctx, r.toTransactionModel(tx))
}

func (r *accountRepository) ListTransactions(ctx context.Context, tenantId int64, txType int32, page, pageSize int32) ([]domain.AccountTransaction, int64, error) {
	offset := int((page - 1) * pageSize)
	models, total, err := r.dao.ListTransactions(ctx, tenantId, txType, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	txs := make([]domain.AccountTransaction, 0, len(models))
	for _, m := range models {
		txs = append(txs, r.toTransactionDomain(m))
	}
	return txs, total, nil
}

// ========== Converters ==========

func (r *accountRepository) toAccountModel(a domain.MerchantAccount) dao.MerchantAccountModel {
	return dao.MerchantAccountModel{
		TenantId: a.TenantID, Balance: a.Balance, FrozenBalance: a.FrozenBalance,
		TotalIncome: a.TotalIncome, TotalWithdrawn: a.TotalWithdrawn, TotalCommission: a.TotalCommission,
		BankAccountName: a.BankAccountName, BankAccountNo: a.BankAccountNo, BankName: a.BankName,
		Status: int32(a.Status),
	}
}

func (r *accountRepository) toAccountDomain(m dao.MerchantAccountModel) domain.MerchantAccount {
	return domain.MerchantAccount{
		ID: m.ID, TenantID: m.TenantId, Balance: m.Balance, FrozenBalance: m.FrozenBalance,
		TotalIncome: m.TotalIncome, TotalWithdrawn: m.TotalWithdrawn, TotalCommission: m.TotalCommission,
		BankAccountName: m.BankAccountName, BankAccountNo: m.BankAccountNo, BankName: m.BankName,
		Status: domain.AccountStatus(m.Status),
		Ctime:  time.UnixMilli(m.Ctime), Utime: time.UnixMilli(m.Utime),
	}
}

func (r *accountRepository) toSettlementModel(s domain.SettlementRecord) dao.SettlementRecordModel {
	return dao.SettlementRecordModel{
		TenantId: s.TenantID, SettlementNo: s.SettlementNo, OrderNo: s.OrderNo, PaymentNo: s.PaymentNo,
		OrderAmount: s.OrderAmount, CommissionRate: s.CommissionRate,
		CommissionAmount: s.CommissionAmount, SettlementAmount: s.SettlementAmount,
		Status: int32(s.Status), SettledAt: s.SettledAt, SettleDate: s.SettleDate,
	}
}

func (r *accountRepository) toSettlementDomain(m dao.SettlementRecordModel) domain.SettlementRecord {
	return domain.SettlementRecord{
		ID: m.ID, TenantID: m.TenantId, SettlementNo: m.SettlementNo, OrderNo: m.OrderNo, PaymentNo: m.PaymentNo,
		OrderAmount: m.OrderAmount, CommissionRate: m.CommissionRate,
		CommissionAmount: m.CommissionAmount, SettlementAmount: m.SettlementAmount,
		Status: domain.SettlementStatus(m.Status), SettledAt: m.SettledAt, SettleDate: m.SettleDate,
		Ctime: time.UnixMilli(m.Ctime), Utime: time.UnixMilli(m.Utime),
	}
}

func (r *accountRepository) toWithdrawalModel(w domain.WithdrawalRecord) dao.WithdrawalRecordModel {
	return dao.WithdrawalRecordModel{
		TenantId: w.TenantID, WithdrawalNo: w.WithdrawalNo, Amount: w.Amount, Status: int32(w.Status),
		BankAccountName: w.BankAccountName, BankAccountNo: w.BankAccountNo, BankName: w.BankName,
		RejectReason: w.RejectReason, ReviewedAt: w.ReviewedAt, PaidAt: w.PaidAt,
	}
}

func (r *accountRepository) toWithdrawalDomain(m dao.WithdrawalRecordModel) domain.WithdrawalRecord {
	return domain.WithdrawalRecord{
		ID: m.ID, TenantID: m.TenantId, WithdrawalNo: m.WithdrawalNo, Amount: m.Amount,
		Status: domain.WithdrawalStatus(m.Status),
		BankAccountName: m.BankAccountName, BankAccountNo: m.BankAccountNo, BankName: m.BankName,
		RejectReason: m.RejectReason, ReviewedAt: m.ReviewedAt, PaidAt: m.PaidAt,
		Ctime: time.UnixMilli(m.Ctime), Utime: time.UnixMilli(m.Utime),
	}
}

func (r *accountRepository) toTransactionModel(t domain.AccountTransaction) dao.AccountTransactionModel {
	return dao.AccountTransactionModel{
		TenantId: t.TenantID, TransactionNo: t.TransactionNo, Type: int32(t.Type),
		Amount: t.Amount, BalanceAfter: t.BalanceAfter, RefNo: t.RefNo, Remark: t.Remark,
	}
}

func (r *accountRepository) toTransactionDomain(m dao.AccountTransactionModel) domain.AccountTransaction {
	return domain.AccountTransaction{
		ID: m.ID, TenantID: m.TenantId, TransactionNo: m.TransactionNo,
		Type: domain.TransactionType(m.Type), Amount: m.Amount, BalanceAfter: m.BalanceAfter,
		RefNo: m.RefNo, Remark: m.Remark, Ctime: time.UnixMilli(m.Ctime),
	}
}
```

**Step 2: Commit**

```bash
git add account/domain/ account/repository/
git commit -m "feat(account): add domain models, DAO, cache, and repository layers"
```

---

### Task 3: Account service — service layer, events, gRPC server

**Files:**
- Create: `account/service/account.go`
- Create: `account/service/settlement_job.go`
- Create: `account/events/consumer.go`
- Create: `account/events/types.go`
- Create: `account/integration/tenant.go`
- Create: `account/grpc/account.go`

This task creates the core business logic. The service layer handles settlement creation, T+N batch execution, withdrawal with freeze/unfreeze, and transaction ledger recording. The event consumer listens for `order_completed` and `refund_completed`. The gRPC server exposes all RPCs.

**Due to the size of this task, the implementer should:**
1. Read the plan's design doc at `docs/plans/2026-03-19-merchant-account-design.md` for the complete data flow
2. Follow the exact patterns from `payment/service/payment.go` and `payment/grpc/payment.go`
3. The service interface should match the proto's AccountService RPCs
4. The event consumer follows `order/events/consumer.go` pattern using `saramax.NewHandler`
5. The integration layer calls tenant service via gRPC to get commission_rate from the tenant's plan
6. The settlement job uses a simple ticker or is called via gRPC `ExecuteSettlement`

**Commit:**
```bash
git add account/service/ account/events/ account/integration/ account/grpc/
git commit -m "feat(account): add service layer, event consumers, tenant integration, and gRPC server"
```

---

### Task 4: Account service — IoC, config, wire, main, app

**Files:**
- Create: `account/config/dev.yaml`
- Create: `account/ioc/db.go`
- Create: `account/ioc/redis.go`
- Create: `account/ioc/kafka.go`
- Create: `account/ioc/grpc.go`
- Create: `account/ioc/logger.go`
- Create: `account/ioc/snowflake.go`
- Create: `account/ioc/tenant.go`
- Create: `account/app.go`
- Create: `account/main.go`
- Create: `account/wire.go`

Follow the payment service's IoC pattern exactly. Config uses port 8087, DB `mall_account`, Redis db 6.

The `ioc/tenant.go` initializes the tenant gRPC client via etcd discovery for fetching commission rates.

After creating all files, run `cd account && wire` to generate `wire_gen.go`.

**Commit:**
```bash
git add account/
git commit -m "feat(account): add IoC, config, wire, and service entry point"
```

---

### Task 5: Merchant BFF — account handler and routes

**Files:**
- Create: `merchant-bff/handler/account.go`
- Modify: `merchant-bff/ioc/gin.go` (add routes)
- Modify: `merchant-bff/ioc/grpc.go` (add account client)
- Modify: `merchant-bff/wire.go` (add account handler)

New merchant BFF endpoints:
```
GET  /api/v1/account               → GetAccount
PUT  /api/v1/account/bank-info     → UpdateBankInfo
GET  /api/v1/settlements           → ListSettlements
POST /api/v1/withdrawals           → RequestWithdrawal
GET  /api/v1/withdrawals           → ListWithdrawals
GET  /api/v1/transactions          → ListTransactions
GET  /api/v1/account/summary       → GetAccountSummary
```

Follow `merchant-bff/handler/payment.go` pattern for handler struct, request structs, and ginx.Result responses.

**Commit:**
```bash
git add merchant-bff/handler/account.go merchant-bff/ioc/ merchant-bff/wire.go merchant-bff/wire_gen.go
git commit -m "feat(merchant-bff): add account, settlement, and withdrawal endpoints"
```

---

### Task 6: Admin BFF — account handler and routes

**Files:**
- Create: `admin-bff/handler/account.go`
- Modify: `admin-bff/ioc/gin.go` (add routes)
- Modify: `admin-bff/ioc/grpc.go` (add account client)
- Modify: `admin-bff/wire.go` (add account handler)

New admin BFF endpoints:
```
GET  /api/v1/accounts              → ListAccounts
GET  /api/v1/accounts/:tenantId    → GetAccount
GET  /api/v1/settlements           → ListSettlements (all tenants)
GET  /api/v1/withdrawals           → ListWithdrawals (all tenants)
POST /api/v1/withdrawals/:id/review   → ReviewWithdrawal
POST /api/v1/withdrawals/:id/confirm  → ConfirmWithdrawalPaid
GET  /api/v1/transactions          → ListTransactions (all tenants)
```

**Commit:**
```bash
git add admin-bff/handler/account.go admin-bff/ioc/ admin-bff/wire.go admin-bff/wire_gen.go
git commit -m "feat(admin-bff): add account supervision and withdrawal review endpoints"
```

---

### Task 7: Tenant service — add commission_rate to plan

**Files:**
- Modify: `tenant/repository/dao/tenant.go` (add CommissionRate field to TenantPlanModel)
- Modify: `tenant/domain/tenant.go` (add CommissionRate to domain Plan)
- Modify: `tenant/grpc/tenant.go` (include commission_rate in DTO conversion)

This is a small change: add `CommissionRate int32` field to TenantPlanModel and Plan domain, and map it in the gRPC conversion.

**Commit:**
```bash
git add tenant/
git commit -m "feat(tenant): add commission_rate to subscription plan"
```

---

### Task 8: Order service — enrich order_completed event

**Files:**
- Modify: `order/events/types.go` (already done in Task 1)
- Modify: `order/service/order.go` (include PaymentNo and Amount in ProduceCompleted call)

Find the `ProduceCompleted` call in `order/service/order.go:332` and add the `PaymentNo` and `Amount` fields from the order domain model.

**Commit:**
```bash
git add order/
git commit -m "feat(order): include payment_no and amount in order_completed event"
```

---

### Task 9: Docker Compose — add account service

**Files:**
- Modify: `docker-compose.yml` (add account service)
- Modify: `Makefile` (add account targets)

Add `account` service following the existing service pattern with `SERVICE=account` build arg.

**Commit:**
```bash
git add docker-compose.yml Makefile
git commit -m "feat: add account service to Docker Compose and Makefile"
```

---

### Task 10: Build verification

**Step 1:** `cd account && go build ./...`
**Step 2:** `cd merchant-bff && go build ./...`
**Step 3:** `cd admin-bff && go build ./...`
**Step 4:** `cd order && go build ./...`
**Step 5:** `cd tenant && go build ./...`

Fix any compilation errors.

**Commit:**
```bash
git add -A
git commit -m "feat(account): complete merchant account service with settlement, withdrawal, and ledger"
```

---

## Summary

| Task | Description | Scope |
|------|-------------|-------|
| 1 | Proto definitions + code gen | account.proto, tenant.proto commission_rate, order event enrichment |
| 2 | Domain, DAO, Cache, Repository | account service data layer (4 tables) |
| 3 | Service, Events, Integration, gRPC | Core business logic + Kafka consumers + tenant integration |
| 4 | IoC, Config, Wire, Main | Service bootstrap and dependency injection |
| 5 | Merchant BFF handler | 7 new merchant endpoints |
| 6 | Admin BFF handler | 7 new admin endpoints |
| 7 | Tenant service change | Add commission_rate to plan |
| 8 | Order service change | Enrich order_completed event |
| 9 | Docker/Makefile | Infrastructure integration |
| 10 | Build verification | Ensure all services compile |
