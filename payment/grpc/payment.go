package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/payment/service"
)

type PaymentGRPCServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	svc     service.PaymentService
	reconSvc service.ReconciliationService
}

func NewPaymentGRPCServer(svc service.PaymentService, reconSvc service.ReconciliationService) *PaymentGRPCServer {
	return &PaymentGRPCServer{svc: svc, reconSvc: reconSvc}
}

func (s *PaymentGRPCServer) Register(server *grpc.Server) {
	paymentv1.RegisterPaymentServiceServer(server, s)
}

func (s *PaymentGRPCServer) CreatePayment(ctx context.Context, req *paymentv1.CreatePaymentRequest) (*paymentv1.CreatePaymentResponse, error) {
	paymentNo, payUrl, err := s.svc.CreatePayment(ctx, req.GetTenantId(), req.GetOrderId(), req.GetOrderNo(), req.GetChannel(), req.GetAmount())
	if err != nil {
		return nil, err
	}
	return &paymentv1.CreatePaymentResponse{PaymentNo: paymentNo, PayUrl: payUrl}, nil
}

func (s *PaymentGRPCServer) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.GetPaymentResponse, error) {
	payment, err := s.svc.GetPayment(ctx, req.GetPaymentNo())
	if err != nil {
		return nil, err
	}
	return &paymentv1.GetPaymentResponse{Payment: s.toPaymentDTO(payment)}, nil
}

func (s *PaymentGRPCServer) HandleNotify(ctx context.Context, req *paymentv1.HandleNotifyRequest) (*paymentv1.HandleNotifyResponse, error) {
	success, err := s.svc.HandleNotify(ctx, req.GetChannel(), req.GetNotifyBody())
	if err != nil {
		return nil, err
	}
	return &paymentv1.HandleNotifyResponse{Success: success}, nil
}

func (s *PaymentGRPCServer) ClosePayment(ctx context.Context, req *paymentv1.ClosePaymentRequest) (*paymentv1.ClosePaymentResponse, error) {
	err := s.svc.ClosePayment(ctx, req.GetPaymentNo())
	if err != nil {
		return nil, err
	}
	return &paymentv1.ClosePaymentResponse{}, nil
}

func (s *PaymentGRPCServer) CloseOrderPayments(ctx context.Context, req *paymentv1.CloseOrderPaymentsRequest) (*paymentv1.CloseOrderPaymentsResponse, error) {
	if err := s.svc.CloseOrderPayments(ctx, req.GetOrderNo()); err != nil {
		return nil, err
	}
	return &paymentv1.CloseOrderPaymentsResponse{}, nil
}

func (s *PaymentGRPCServer) Refund(ctx context.Context, req *paymentv1.RefundRequest) (*paymentv1.RefundResponse, error) {
	refundNo, err := s.svc.Refund(ctx, req.GetPaymentNo(), req.GetAmount(), req.GetReason())
	if err != nil {
		return nil, err
	}
	return &paymentv1.RefundResponse{RefundNo: refundNo}, nil
}

func (s *PaymentGRPCServer) GetRefund(ctx context.Context, req *paymentv1.GetRefundRequest) (*paymentv1.GetRefundResponse, error) {
	refund, err := s.svc.GetRefund(ctx, req.GetRefundNo())
	if err != nil {
		return nil, err
	}
	return &paymentv1.GetRefundResponse{Refund: s.toRefundDTO(refund)}, nil
}

func (s *PaymentGRPCServer) ListPayments(ctx context.Context, req *paymentv1.ListPaymentsRequest) (*paymentv1.ListPaymentsResponse, error) {
	payments, total, err := s.svc.ListPayments(ctx, req.GetTenantId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*paymentv1.PaymentOrder, 0, len(payments))
	for _, p := range payments {
		dtos = append(dtos, s.toPaymentDTO(p))
	}
	return &paymentv1.ListPaymentsResponse{Payments: dtos, Total: total}, nil
}

func (s *PaymentGRPCServer) toPaymentDTO(p domain.PaymentOrder) *paymentv1.PaymentOrder {
	return &paymentv1.PaymentOrder{
		Id:             p.ID,
		TenantId:       p.TenantID,
		PaymentNo:      p.PaymentNo,
		OrderId:        p.OrderID,
		OrderNo:        p.OrderNo,
		Channel:        p.Channel,
		Amount:         p.Amount,
		Status:         int32(p.Status),
		ChannelTradeNo: p.ChannelTradeNo,
		PayTime:        p.PayTime,
		ExpireTime:     p.ExpireTime,
		NotifyUrl:      p.NotifyUrl,
		Ctime:          timestamppb.New(p.Ctime),
		Utime:          timestamppb.New(p.Utime),
	}
}

