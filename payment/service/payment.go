package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rermrf/emo/idempotent"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/payment/events"
	"github.com/rermrf/mall/payment/repository"
	"github.com/rermrf/mall/payment/service/channel"
	"github.com/rermrf/mall/pkg/snowflake"
)

type PaymentService interface {
	CreatePayment(ctx context.Context, tenantId int64, orderId int64, orderNo, ch string, amount int64) (string, string, error)
	GetPayment(ctx context.Context, paymentNo string) (domain.PaymentOrder, error)
	HandleNotify(ctx context.Context, ch string, notifyBody string) (bool, error)
	ClosePayment(ctx context.Context, paymentNo string) error
	Refund(ctx context.Context, paymentNo string, amount int64, reason string) (string, error)
	GetRefund(ctx context.Context, refundNo string) (domain.RefundRecord, error)
	ListPayments(ctx context.Context, tenantId int64, status, page, pageSize int32) ([]domain.PaymentOrder, int64, error)
}

type paymentService struct {
	repo           repository.PaymentRepository
	producer       events.Producer
	idempotencySvc idempotent.IdempotencyService
	node           *snowflake.Node
	channels       map[string]channel.Channel
	l              logger.Logger
}

func NewPaymentService(
	repo repository.PaymentRepository,
	producer events.Producer,
	idempotencySvc idempotent.IdempotencyService,
	node *snowflake.Node,
	mockCh *channel.MockChannel,
	alipayCh *channel.AlipayChannel,
	wechatCh *channel.WechatChannel,
	l logger.Logger,
) PaymentService {
	channels := map[string]channel.Channel{
		"mock": mockCh,
	}
	if alipayCh != nil {
		channels["alipay"] = alipayCh
	}
	if wechatCh != nil {
		channels["wechat"] = wechatCh
	}
	return &paymentService{
		repo:           repo,
		producer:       producer,
		idempotencySvc: idempotencySvc,
		node:           node,
		channels:       channels,
		l:              l,
	}
}

func (s *paymentService) getChannel(ch string) (channel.Channel, error) {
	c, ok := s.channels[ch]
	if !ok {
		return nil, fmt.Errorf("不支持的支付渠道: %s", ch)
	}
	return c, nil
}

func (s *paymentService) CreatePayment(ctx context.Context, tenantId int64, orderId int64, orderNo, ch string, amount int64) (string, string, error) {
	if amount <= 0 {
		return "", "", fmt.Errorf("支付金额必须大于0")
	}
	// Check if payment already exists for this order (idempotent)
	existing, findErr := s.repo.FindByOrderNo(ctx, orderNo)
	if findErr == nil && existing.ID > 0 {
		if existing.Status == domain.PaymentStatusPending || existing.Status == domain.PaymentStatusPaying {
			s.l.Info("支付单已存在，返回已有支付单",
				logger.String("orderNo", orderNo),
				logger.String("paymentNo", existing.PaymentNo))
			return existing.PaymentNo, "", nil
		}
	}
	c, err := s.getChannel(ch)
	if err != nil {
		return "", "", err
	}
	paymentNo := fmt.Sprintf("P%d", s.node.Generate())
	payment := domain.PaymentOrder{
		TenantID:   tenantId,
		PaymentNo:  paymentNo,
		OrderID:    orderId,
		OrderNo:    orderNo,
		Channel:    ch,
		Amount:     amount,
		Status:     domain.PaymentStatusPending,
		ExpireTime: time.Now().Add(30 * time.Minute).UnixMilli(),
	}
	payment, err = s.repo.CreatePayment(ctx, payment)
	if err != nil {
		return "", "", fmt.Errorf("创建支付单失败: %w", err)
	}
	// 调用渠道发起支付
	channelTradeNo, payUrl, err := c.Pay(ctx, payment)
	if err != nil {
		return "", "", fmt.Errorf("渠道发起支付失败: %w", err)
	}
	// 更新渠道交易号
	if channelTradeNo != "" {
		if err := s.repo.UpdateStatus(ctx, paymentNo, domain.PaymentStatusPending, domain.PaymentStatusPending, map[string]any{
			"channel_trade_no": channelTradeNo,
		}); err != nil {
			s.l.Error("保存渠道交易号失败", logger.String("paymentNo", paymentNo), logger.Error(err))
		}
	}
	return paymentNo, payUrl, nil
}

func (s *paymentService) GetPayment(ctx context.Context, paymentNo string) (domain.PaymentOrder, error) {
	return s.repo.FindByPaymentNo(ctx, paymentNo)
}

