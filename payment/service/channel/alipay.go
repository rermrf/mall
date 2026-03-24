package channel

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/smartwalle/alipay/v3"
	"github.com/spf13/viper"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

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

// downloadAndParseBill downloads the zip file from Alipay, extracts CSV, and parses bill items.
// Alipay ZIP contains 2 CSVs: one with "业务明细" in the filename (detail) and one summary.
// The detail CSV uses GBK encoding.
func downloadAndParseBill(downloadURL string) ([]BillItem, error) {
	// 1. Download ZIP
	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("下载对账单失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取对账单失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载对账单HTTP状态异常: %d", resp.StatusCode)
	}

	// 2. Open ZIP in memory
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("解压对账单失败: %w", err)
	}

	// 3. Find the detail CSV file (contains "业务明细" in filename)
	var detailFile *zip.File
	for _, f := range zipReader.File {
		if strings.Contains(f.Name, "业务明细") {
			detailFile = f
			break
		}
	}
	if detailFile == nil {
		return nil, fmt.Errorf("对账单中未找到业务明细文件")
	}

	// 4. Read and decode GBK → UTF-8
	rc, err := detailFile.Open()
	if err != nil {
		return nil, fmt.Errorf("打开明细文件失败: %w", err)
	}
	defer rc.Close()

	decoder := simplifiedchinese.GBK.NewDecoder()
	reader := csv.NewReader(transform.NewReader(rc, decoder))
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("解析CSV失败: %w", err)
	}

	// 5. Parse records, skip header row and footer/summary rows
	var items []BillItem
	for i, row := range records {
		if i == 0 {
			continue // skip column header
		}
		if len(row) < 12 {
			continue
		}

		// Trim all fields (Alipay adds leading/trailing spaces)
		for j := range row {
			row[j] = strings.TrimSpace(row[j])
		}

		// Skip summary/footer rows (start with '#', contain '总计', or are empty)
		if row[0] == "" || strings.HasPrefix(row[0], "#") || strings.Contains(row[0], "总") {
			continue
		}

		// Map columns:
		//  0: 支付宝交易号  → ChannelTradeNo
		//  1: 商户订单号    → OutTradeNo (payment_no)
		//  5: 完成时间      → PayTime
		// 11: 订单金额      → Amount (yuan → fen)
		// last or 22: 交易状态 → Status
		statusIdx := len(row) - 1
		if statusIdx > 22 {
			statusIdx = 22
		}

		items = append(items, BillItem{
			ChannelTradeNo: row[0],
			OutTradeNo:     row[1],
			Amount:         YuanToFen(row[11]),
			Status:         mapAlipayBillStatus(row[statusIdx]),
			PayTime:        row[5],
		})
	}

	return items, nil
}

// mapAlipayBillStatus converts Chinese trade status from Alipay bill CSV
// to standardised status strings used by the reconciliation engine.
func mapAlipayBillStatus(status string) string {
	status = strings.TrimSpace(status)
	switch status {
	case "交易成功":
		return "TRADE_SUCCESS"
	case "交易关闭":
		return "TRADE_CLOSED"
	case "退款成功":
		return "REFUND_SUCCESS"
	default:
		return status
	}
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
