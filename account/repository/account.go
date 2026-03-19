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