func (s *PaymentGRPCServer) toRefundDTO(r domain.RefundRecord) *paymentv1.RefundRecord {
	return &paymentv1.RefundRecord{
		Id:              r.ID,
		TenantId:        r.TenantID,
		PaymentNo:       r.PaymentNo,
		RefundNo:        r.RefundNo,
		Channel:         r.Channel,
		Amount:          r.Amount,
		Status:          int32(r.Status),
		ChannelRefundNo: r.ChannelRefundNo,
		Ctime:           timestamppb.New(r.Ctime),
	}
}

// ==================== Reconciliation RPCs ====================

func (s *PaymentGRPCServer) RunReconciliation(ctx context.Context, req *paymentv1.RunReconciliationRequest) (*paymentv1.RunReconciliationResponse, error) {
	batchId, err := s.reconSvc.RunReconciliation(ctx, req.GetChannel(), req.GetBillDate())
	if err != nil {
		return nil, err
	}
	return &paymentv1.RunReconciliationResponse{BatchId: batchId}, nil
}

func (s *PaymentGRPCServer) ListReconciliationBatches(ctx context.Context, req *paymentv1.ListReconciliationBatchesRequest) (*paymentv1.ListReconciliationBatchesResponse, error) {
	batches, total, err := s.reconSvc.ListBatches(ctx, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*paymentv1.ReconciliationBatch, 0, len(batches))
	for _, b := range batches {
		dtos = append(dtos, &paymentv1.ReconciliationBatch{
			Id:            b.ID,
			BatchNo:       b.BatchNo,
			Channel:       b.Channel,
			BillDate:      b.BillDate,
			Status:        b.Status,
			TotalChannel:  b.TotalChannel,
			TotalLocal:    b.TotalLocal,
			TotalMatch:    b.TotalMatch,
			TotalMismatch: b.TotalMismatch,
			ChannelAmount: b.ChannelAmount,
			LocalAmount:   b.LocalAmount,
			ErrorMsg:      b.ErrorMsg,
			Ctime:         timestamppb.New(time.UnixMilli(b.Ctime)),
		})
	}
	return &paymentv1.ListReconciliationBatchesResponse{Batches: dtos, Total: total}, nil
}

func (s *PaymentGRPCServer) GetReconciliationBatchDetail(ctx context.Context, req *paymentv1.GetReconciliationBatchDetailRequest) (*paymentv1.GetReconciliationBatchDetailResponse, error) {
	batch, details, total, err := s.reconSvc.GetBatchDetail(ctx, req.GetBatchId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}

	batchDTO := &paymentv1.ReconciliationBatch{
		Id:            batch.ID,
		BatchNo:       batch.BatchNo,
		Channel:       batch.Channel,
		BillDate:      batch.BillDate,
		Status:        batch.Status,
		TotalChannel:  batch.TotalChannel,
		TotalLocal:    batch.TotalLocal,
		TotalMatch:    batch.TotalMatch,
		TotalMismatch: batch.TotalMismatch,
		ChannelAmount: batch.ChannelAmount,
		LocalAmount:   batch.LocalAmount,
		ErrorMsg:      batch.ErrorMsg,
		Ctime:         timestamppb.New(time.UnixMilli(batch.Ctime)),
	}

	detailDTOs := make([]*paymentv1.ReconciliationDetail, 0, len(details))
	for _, d := range details {
		detailDTOs = append(detailDTOs, &paymentv1.ReconciliationDetail{
			Id:             d.ID,
			BatchId:        d.BatchId,
			PaymentNo:      d.PaymentNo,
			ChannelTradeNo: d.ChannelTradeNo,
			Type:           d.Type,
			LocalAmount:    d.LocalAmount,
			ChannelAmount:  d.ChannelAmount,
			LocalStatus:    d.LocalStatus,
			ChannelStatus:  d.ChannelStatus,
			Handled:        d.Handled,
			Remark:         d.Remark,
		})
	}

	return &paymentv1.GetReconciliationBatchDetailResponse{
		Batch:   batchDTO,
		Details: detailDTOs,
		Total:   total,
	}, nil
}
