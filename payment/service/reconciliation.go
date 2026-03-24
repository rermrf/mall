package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/payment/repository"
	"github.com/rermrf/mall/payment/repository/dao"
	"github.com/rermrf/mall/payment/service/channel"
	"github.com/rermrf/mall/pkg/snowflake"
)

// ReconciliationService 对账服务接口
type ReconciliationService interface {
	RunReconciliation(ctx context.Context, ch string, billDate string) (int64, error)
	ListBatches(ctx context.Context, page, pageSize int32) ([]dao.ReconciliationBatchModel, int64, error)
	GetBatchDetail(ctx context.Context, batchId int64, page, pageSize int32) (dao.ReconciliationBatchModel, []dao.ReconciliationDetailModel, int64, error)
}

type reconciliationService struct {
	reconDAO dao.ReconciliationDAO
	repo     repository.PaymentRepository
	channels map[string]channel.Channel
	node     *snowflake.Node
	l        logger.Logger
}

func NewReconciliationService(
	reconDAO dao.ReconciliationDAO,
	repo repository.PaymentRepository,
	mockCh *channel.MockChannel,
	alipayCh *channel.AlipayChannel,
	wechatCh *channel.WechatChannel,
	node *snowflake.Node,
	l logger.Logger,
) ReconciliationService {
	channels := map[string]channel.Channel{
		"mock": mockCh,
	}
	if alipayCh != nil {
		channels["alipay"] = alipayCh
	}
	if wechatCh != nil {
		channels["wechat"] = wechatCh
	}
	return &reconciliationService{
		reconDAO: reconDAO,
		repo:     repo,
		channels: channels,
		node:     node,
		l:        l,
	}
}

