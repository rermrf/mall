package channel

import (
	"context"
	"fmt"

	"github.com/rermrf/mall/payment/domain"
)

type AlipayChannel struct{}

func NewAlipayChannel() *AlipayChannel {
	return &AlipayChannel{}
}

func (c *AlipayChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	return "", "", fmt.Errorf("支付宝渠道暂未实现")
}

func (c *AlipayChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	return 0, "", fmt.Errorf("支付宝渠道暂未实现")
}

func (c *AlipayChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	return "", fmt.Errorf("支付宝渠道暂未实现")
}

func (c *AlipayChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	return 0, "", fmt.Errorf("支付宝渠道暂未实现")
}

func (c *AlipayChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	return "", "", fmt.Errorf("支付宝渠道暂未实现")
}