func (s *paymentService) HandleNotify(ctx context.Context, ch string, notifyBody string) (bool, error) {
	c, err := s.getChannel(ch)
	if err != nil {
		return false, err
	}
	// 解析回调数据
	var data map[string]string
	if err := json.Unmarshal([]byte(notifyBody), &data); err != nil {
		return false, fmt.Errorf("解析回调报文失败: %w", err)
	}
	// 验证回调
	paymentNo, channelTradeNo, err := c.VerifyNotify(ctx, data)
	if err != nil {
		return false, fmt.Errorf("验证回调失败: %w", err)
	}
	// 布隆过滤器幂等检查
	bloomKey := fmt.Sprintf("payment:notify:%s", paymentNo)
	exists, err := s.idempotencySvc.Exists(ctx, bloomKey)
	if err != nil {
		s.l.Error("幂等检查失败", logger.Error(err))
	}
	if exists {
		payment, dbErr := s.repo.FindByPaymentNo(ctx, paymentNo)
		if dbErr == nil && payment.Status == domain.PaymentStatusPaid {
			return true, nil
		}
	}
	// 查询支付单
	payment, err := s.repo.FindByPaymentNo(ctx, paymentNo)
	if err != nil {
		return false, fmt.Errorf("支付单不存在: %w", err)
	}
	if payment.Status == domain.PaymentStatusPaid {
		return true, nil
	}
	if payment.Status != domain.PaymentStatusPending && payment.Status != domain.PaymentStatusPaying {
		return false, fmt.Errorf("支付单状态不允许回调: %d", payment.Status)
	}
	// 更新状态为已支付
	now := time.Now().UnixMilli()
	err = s.repo.UpdateStatus(ctx, paymentNo, payment.Status, domain.PaymentStatusPaid, map[string]any{
		"channel_trade_no": channelTradeNo,
		"pay_time":         now,
	})
	if err != nil {
		// CAS failed — check if another goroutine already set it to paid
		current, queryErr := s.repo.FindByPaymentNo(ctx, paymentNo)
		if queryErr == nil && current.Status == domain.PaymentStatusPaid {
			return true, nil // idempotent success
		}
		return false, fmt.Errorf("更新支付状态失败: %w", err)
	}
	// 发送 order_paid 事件
	if produceErr := s.producer.ProduceOrderPaid(ctx, events.OrderPaidEvent{
		OrderNo:   payment.OrderNo,
		PaymentNo: paymentNo,
		PaidAt:    now,
	}); produceErr != nil {
		s.l.Error("发送 order_paid 事件失败，需要人工补偿",
			logger.String("paymentNo", paymentNo),
			logger.String("orderNo", payment.OrderNo),
			logger.Error(produceErr))
		// TODO: 实现补偿任务扫描已支付但未发送事件的支付单
	}
	return true, nil
}

func (s *paymentService) ClosePayment(ctx context.Context, paymentNo string) error {
	payment, err := s.repo.FindByPaymentNo(ctx, paymentNo)
	if err != nil {
		return fmt.Errorf("支付单不存在: %w", err)
	}
	if payment.Status != domain.PaymentStatusPending && payment.Status != domain.PaymentStatusPaying {
		return fmt.Errorf("当前状态不允许关闭: %d", payment.Status)
	}
	return s.repo.UpdateStatus(ctx, paymentNo, payment.Status, domain.PaymentStatusClosed, map[string]any{})
}

func (s *paymentService) Refund(ctx context.Context, paymentNo string, amount int64, reason string) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("退款金额必须大于0")
	}
	payment, err := s.repo.FindByPaymentNo(ctx, paymentNo)
	if err != nil {
		return "", fmt.Errorf("支付单不存在: %w", err)
	}
	if payment.Status != domain.PaymentStatusPaid {
		return "", fmt.Errorf("当前状态不允许退款: %d", payment.Status)
	}
	if amount > payment.Amount {
		return "", fmt.Errorf("退款金额超出支付金额")
	}
	c, err := s.getChannel(payment.Channel)
	if err != nil {
		return "", err
	}
	refundNo := fmt.Sprintf("R%d", s.node.Generate())
	refund := domain.RefundRecord{
		TenantID:    payment.TenantID,
		PaymentNo:   paymentNo,
		RefundNo:    refundNo,
		Channel:     payment.Channel,
		Amount:      amount,
		TotalAmount: payment.Amount,
		Status:      domain.RefundStatusRefunding,
	}
	if err := s.repo.CreateRefund(ctx, refund); err != nil {
		return "", fmt.Errorf("创建退款记录失败: %w", err)
	}
	channelRefundNo, err := c.Refund(ctx, refund)
	if err != nil {
		if updateErr := s.repo.UpdateRefundStatus(ctx, refundNo, domain.RefundStatusFailed, map[string]any{}); updateErr != nil {
			s.l.Error("更新退款记录状态为失败时出错", logger.String("refundNo", refundNo), logger.Error(updateErr))
		}
		return "", fmt.Errorf("渠道退款失败: %w", err)
	}
	if err := s.repo.UpdateRefundStatus(ctx, refundNo, domain.RefundStatusRefunded, map[string]any{
		"channel_refund_no": channelRefundNo,
	}); err != nil {
		s.l.Error("更新退款记录状态失败", logger.String("refundNo", refundNo), logger.Error(err))
		return "", fmt.Errorf("更新退款记录状态失败: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, paymentNo, domain.PaymentStatusPaid, domain.PaymentStatusRefunded, map[string]any{}); err != nil {
		s.l.Error("更新支付单状态为已退款失败", logger.String("paymentNo", paymentNo), logger.Error(err))
	}
	return refundNo, nil
}

func (s *paymentService) GetRefund(ctx context.Context, refundNo string) (domain.RefundRecord, error) {
	return s.repo.FindRefundByNo(ctx, refundNo)
}

func (s *paymentService) ListPayments(ctx context.Context, tenantId int64, status, page, pageSize int32) ([]domain.PaymentOrder, int64, error) {
	return s.repo.ListPayments(ctx, tenantId, status, page, pageSize)
}
