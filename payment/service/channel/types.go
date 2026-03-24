package channel

import (
	"context"
	"strconv"
	"strings"

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

// Reconciler is an optional interface for channels that support bill download
type Reconciler interface {
	DownloadBill(ctx context.Context, billDate string) ([]BillItem, error)
}

type BillItem struct {
	ChannelTradeNo string
	OutTradeNo     string // payment_no
	Amount         int64  // fen
	Status         string // TRADE_SUCCESS / TRADE_CLOSED etc.
	PayTime        string
}

// YuanToFen converts a yuan string (e.g. "99.00") to fen (int64).
// Used by both Alipay and WeChat bill parsers.
func YuanToFen(yuan string) int64 {
	yuan = strings.TrimSpace(yuan)
	if yuan == "" {
		return 0
	}
	// Handle negative amounts
	negative := false
	if strings.HasPrefix(yuan, "-") {
		negative = true
		yuan = yuan[1:]
	}
	parts := strings.Split(yuan, ".")
	intPart, _ := strconv.ParseInt(parts[0], 10, 64)
	result := intPart * 100
	if len(parts) == 2 {
		dec := parts[1]
		if len(dec) > 2 {
			dec = dec[:2]
		}
		for len(dec) < 2 {
			dec += "0"
		}
		d, _ := strconv.ParseInt(dec, 10, 64)
		result += d
	}
	if negative {
		result = -result
	}
	return result
}
