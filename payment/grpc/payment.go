package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/payment/service"
)

type PaymentGRPCServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	svc service.PaymentService
}

func NewPaymentGRPCServer(svc service.PaymentService) *PaymentGRPCServer {
	return &PaymentGRPCServer{svc: svc}
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
