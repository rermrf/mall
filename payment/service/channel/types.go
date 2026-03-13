package channel

import (
	"context"

	"github.com/rermrf/mall/payment/domain"
)

// Channel 支付渠道抽象接口
type Channel interface {
	Pay(ctx context.Context, payment domain.PaymentOrder) (channelTradeNo string, payUrl string, err error)
	QueryPayment(ctx context.Context, paymentNo string) (status int32, channelTradeNo string, err error)
	Refund(ctx context.Context, refund domain.RefundRecord) (channelRefundNo string, err error)
	QueryRefund(ctx context.Context, refundNo string) (status int32, channelRefundNo string, err error)
	VerifyNotify(ctx context.Context, data map[string]string) (paymentNo string, channelTradeNo string, err error)
}
