package service

import (
	"context"
	"time"

	"github.com/rermrf/emo/logger"
)

type SettlementJob struct {
	svc AccountService
	l   logger.Logger
}

func NewSettlementJob(svc AccountService, l logger.Logger) *SettlementJob {
	return &SettlementJob{svc: svc, l: l}
}

func (j *SettlementJob) Start() {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		// 启动时立即执行一次
		j.run()
		for range ticker.C {
			j.run()
		}
	}()
}

func (j *SettlementJob) run() {
	today := time.Now().Format("2006-01-02")
	j.l.Info("开始执行每日结算任务", logger.String("settleDate", today))
	count, amount, err := j.svc.ExecuteSettlement(context.Background(), today)
	if err != nil {
		j.l.Error("每日结算任务执行失败", logger.Error(err))
		return
	}
	j.l.Info("每日结算任务执行完成",
		logger.Int32("settledCount", count),
		logger.Int64("settledAmount", amount))
}
