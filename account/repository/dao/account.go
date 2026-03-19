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
