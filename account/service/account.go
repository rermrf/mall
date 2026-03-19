package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/account/domain"
	"github.com/rermrf/mall/account/integration"
	"github.com/rermrf/mall/account/repository"
	"github.com/rermrf/mall/pkg/snowflake"
	"github.com/spf13/viper"
)

type AccountService interface {
	// Account
	GetAccount(ctx context.Context, tenantId int64) (domain.MerchantAccount, error)
	CreateAccount(ctx context.Context, tenantId int64) (int64, error)
	UpdateBankInfo(ctx context.Context, tenantId int64, name, no, bank string) error

	// Settlement
	CreateSettlement(ctx context.Context, tenantId int64, orderNo, paymentNo string, orderAmount int64) (string, error)
	ExecuteSettlement(ctx context.Context, settleDate string) (int32, int64, error)
	ListSettlements(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SettlementRecord, int64, error)

	// Withdrawal
	RequestWithdrawal(ctx context.Context, tenantId int64, amount int64) (string, error)
	ReviewWithdrawal(ctx context.Context, id int64, approved bool, rejectReason string) error
	ConfirmWithdrawalPaid(ctx context.Context, id int64) error
	ListWithdrawals(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.WithdrawalRecord, int64, error)

	// Transaction & Summary
	ListTransactions(ctx context.Context, tenantId int64, txType int32, page, pageSize int32) ([]domain.AccountTransaction, int64, error)
	GetAccountSummary(ctx context.Context, tenantId int64) (domain.MerchantAccount, int64, error)

	// Admin
	ListAccounts(ctx context.Context, page, pageSize int32) ([]domain.MerchantAccount, int64, error)
}

type accountService struct {
	repo       repository.AccountRepository
	tenantSvc  integration.TenantIntegration
	node       *snowflake.Node
	l          logger.Logger
}

func NewAccountService(
	repo repository.AccountRepository,
	tenantSvc integration.TenantIntegration,
	node *snowflake.Node,
	l logger.Logger,
) AccountService {
	return &accountService{
		repo:       repo,
		tenantSvc:  tenantSvc,
		node:       node,
		l:          l,
	}
}

// ========== Account ==========

func (s *accountService) GetAccount(ctx context.Context, tenantId int64) (domain.MerchantAccount, error) {
	return s.repo.FindAccountByTenantId(ctx, tenantId)
}

func (s *accountService) CreateAccount(ctx context.Context, tenantId int64) (int64, error) {
	account, err := s.repo.CreateAccount(ctx, domain.MerchantAccount{
		TenantID: tenantId,
		Status:   domain.AccountStatusActive,
	})
	if err != nil {
		return 0, fmt.Errorf("创建商户账户失败: %w", err)
	}
	return account.ID, nil
}

func (s *accountService) UpdateBankInfo(ctx context.Context, tenantId int64, name, no, bank string) error {
	if name == "" || no == "" || bank == "" {
		return fmt.Errorf("银行信息不完整")
	}
	return s.repo.UpdateBankInfo(ctx, tenantId, name, no, bank)
}

// ========== Settlement ==========