func (s *reconciliationService) RunReconciliation(ctx context.Context, ch string, billDate string) (int64, error) {
	// 0. 去重检查：同渠道同日期已有成功的对账批次则跳过
	existing, err := s.reconDAO.FindBatchByChannelAndDate(ctx, ch, billDate)
	if err == nil && existing.ID > 0 {
		s.l.Info("该渠道和日期已完成对账，跳过",
			logger.String("channel", ch),
			logger.String("billDate", billDate),
			logger.String("batchNo", existing.BatchNo))
		return existing.ID, nil
	}

	// 1. 生成批次号并创建批次记录
	batchNo := fmt.Sprintf("R%d", s.node.Generate())
	batch, err := s.reconDAO.CreateBatch(ctx, dao.ReconciliationBatchModel{
		BatchNo:  batchNo,
		Channel:  ch,
		BillDate: billDate,
		Status:   1, // 处理中
	})
	if err != nil {
		return 0, fmt.Errorf("创建对账批次失败: %w", err)
	}

	// 2. 获取渠道并检查是否支持对账
	c, ok := s.channels[ch]
	if !ok {
		s.failBatch(ctx, batch.ID, "不支持的支付渠道: "+ch)
		return batch.ID, fmt.Errorf("不支持的支付渠道: %s", ch)
	}
	reconciler, ok := c.(channel.Reconciler)
	if !ok {
		s.failBatch(ctx, batch.ID, "该渠道不支持对账")
		return batch.ID, fmt.Errorf("渠道 %s 不支持对账", ch)
	}

	// 3. 下载渠道账单
	billItems, err := reconciler.DownloadBill(ctx, billDate)
	if err != nil {
		s.failBatch(ctx, batch.ID, "下载对账单失败: "+err.Error())
		return batch.ID, fmt.Errorf("下载对账单失败: %w", err)
	}

	// 4. 查询本地当日支付记录
	loc, _ := time.LoadLocation("Asia/Shanghai")
	billDateParsed, err := time.ParseInLocation("2006-01-02", billDate, loc)
	if err != nil {
		s.failBatch(ctx, batch.ID, "对账日期格式错误: "+err.Error())
		return batch.ID, fmt.Errorf("对账日期格式错误: %w", err)
	}
	startTime := billDateParsed.UnixMilli()
	endTime := billDateParsed.Add(24 * time.Hour).UnixMilli()

	localPayments, err := s.repo.ListPaymentsByDateAndChannel(ctx, ch, startTime, endTime)
	if err != nil {
		s.failBatch(ctx, batch.ID, "查询本地支付记录失败: "+err.Error())
		return batch.ID, fmt.Errorf("查询本地支付记录失败: %w", err)
	}

	// 5. 构建映射进行对比
	// 渠道账单 map: channel_trade_no → BillItem
	channelMap := make(map[string]channel.BillItem, len(billItems))
	var channelTotalAmount int64
	for _, item := range billItems {
		channelMap[item.ChannelTradeNo] = item
		channelTotalAmount += item.Amount
	}

	// 本地支付记录 map: channel_trade_no → PaymentOrder
	type localRecord struct {
		PaymentNo      string
		ChannelTradeNo string
		Amount         int64
		Status         int32
	}
	localMap := make(map[string]localRecord, len(localPayments))
	var localTotalAmount int64
	var details []dao.ReconciliationDetailModel
	for _, p := range localPayments {
		localTotalAmount += p.Amount
		if p.ChannelTradeNo == "" {
			s.l.Warn("支付记录缺少渠道交易号，记为对账差异", logger.String("paymentNo", p.PaymentNo))
			details = append(details, dao.ReconciliationDetailModel{
				BatchId:       batch.ID,
				PaymentNo:     p.PaymentNo,
				Type:          1,
				LocalAmount:   p.Amount,
				LocalStatus:   int32(p.Status),
				ChannelAmount: 0,
				Remark:        "本地支付记录缺少渠道交易号，无法与渠道账单匹配",
			})
			continue
		}
		localMap[p.ChannelTradeNo] = localRecord{
			PaymentNo:      p.PaymentNo,
			ChannelTradeNo: p.ChannelTradeNo,
			Amount:         p.Amount,
			Status:         int32(p.Status),
		}
	}

	// 6. 对比并生成差异明细
	matchCount := 0

	// 本地记录逐条与渠道对比
	for tradeNo, local := range localMap {
		chItem, exists := channelMap[tradeNo]
		if !exists {
			// 本地有，渠道无 → 类型1: 本地多
			details = append(details, dao.ReconciliationDetailModel{
				BatchId:        batch.ID,
				PaymentNo:      local.PaymentNo,
				ChannelTradeNo: tradeNo,
				Type:           1,
				LocalAmount:    local.Amount,
				ChannelAmount:  0,
				LocalStatus:    local.Status,
				Remark:         "本地存在但渠道账单中未找到",
			})
			continue
		}

		// 双方都有，对比金额和状态
		if local.Amount != chItem.Amount {
			// 金额不一致 → 类型3
			details = append(details, dao.ReconciliationDetailModel{
				BatchId:        batch.ID,
				PaymentNo:      local.PaymentNo,
				ChannelTradeNo: tradeNo,
				Type:           3,
				LocalAmount:    local.Amount,
				ChannelAmount:  chItem.Amount,
				LocalStatus:    local.Status,
				ChannelStatus:  chItem.Status,
				Remark:         fmt.Sprintf("金额不一致: 本地=%d, 渠道=%d", local.Amount, chItem.Amount),
			})
		} else if chItem.Status != "TRADE_SUCCESS" && chItem.Status != "TRADE_FINISHED" {
			// 状态不一致 → 类型4
			details = append(details, dao.ReconciliationDetailModel{
				BatchId:        batch.ID,
				PaymentNo:      local.PaymentNo,
				ChannelTradeNo: tradeNo,
				Type:           4,
				LocalAmount:    local.Amount,
				ChannelAmount:  chItem.Amount,
				LocalStatus:    local.Status,
				ChannelStatus:  chItem.Status,
				Remark:         fmt.Sprintf("状态不一致: 渠道状态=%s", chItem.Status),
			})
		} else {
			matchCount++
		}
	}

	// 渠道有，本地无 → 类型2: 渠道多
	for tradeNo, chItem := range channelMap {
		if _, exists := localMap[tradeNo]; !exists {
			details = append(details, dao.ReconciliationDetailModel{
				BatchId:        batch.ID,
				PaymentNo:      chItem.OutTradeNo,
				ChannelTradeNo: tradeNo,
				Type:           2,
				LocalAmount:    0,
				ChannelAmount:  chItem.Amount,
				ChannelStatus:  chItem.Status,
				Remark:         "渠道存在但本地未找到对应记录",
			})
		}
	}

	// 7. 保存差异明细
	if len(details) > 0 {
		if err := s.reconDAO.CreateDetails(ctx, details); err != nil {
			s.l.Error("保存对账差异明细失败", logger.Error(err))
			_ = s.reconDAO.UpdateBatch(ctx, batch.ID, map[string]any{"status": 3, "error_msg": "保存差异明细失败: " + err.Error()})
			return batch.ID, fmt.Errorf("保存对账差异明细失败: %w", err)
		}
	}

	// 8. 更新批次统计
	updateErr := s.reconDAO.UpdateBatch(ctx, batch.ID, map[string]any{
		"status":         2, // 已完成
		"total_channel":  int32(len(billItems)),
		"total_local":    int32(len(localPayments)),
		"total_match":    int32(matchCount),
		"total_mismatch": int32(len(details)),
		"channel_amount": channelTotalAmount,
		"local_amount":   localTotalAmount,
	})
	if updateErr != nil {
		s.l.Error("更新对账批次统计失败", logger.Error(updateErr))
	}

	s.l.Info("对账完成",
		logger.String("batchNo", batchNo),
		logger.String("channel", ch),
		logger.String("billDate", billDate),
		logger.Int32("totalChannel", int32(len(billItems))),
		logger.Int32("totalLocal", int32(len(localPayments))),
		logger.Int32("totalMatch", int32(matchCount)),
		logger.Int32("totalMismatch", int32(len(details))),
	)

	return batch.ID, nil
}

func (s *reconciliationService) ListBatches(ctx context.Context, page, pageSize int32) ([]dao.ReconciliationBatchModel, int64, error) {
	offset := int((page - 1) * pageSize)
	return s.reconDAO.ListBatches(ctx, offset, int(pageSize))
}

func (s *reconciliationService) GetBatchDetail(ctx context.Context, batchId int64, page, pageSize int32) (dao.ReconciliationBatchModel, []dao.ReconciliationDetailModel, int64, error) {
	batch, err := s.reconDAO.GetBatch(ctx, batchId)
	if err != nil {
		return dao.ReconciliationBatchModel{}, nil, 0, err
	}
	offset := int((page - 1) * pageSize)
	details, total, err := s.reconDAO.ListDetails(ctx, batchId, offset, int(pageSize))
	if err != nil {
		return batch, nil, 0, err
	}
	return batch, details, total, nil
}

func (s *reconciliationService) failBatch(ctx context.Context, batchId int64, errMsg string) {
	_ = s.reconDAO.UpdateBatch(ctx, batchId, map[string]any{
		"status":    3, // 失败
		"error_msg": errMsg,
	})
}
