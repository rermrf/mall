package channel

import (
	"context"
	"fmt"
	"net/url"

	"github.com/smartwalle/alipay/v3"
	"github.com/spf13/viper"

	"github.com/rermrf/mall/payment/domain"
)

type AlipayChannel struct {
	client    *alipay.Client
	notifyURL string
	returnURL string
}

func NewAlipayChannel(client *alipay.Client) *AlipayChannel {
	return &AlipayChannel{
		client:    client,
		notifyURL: viper.GetString("alipay.notifyUrl"),
		returnURL: viper.GetString("alipay.returnUrl"),
	}
}

func (c *AlipayChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	if c.client == nil {
		return "", "", fmt.Errorf("支付宝客户端未初始化")
	}

	// 金额从分转为元（字符串，保留两位小数）
	amount := fmt.Sprintf("%d.%02d", payment.Amount/100, payment.Amount%100)

	param := alipay.TradeWapPay{
		Trade: alipay.Trade{
			NotifyURL:   c.notifyURL,
			ReturnURL:   c.returnURL,
			Subject:     fmt.Sprintf("订单 %s", payment.OrderNo),
			OutTradeNo:  payment.PaymentNo,
			TotalAmount: amount,
			ProductCode: "QUICK_WAP_WAY",
		},
	}

	payURL, err := c.client.TradeWapPay(param)
	if err != nil {
		return "", "", fmt.Errorf("创建支付宝H5支付失败: %w", err)
	}

	return "", payURL.String(), nil
}

func (c *AlipayChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	if c.client == nil {
		return 0, "", fmt.Errorf("支付宝客户端未初始化")
	}

	param := alipay.TradeQuery{
		OutTradeNo: paymentNo,
	}

	var result alipay.TradeQueryRsp
	if err := c.client.Request(ctx, param, &result); err != nil {
		return 0, "", fmt.Errorf("查询支付宝支付状态失败: %w", err)
	}

	if result.Code != alipay.CodeSuccess {
		return 0, "", fmt.Errorf("支付宝查询失败: %s - %s", result.SubCode, result.SubMsg)
	}

	status := mapAlipayTradeStatus(result.TradeStatus)
	return int32(status), result.TradeNo, nil
}

func (c *AlipayChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("支付宝客户端未初始化")
	}

	amount := fmt.Sprintf("%d.%02d", refund.Amount/100, refund.Amount%100)

	param := alipay.TradeRefund{
		OutTradeNo:   refund.PaymentNo,
		RefundAmount: amount,
		RefundReason: "用户退款",
		OutRequestNo: refund.RefundNo,
	}

	var result alipay.TradeRefundRsp
	if err := c.client.Request(ctx, param, &result); err != nil {
		return "", fmt.Errorf("支付宝退款失败: %w", err)
	}

	if result.Code != alipay.CodeSuccess {
		return "", fmt.Errorf("支付宝退款失败: %s - %s", result.SubCode, result.SubMsg)
	}

	return result.TradeNo, nil
}

func (c *AlipayChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	if c.client == nil {
		return 0, "", fmt.Errorf("支付宝客户端未初始化")
	}

	param := alipay.TradeFastPayRefundQuery{
		OutRequestNo: refundNo,
	}

	var result alipay.TradeFastPayRefundQueryRsp
	if err := c.client.Request(ctx, param, &result); err != nil {
		return 0, "", fmt.Errorf("查询支付宝退款状态失败: %w", err)
	}

	if result.Code != alipay.CodeSuccess {
		return 0, "", fmt.Errorf("支付宝退款查询失败: %s - %s", result.SubCode, result.SubMsg)
	}

	if result.RefundStatus == "REFUND_SUCCESS" {
		return int32(domain.RefundStatusRefunded), result.TradeNo, nil
	}
	return int32(domain.RefundStatusRefunding), result.TradeNo, nil
}

func (c *AlipayChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	if c.client == nil {
		return "", "", fmt.Errorf("支付宝客户端未初始化")
	}

	// Convert map to url.Values for SDK verification
	values := url.Values{}
	for k, v := range data {
		values.Set(k, v)
	}

	notification, err := c.client.DecodeNotification(ctx, values)
	if err != nil {
		return "", "", fmt.Errorf("支付宝回调验签失败: %w", err)
	}

	if notification.TradeStatus != "TRADE_SUCCESS" && notification.TradeStatus != "TRADE_FINISHED" {
		return "", "", fmt.Errorf("支付宝交易状态非成功: %s", notification.TradeStatus)
	}

	return notification.OutTradeNo, notification.TradeNo, nil
}

// DownloadBill implements Reconciler interface
func (c *AlipayChannel) DownloadBill(ctx context.Context, billDate string) ([]BillItem, error) {
	if c.client == nil {
		return nil, fmt.Errorf("支付宝客户端未初始化")
	}

	param := alipay.BillDownloadURLQuery{
		BillType: "trade",
		BillDate: billDate,
	}

	var result alipay.BillDownloadURLQueryRsp
	if err := c.client.Request(ctx, param, &result); err != nil {
		return nil, fmt.Errorf("查询支付宝对账单下载地址失败: %w", err)
	}

	if result.Code != alipay.CodeSuccess {
		return nil, fmt.Errorf("获取对账单失败: %s - %s", result.SubCode, result.SubMsg)
	}

	// Download and parse the CSV bill file
	return downloadAndParseBill(result.BillDownloadURL)
}

// downloadAndParseBill downloads the zip file from Alipay, extracts CSV, and parses bill items
func downloadAndParseBill(downloadURL string) ([]BillItem, error) {
	// TODO: implement CSV download and parsing
	// 1. HTTP GET downloadURL -> zip file
	// 2. Unzip -> find the detail CSV file (not summary)
	// 3. Parse each row into BillItem
	// For now, return empty - will be implemented in reconciliation task
	return nil, fmt.Errorf("对账单解析暂未实现")
}

func mapAlipayTradeStatus(status alipay.TradeStatus) domain.PaymentStatus {
	switch status {
	case alipay.TradeStatusWaitBuyerPay:
		return domain.PaymentStatusPending
	case alipay.TradeStatusClosed:
		return domain.PaymentStatusClosed
	case alipay.TradeStatusSuccess:
		return domain.PaymentStatusPaid
	case alipay.TradeStatusFinished:
		return domain.PaymentStatusPaid
	default:
		return domain.PaymentStatusPending
	}
}
