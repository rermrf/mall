package channel

import (
	"context"
	"fmt"

	"github.com/rermrf/mall/payment/domain"
)

type WechatChannel struct{}

func NewWechatChannel() *WechatChannel {
	return &WechatChannel{}
}

func (c *WechatChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	return "", "", fmt.Errorf("微信支付渠道暂未实现")
}

func (c *WechatChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	return 0, "", fmt.Errorf("微信支付渠道暂未实现")
}

func (c *WechatChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	return "", fmt.Errorf("微信支付渠道暂未实现")
}

func (c *WechatChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	return 0, "", fmt.Errorf("微信支付渠道暂未实现")
}

func (c *WechatChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	return "", "", fmt.Errorf("微信支付渠道暂未实现")
}
