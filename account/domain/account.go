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