func (s *accountService) CreateSettlement(ctx context.Context, tenantId int64, orderNo, paymentNo string, orderAmount int64) (string, error) {
	// 幂等检查：如果该订单已有结算记录，直接返回
	existing, err := s.repo.FindSettlementByOrderNo(ctx, orderNo)
	if err == nil && existing.ID > 0 {
		s.l.Info("结算记录已存在，跳过",
			logger.String("orderNo", orderNo),
			logger.String("settlementNo", existing.SettlementNo))
		return existing.SettlementNo, nil
	}

	// 确保商户账户存在，不存在则自动创建
	_, err = s.repo.FindAccountByTenantId(ctx, tenantId)
	if err != nil {
		s.l.Info("商户账户不存在，自动创建", logger.Int64("tenantId", tenantId))
		_, err = s.CreateAccount(ctx, tenantId)
		if err != nil {
			return "", fmt.Errorf("自动创建商户账户失败: %w", err)
		}
	}

	// 获取佣金比例
	commissionRate, err := s.tenantSvc.GetCommissionRate(ctx, tenantId)
	if err != nil {
		s.l.Warn("获取佣金比例失败，使用默认值", logger.Error(err))
		commissionRate = int32(viper.GetInt("settlement.defaultCommissionRate"))
	}
	if commissionRate <= 0 {
		commissionRate = int32(viper.GetInt("settlement.defaultCommissionRate"))
	}

	// 计算佣金和结算金额
	commissionAmount := orderAmount * int64(commissionRate) / 10000
	settlementAmount := orderAmount - commissionAmount

	// T+N 结算日期
	settleDays := viper.GetInt("settlement.settleDays")
	if settleDays <= 0 {
		settleDays = 7
	}
	settleDate := time.Now().AddDate(0, 0, settleDays).Format("2006-01-02")

	settlementNo := fmt.Sprintf("S%d", s.node.Generate())
	record := domain.SettlementRecord{
		TenantID:         tenantId,
		SettlementNo:     settlementNo,
		OrderNo:          orderNo,
		PaymentNo:        paymentNo,
		OrderAmount:      orderAmount,
		CommissionRate:   commissionRate,
		CommissionAmount: commissionAmount,
		SettlementAmount: settlementAmount,
		Status:           domain.SettlementStatusPending,
		SettleDate:       settleDate,
	}
	if err := s.repo.CreateSettlement(ctx, record); err != nil {
		return "", fmt.Errorf("创建结算记录失败: %w", err)
	}
	s.l.Info("创建结算记录成功",
		logger.String("settlementNo", settlementNo),
		logger.String("orderNo", orderNo),
		logger.Int64("orderAmount", orderAmount),
		logger.Int64("commissionAmount", commissionAmount),
		logger.Int64("settlementAmount", settlementAmount),
		logger.String("settleDate", settleDate))
	return settlementNo, nil
}

func (s *accountService) ExecuteSettlement(ctx context.Context, settleDate string) (int32, int64, error) {
	var settledCount int32
	var settledAmount int64

	for {
		records, err := s.repo.FindPendingSettlements(ctx, settleDate, 100)
		if err != nil {
			return settledCount, settledAmount, fmt.Errorf("查询待结算记录失败: %w", err)
		}
		if len(records) == 0 {
			break
		}
		for _, record := range records {
			if err := s.settleOne(ctx, record); err != nil {
				s.l.Error("结算单笔记录失败",
					logger.String("settlementNo", record.SettlementNo),
					logger.Error(err))
				continue
			}
			settledCount++
			settledAmount += record.SettlementAmount
		}
	}
	s.l.Info("批量结算完成",
		logger.String("settleDate", settleDate),
		logger.Int32("settledCount", settledCount),
		logger.Int64("settledAmount", settledAmount))
	return settledCount, settledAmount, nil
}

func (s *accountService) settleOne(ctx context.Context, record domain.SettlementRecord) error {
	now := time.Now().UnixMilli()

	// 更新结算记录状态
	err := s.repo.UpdateSettlementStatus(ctx, record.ID,
		domain.SettlementStatusPending, domain.SettlementStatusSettled,
		map[string]any{"settled_at": now})
	if err != nil {
		return fmt.Errorf("更新结算记录状态失败: %w", err)
	}

	// 增加商户可用余额
	err = s.repo.UpdateBalance(ctx, record.TenantID, record.SettlementAmount, 0, map[string]any{
		"total_income":     fmt.Sprintf("total_income + %d", record.SettlementAmount),
		"total_commission": fmt.Sprintf("total_commission + %d", record.CommissionAmount),
	})
	if err != nil {
		s.l.Error("更新商户余额失败",
			logger.String("settlementNo", record.SettlementNo),
			logger.Error(err))
	}

	// 查询更新后的账户余额
	account, err := s.repo.FindAccountByTenantId(ctx, record.TenantID)
	if err != nil {
		s.l.Error("查询账户余额失败", logger.Error(err))
	}

	// 记录交易流水
	txNo := fmt.Sprintf("T%d", s.node.Generate())
	_ = s.repo.CreateTransaction(ctx, domain.AccountTransaction{
		TenantID:      record.TenantID,
		TransactionNo: txNo,
		Type:          domain.TxTypeSettlementCredit,
		Amount:        record.SettlementAmount,
		BalanceAfter:  account.Balance,
		RefNo:         record.SettlementNo,
		Remark:        fmt.Sprintf("订单结算入账，订单号：%s", record.OrderNo),
	})

	return nil
}

