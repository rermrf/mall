package channel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/pkg/snowflake"
)

type MockChannel struct {
	node *snowflake.Node
}

func NewMockChannel(node *snowflake.Node) *MockChannel {
	return &MockChannel{node: node}
}

func (c *MockChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	channelTradeNo := fmt.Sprintf("MOCK_%d", c.node.Generate())
	return channelTradeNo, "", nil
}

func (c *MockChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	return int32(domain.PaymentStatusPaid), fmt.Sprintf("MOCK_QUERY_%s", paymentNo), nil
}

func (c *MockChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	channelRefundNo := fmt.Sprintf("MOCK_REFUND_%d", c.node.Generate())
	return channelRefundNo, nil
}

func (c *MockChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	return int32(domain.RefundStatusRefunded), fmt.Sprintf("MOCK_REFUND_QUERY_%s", refundNo), nil
}

func (c *MockChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	paymentNo, ok := data["payment_no"]
	if !ok {
		return "", "", fmt.Errorf("缺少 payment_no")
	}
	channelTradeNo, ok := data["channel_trade_no"]
	if !ok {
		return "", "", fmt.Errorf("缺少 channel_trade_no")
	}
	return paymentNo, channelTradeNo, nil
}

// BuildMockNotifyBody 构造 mock 回调报文（用于测试）
func BuildMockNotifyBody(paymentNo, channelTradeNo string) string {
	data, _ := json.Marshal(map[string]string{
		"payment_no":       paymentNo,
		"channel_trade_no": channelTradeNo,
	})
	return string(data)
}
