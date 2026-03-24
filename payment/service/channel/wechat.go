package channel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"

	"github.com/rermrf/mall/payment/domain"
)

// WechatConfig holds WeChat Pay configuration values needed by the channel layer
type WechatConfig struct {
	AppId          string
	MchId          string
	MchApiV3Key    string
	NotifyUrl      string
	PrivateKeyPath string
	SerialNo       string
}

type WechatChannel struct {
	client    *core.Client
	appId     string
	mchId     string
	apiV3Key  string
	notifyUrl string
	notifier  *notify.Handler
}

func NewWechatChannel(client *core.Client, cfg *WechatConfig) *WechatChannel {
	if client == nil || cfg == nil {
		return nil
	}
	notifier, err := notify.NewRSANotifyHandler(
		cfg.MchApiV3Key,
		verifiers.NewSHA256WithRSAVerifier(downloader.MgrInstance().GetCertificateVisitor(cfg.MchId)),
	)
	if err != nil {
		return nil
	}
	return &WechatChannel{
		client:    client,
		appId:     cfg.AppId,
		mchId:     cfg.MchId,
		apiV3Key:  cfg.MchApiV3Key,
		notifyUrl: cfg.NotifyUrl,
		notifier:  notifier,
	}
}

func (c *WechatChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	if c.client == nil {
		return "", "", fmt.Errorf("微信支付客户端未初始化")
	}

	svc := &native.NativeApiService{Client: c.client}
	resp, _, err := svc.Prepay(ctx, native.PrepayRequest{
		Appid:       core.String(c.appId),
		Mchid:       core.String(c.mchId),
		Description: core.String(fmt.Sprintf("订单 %s", payment.OrderNo)),
		OutTradeNo:  core.String(payment.PaymentNo),
		NotifyUrl:   core.String(c.notifyUrl),
		Amount: &native.Amount{
			Total: core.Int64(payment.Amount),
		},
	})
	if err != nil {
		return "", "", fmt.Errorf("创建微信Native支付失败: %w", err)
	}

	return "", *resp.CodeUrl, nil
}

func (c *WechatChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	if c.client == nil {
		return 0, "", fmt.Errorf("微信支付客户端未初始化")
	}

	svc := &native.NativeApiService{Client: c.client}
	resp, _, err := svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(paymentNo),
		Mchid:      core.String(c.mchId),
	})
	if err != nil {
		return 0, "", fmt.Errorf("查询微信支付状态失败: %w", err)
	}

	status := mapWechatTradeState(*resp.TradeState)
	var tradeNo string
	if resp.TransactionId != nil {
		tradeNo = *resp.TransactionId
	}
	return int32(status), tradeNo, nil
}

func (c *WechatChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("微信支付客户端未初始化")
	}

	refundSvc := &refunddomestic.RefundsApiService{Client: c.client}
	resp, _, err := refundSvc.Create(ctx, refunddomestic.CreateRequest{
		OutTradeNo:  core.String(refund.PaymentNo),
		OutRefundNo: core.String(refund.RefundNo),
		Reason:      core.String("用户退款"),
		Amount: &refunddomestic.AmountReq{
			Refund:   core.Int64(refund.Amount),
			Total:    core.Int64(refund.TotalAmount),
			Currency: core.String("CNY"),
		},
	})
	if err != nil {
		return "", fmt.Errorf("微信退款失败: %w", err)
	}

	return *resp.RefundId, nil
}

func (c *WechatChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	if c.client == nil {
		return 0, "", fmt.Errorf("微信支付客户端未初始化")
	}

	refundSvc := &refunddomestic.RefundsApiService{Client: c.client}
	resp, _, err := refundSvc.QueryByOutRefundNo(ctx, refunddomestic.QueryByOutRefundNoRequest{
		OutRefundNo: core.String(refundNo),
	})
	if err != nil {
		return 0, "", fmt.Errorf("查询微信退款状态失败: %w", err)
	}

	var channelRefundNo string
	if resp.RefundId != nil {
		channelRefundNo = *resp.RefundId
	}

	if resp.Status == nil {
		return int32(domain.RefundStatusRefunding), channelRefundNo, nil
	}

	switch *resp.Status {
	case refunddomestic.STATUS_SUCCESS:
		return int32(domain.RefundStatusRefunded), channelRefundNo, nil
	case refunddomestic.STATUS_PROCESSING:
		return int32(domain.RefundStatusRefunding), channelRefundNo, nil
	case refunddomestic.STATUS_ABNORMAL, refunddomestic.STATUS_CLOSED:
		return int32(domain.RefundStatusFailed), channelRefundNo, nil
	default:
		return int32(domain.RefundStatusRefunding), channelRefundNo, nil
	}
}

func (c *WechatChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	if c.client == nil {
		return "", "", fmt.Errorf("微信支付客户端未初始化")
	}
	if c.notifier == nil {
		return "", "", fmt.Errorf("微信支付通知处理器未初始化")
	}

	body := data["body"]
	if body == "" {
		return "", "", fmt.Errorf("微信支付回调数据为空")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.notifyUrl, io.NopCloser(strings.NewReader(body)))
	if err != nil {
		return "", "", fmt.Errorf("构造微信支付回调请求失败: %w", err)
	}
	req.Header.Set("Wechatpay-Timestamp", data["Wechatpay-Timestamp"])
	req.Header.Set("Wechatpay-Nonce", data["Wechatpay-Nonce"])
	req.Header.Set("Wechatpay-Signature", data["Wechatpay-Signature"])
	req.Header.Set("Wechatpay-Serial", data["Wechatpay-Serial"])
	if data["Wechatpay-Signature-Type"] != "" {
		req.Header.Set("Wechatpay-Signature-Type", data["Wechatpay-Signature-Type"])
	}

	transaction := new(payments.Transaction)
	if _, err := c.notifier.ParseNotifyRequest(ctx, req, transaction); err != nil {
		return "", "", fmt.Errorf("验证微信支付回调失败: %w", err)
	}
	if transaction.TradeState == nil || *transaction.TradeState != "SUCCESS" {
		return "", "", fmt.Errorf("微信交易状态非成功")
	}
	if transaction.OutTradeNo == nil || transaction.TransactionId == nil {
		return "", "", fmt.Errorf("微信支付回调缺少订单标识")
	}

	return *transaction.OutTradeNo, *transaction.TransactionId, nil
}

// DownloadBill implements Reconciler interface
func (c *WechatChannel) DownloadBill(ctx context.Context, billDate string) ([]BillItem, error) {
	if c.client == nil {
		return nil, fmt.Errorf("微信支付客户端未初始化")
	}
	// TODO: implement bill download using /v3/bill/tradebill
	return nil, fmt.Errorf("对账单解析暂未实现")
}

func mapWechatTradeState(state string) domain.PaymentStatus {
	switch state {
	case "SUCCESS":
		return domain.PaymentStatusPaid
	case "NOTPAY", "USERPAYING":
		return domain.PaymentStatusPending
	case "CLOSED", "REVOKED", "PAYERROR":
		return domain.PaymentStatusClosed
	default:
		return domain.PaymentStatusPending
	}
}