func (s *accountService) ListSettlements(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.SettlementRecord, int64, error) {
	return s.repo.ListSettlements(ctx, tenantId, status, page, pageSize)
}

// ========== Withdrawal ==========

func (s *accountService) RequestWithdrawal(ctx context.Context, tenantId int64, amount int64) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("提现金额必须大于0")
	}

	account, err := s.repo.FindAccountByTenantId(ctx, tenantId)
	if err != nil {
		return "", fmt.Errorf("商户账户不存在: %w", err)
	}
	if account.Status != domain.AccountStatusActive {
		return "", fmt.Errorf("账户已冻结，无法提现")
	}
	if account.Balance < amount {
		return "", fmt.Errorf("可用余额不足，当前余额: %d，申请提现: %d", account.Balance, amount)
	}
	if account.BankAccountNo == "" {
		return "", fmt.Errorf("请先绑定银行账户")
	}

	withdrawalNo := fmt.Sprintf("W%d", s.node.Generate())
	record := domain.WithdrawalRecord{
		TenantID:        tenantId,
		WithdrawalNo:    withdrawalNo,
		Amount:          amount,
		Status:          domain.WithdrawalStatusPendingReview,
		BankAccountName: account.BankAccountName,
		BankAccountNo:   account.BankAccountNo,
		BankName:        account.BankName,
	}

	// 冻结资金
	err = s.repo.UpdateBalance(ctx, tenantId, -amount, amount, nil)
	if err != nil {
		return "", fmt.Errorf("冻结资金失败: %w", err)
	}

	if err := s.repo.CreateWithdrawal(ctx, record); err != nil {
		// 回滚冻结
		_ = s.repo.UpdateBalance(ctx, tenantId, amount, -amount, nil)
		return "", fmt.Errorf("创建提现记录失败: %w", err)
	}

	// 查询更新后的账户余额
	account, _ = s.repo.FindAccountByTenantId(ctx, tenantId)

	// 记录交易流水
	txNo := fmt.Sprintf("T%d", s.node.Generate())
	_ = s.repo.CreateTransaction(ctx, domain.AccountTransaction{
		TenantID:      tenantId,
		TransactionNo: txNo,
		Type:          domain.TxTypeWithdrawFreeze,
		Amount:        -amount,
		BalanceAfter:  account.Balance,
		RefNo:         withdrawalNo,
		Remark:        "提现冻结",
	})

	s.l.Info("提现申请成功",
		logger.String("withdrawalNo", withdrawalNo),
		logger.Int64("amount", amount))
	return withdrawalNo, nil
}

func (s *accountService) ReviewWithdrawal(ctx context.Context, id int64, approved bool, rejectReason string) error {
	record, err := s.repo.FindWithdrawalById(ctx, id)
	if err != nil {
		return fmt.Errorf("提现记录不存在: %w", err)
	}
	if record.Status != domain.WithdrawalStatusPendingReview {
		return fmt.Errorf("提现记录状态不允许审核: %d", record.Status)
	}

	now := time.Now().UnixMilli()
	if approved {
		// 审核通过：保持冻结状态，等待打款
		err = s.repo.UpdateWithdrawalStatus(ctx, id,
			domain.WithdrawalStatusPendingReview, domain.WithdrawalStatusApproved,
			map[string]any{"reviewed_at": now})
	} else {
		// 审核拒绝：解冻资金
		err = s.repo.UpdateWithdrawalStatus(ctx, id,
			domain.WithdrawalStatusPendingReview, domain.WithdrawalStatusRejected,
			map[string]any{"reviewed_at": now, "reject_reason": rejectReason})
		if err != nil {
			return err
		}

		// 解冻资金
		err = s.repo.UpdateBalance(ctx, record.TenantID, record.Amount, -record.Amount, nil)
		if err != nil {
			s.l.Error("解冻资金失败",
				logger.String("withdrawalNo", record.WithdrawalNo),
				logger.Error(err))
		}

		// 查询更新后的账户余额
		account, _ := s.repo.FindAccountByTenantId(ctx, record.TenantID)

		// 记录交易流水
		txNo := fmt.Sprintf("T%d", s.node.Generate())
		_ = s.repo.CreateTransaction(ctx, domain.AccountTransaction{
			TenantID:      record.TenantID,
			TransactionNo: txNo,
			Type:          domain.TxTypeWithdrawReturn,
			Amount:        record.Amount,
			BalanceAfter:  account.Balance,
			RefNo:         record.WithdrawalNo,
			Remark:        fmt.Sprintf("提现审核拒绝，资金退回，原因：%s", rejectReason),
		})
	}
	return err
}

