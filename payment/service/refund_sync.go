package service

import (
	"context"

	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/payment/domain"
)

type RefundSyncer interface {
	SyncRefund(ctx context.Context, payment domain.PaymentOrder, refund domain.RefundRecord) error
}

type noopRefundSyncer struct{}

func (noopRefundSyncer) SyncRefund(ctx context.Context, payment domain.PaymentOrder, refund domain.RefundRecord) error {
	return nil
}

type orderRefundSyncer struct {
	client orderv1.OrderServiceClient
}

func NewOrderRefundSyncer(client orderv1.OrderServiceClient) RefundSyncer {
	if client == nil {
		return noopRefundSyncer{}
	}
	return &orderRefundSyncer{client: client}
}

func (s *orderRefundSyncer) SyncRefund(ctx context.Context, payment domain.PaymentOrder, refund domain.RefundRecord) error {
	_, err := s.client.HandlePaymentRefunded(ctx, &orderv1.HandlePaymentRefundedRequest{
		OrderNo:   payment.OrderNo,
		PaymentNo: payment.PaymentNo,
		RefundNo:  refund.RefundNo,
		Amount:    refund.Amount,
	})
	return err
}
