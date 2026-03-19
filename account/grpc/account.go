package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	accountv1 "github.com/rermrf/mall/api/proto/gen/account/v1"
	"github.com/rermrf/mall/account/domain"
	"github.com/rermrf/mall/account/service"
)

type AccountGRPCServer struct {
	accountv1.UnimplementedAccountServiceServer
	svc service.AccountService
}

func NewAccountGRPCServer(svc service.AccountService) *AccountGRPCServer {
	return &AccountGRPCServer{svc: svc}
}

func (s *AccountGRPCServer) Register(server *grpc.Server) {
	accountv1.RegisterAccountServiceServer(server, s)
}

// ========== Account ==========

func (s *AccountGRPCServer) GetAccount(ctx context.Context, req *accountv1.GetAccountRequest) (*accountv1.GetAccountResponse, error) {
	account, err := s.svc.GetAccount(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	return &accountv1.GetAccountResponse{Account: s.toAccountDTO(account)}, nil
}

func (s *AccountGRPCServer) CreateAccount(ctx context.Context, req *accountv1.CreateAccountRequest) (*accountv1.CreateAccountResponse, error) {
	id, err := s.svc.CreateAccount(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	return &accountv1.CreateAccountResponse{Id: id}, nil
}

func (s *AccountGRPCServer) UpdateBankInfo(ctx context.Context, req *accountv1.UpdateBankInfoRequest) (*accountv1.UpdateBankInfoResponse, error) {
	err := s.svc.UpdateBankInfo(ctx, req.GetTenantId(), req.GetBankAccountName(), req.GetBankAccountNo(), req.GetBankName())
	if err != nil {
		return nil, err
	}
	return &accountv1.UpdateBankInfoResponse{}, nil
}

// ========== Settlement ==========

func (s *AccountGRPCServer) CreateSettlement(ctx context.Context, req *accountv1.CreateSettlementRequest) (*accountv1.CreateSettlementResponse, error) {
	settlementNo, err := s.svc.CreateSettlement(ctx, req.GetTenantId(), req.GetOrderNo(), req.GetPaymentNo(), req.GetOrderAmount())
	if err != nil {
		return nil, err
	}
	return &accountv1.CreateSettlementResponse{SettlementNo: settlementNo}, nil
}

func (s *AccountGRPCServer) ListSettlements(ctx context.Context, req *accountv1.ListSettlementsRequest) (*accountv1.ListSettlementsResponse, error) {
	records, total, err := s.svc.ListSettlements(ctx, req.GetTenantId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*accountv1.SettlementRecord, 0, len(records))
	for _, r := range records {
		dtos = append(dtos, s.toSettlementDTO(r))
	}
	return &accountv1.ListSettlementsResponse{Settlements: dtos, Total: total}, nil
}

func (s *AccountGRPCServer) ExecuteSettlement(ctx context.Context, req *accountv1.ExecuteSettlementRequest) (*accountv1.ExecuteSettlementResponse, error) {
	count, amount, err := s.svc.ExecuteSettlement(ctx, req.GetSettleDate())
	if err != nil {
		return nil, err
	}
	return &accountv1.ExecuteSettlementResponse{SettledCount: count, SettledAmount: amount}, nil
}

// ========== Withdrawal ==========

func (s *AccountGRPCServer) RequestWithdrawal(ctx context.Context, req *accountv1.RequestWithdrawalRequest) (*accountv1.RequestWithdrawalResponse, error) {
	withdrawalNo, err := s.svc.RequestWithdrawal(ctx, req.GetTenantId(), req.GetAmount())
	if err != nil {
		return nil, err
	}
	return &accountv1.RequestWithdrawalResponse{WithdrawalNo: withdrawalNo}, nil
}

func (s *AccountGRPCServer) ReviewWithdrawal(ctx context.Context, req *accountv1.ReviewWithdrawalRequest) (*accountv1.ReviewWithdrawalResponse, error) {
	err := s.svc.ReviewWithdrawal(ctx, req.GetId(), req.GetApproved(), req.GetRejectReason())
	if err != nil {
		return nil, err
	}
	return &accountv1.ReviewWithdrawalResponse{}, nil
}

func (s *AccountGRPCServer) ConfirmWithdrawalPaid(ctx context.Context, req *accountv1.ConfirmWithdrawalPaidRequest) (*accountv1.ConfirmWithdrawalPaidResponse, error) {
	err := s.svc.ConfirmWithdrawalPaid(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &accountv1.ConfirmWithdrawalPaidResponse{}, nil
}

func (s *AccountGRPCServer) ListWithdrawals(ctx context.Context, req *accountv1.ListWithdrawalsRequest) (*accountv1.ListWithdrawalsResponse, error) {
	records, total, err := s.svc.ListWithdrawals(ctx, req.GetTenantId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*accountv1.WithdrawalRecord, 0, len(records))
	for _, r := range records {
		dtos = append(dtos, s.toWithdrawalDTO(r))
	}
	return &accountv1.ListWithdrawalsResponse{Withdrawals: dtos, Total: total}, nil
}

// ========== Transaction & Summary ==========

func (s *AccountGRPCServer) ListTransactions(ctx context.Context, req *accountv1.ListTransactionsRequest) (*accountv1.ListTransactionsResponse, error) {
	txs, total, err := s.svc.ListTransactions(ctx, req.GetTenantId(), req.GetType(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*accountv1.AccountTransaction, 0, len(txs))
	for _, t := range txs {
		dtos = append(dtos, s.toTransactionDTO(t))
	}
	return &accountv1.ListTransactionsResponse{Transactions: dtos, Total: total}, nil
}

func (s *AccountGRPCServer) GetAccountSummary(ctx context.Context, req *accountv1.GetAccountSummaryRequest) (*accountv1.GetAccountSummaryResponse, error) {
	account, pendingAmount, err := s.svc.GetAccountSummary(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	return &accountv1.GetAccountSummaryResponse{
		Summary: &accountv1.AccountSummary{
			Balance:           account.Balance,
			FrozenBalance:     account.FrozenBalance,
			PendingSettlement: pendingAmount,
		},
	}, nil
}

// ========== Admin ==========

func (s *AccountGRPCServer) ListAccounts(ctx context.Context, req *accountv1.ListAccountsRequest) (*accountv1.ListAccountsResponse, error) {
	accounts, total, err := s.svc.ListAccounts(ctx, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*accountv1.MerchantAccount, 0, len(accounts))
	for _, a := range accounts {
		dtos = append(dtos, s.toAccountDTO(a))
	}
	return &accountv1.ListAccountsResponse{Accounts: dtos, Total: total}, nil
}

// ========== DTO Converters ==========

func (s *AccountGRPCServer) toAccountDTO(a domain.MerchantAccount) *accountv1.MerchantAccount {
	return &accountv1.MerchantAccount{
		Id:              a.ID,
		TenantId:        a.TenantID,
		Balance:         a.Balance,
		FrozenBalance:   a.FrozenBalance,
		TotalIncome:     a.TotalIncome,
		TotalWithdrawn:  a.TotalWithdrawn,
		TotalCommission: a.TotalCommission,
		BankAccountName: a.BankAccountName,
		BankAccountNo:   a.BankAccountNo,
		BankName:        a.BankName,
		Status:          int32(a.Status),
		Ctime:           timestamppb.New(a.Ctime),
		Utime:           timestamppb.New(a.Utime),
	}
}

func (s *AccountGRPCServer) toSettlementDTO(r domain.SettlementRecord) *accountv1.SettlementRecord {
	return &accountv1.SettlementRecord{
		Id:               r.ID,
		TenantId:         r.TenantID,
		SettlementNo:     r.SettlementNo,
		OrderNo:          r.OrderNo,
		PaymentNo:        r.PaymentNo,
		OrderAmount:      r.OrderAmount,
		CommissionRate:   r.CommissionRate,
		CommissionAmount: r.CommissionAmount,
		SettlementAmount: r.SettlementAmount,
		Status:           int32(r.Status),
		SettledAt:        r.SettledAt,
		SettleDate:       r.SettleDate,
		Ctime:            timestamppb.New(r.Ctime),
		Utime:            timestamppb.New(r.Utime),
	}
}

func (s *AccountGRPCServer) toWithdrawalDTO(w domain.WithdrawalRecord) *accountv1.WithdrawalRecord {
	return &accountv1.WithdrawalRecord{
		Id:              w.ID,
		TenantId:        w.TenantID,
		WithdrawalNo:    w.WithdrawalNo,
		Amount:          w.Amount,
		Status:          int32(w.Status),
		BankAccountName: w.BankAccountName,
		BankAccountNo:   w.BankAccountNo,
		BankName:        w.BankName,
		RejectReason:    w.RejectReason,
		ReviewedAt:      w.ReviewedAt,
		PaidAt:          w.PaidAt,
		Ctime:           timestamppb.New(w.Ctime),
		Utime:           timestamppb.New(w.Utime),
	}
}

func (s *AccountGRPCServer) toTransactionDTO(t domain.AccountTransaction) *accountv1.AccountTransaction {
	return &accountv1.AccountTransaction{
		Id:            t.ID,
		TenantId:      t.TenantID,
		TransactionNo: t.TransactionNo,
		Type:          int32(t.Type),
		Amount:        t.Amount,
		BalanceAfter:  t.BalanceAfter,
		RefNo:         t.RefNo,
		Remark:        t.Remark,
		Ctime:         timestamppb.New(t.Ctime),
	}
}