func (s *accountService) ConfirmWithdrawalPaid(ctx context.Context, id int64) error {
	record, err := s.repo.FindWithdrawalById(ctx, id)
	if err != nil {
		return fmt.Errorf("提现记录不存在: %w", err)
	}
	if record.Status != domain.WithdrawalStatusApproved {
		return fmt.Errorf("提现记录状态不允许确认打款: %d", record.Status)
	}

	now := time.Now().UnixMilli()
	err = s.repo.UpdateWithdrawalStatus(ctx, id,
		domain.WithdrawalStatusApproved, domain.WithdrawalStatusPaid,
		map[string]any{"paid_at": now})
	if err != nil {
		return fmt.Errorf("更新提现状态失败: %w", err)
	}

	// 扣除冻结余额
	err = s.repo.UpdateBalance(ctx, record.TenantID, 0, -record.Amount, map[string]any{
		"total_withdrawn": fmt.Sprintf("total_withdrawn + %d", record.Amount),
	})
	if err != nil {
		s.l.Error("扣除冻结余额失败",
			logger.String("withdrawalNo", record.WithdrawalNo),
			logger.Error(err))
	}

	// 查询更新后的账户余额
	account, _ := s.repo.FindAccountByTenantId(ctx, record.TenantID)

	// 记录交易流水
	txNo := fmt.Sprintf("T%d", s.node.Generate())
	_ = s.repo.CreateTransaction(ctx, domain.AccountTransaction{
		TenantID:      record.TenantID,
		TransactionNo: txNo,
		Type:          domain.TxTypeWithdrawDebit,
		Amount:        -record.Amount,
		BalanceAfter:  account.Balance,
		RefNo:         record.WithdrawalNo,
		Remark:        "提现打款完成",
	})

	s.l.Info("提现打款确认成功",
		logger.String("withdrawalNo", record.WithdrawalNo),
		logger.Int64("amount", record.Amount))
	return nil
}

func (s *accountService) ListWithdrawals(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.WithdrawalRecord, int64, error) {
	return s.repo.ListWithdrawals(ctx, tenantId, status, page, pageSize)
}

// ========== Transaction & Summary ==========

func (s *accountService) ListTransactions(ctx context.Context, tenantId int64, txType int32, page, pageSize int32) ([]domain.AccountTransaction, int64, error) {
	return s.repo.ListTransactions(ctx, tenantId, txType, page, pageSize)
}

func (s *accountService) GetAccountSummary(ctx context.Context, tenantId int64) (domain.MerchantAccount, int64, error) {
	account, err := s.repo.FindAccountByTenantId(ctx, tenantId)
	if err != nil {
		return domain.MerchantAccount{}, 0, err
	}

	// 查询待结算金额
	pendingRecords, _, err := s.repo.ListSettlements(ctx, tenantId, int32(domain.SettlementStatusPending), 1, 10000)
	if err != nil {
		return account, 0, nil
	}
	var pendingAmount int64
	for _, r := range pendingRecords {
		pendingAmount += r.SettlementAmount
	}
	return account, pendingAmount, nil
}

// ========== Admin ==========

func (s *accountService) ListAccounts(ctx context.Context, page, pageSize int32) ([]domain.MerchantAccount, int64, error) {
	return s.repo.ListAccounts(ctx, page, pageSize)
}
