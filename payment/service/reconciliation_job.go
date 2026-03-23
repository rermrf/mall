package service

import (
	"context"
	"time"

	"github.com/rermrf/emo/logger"
)

// ReconciliationJob 对账定时任务
type ReconciliationJob struct {
	svc      ReconciliationService
	l        logger.Logger
	channels []string
	runHour  int
}

func NewReconciliationJob(svc ReconciliationService, l logger.Logger) *ReconciliationJob {
	return &ReconciliationJob{
		svc:      svc,
		l:        l,
		channels: []string{}, // enable "alipay" once DownloadBill is implemented
		runHour:  1, // 每天凌晨1点执行
	}
}

// Start 启动定时对账任务
func (j *ReconciliationJob) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				j.l.Info("对账定时任务已停止")
				return
			case t := <-ticker.C:
				if t.Hour() == j.runHour && t.Minute() == 0 {
					j.runAll(ctx)
				}
			}
		}
	}()
	j.l.Info("对账定时任务已启动", logger.Int32("runHour", int32(j.runHour)))
}

func (j *ReconciliationJob) runAll(ctx context.Context) {
	// 对账前一天的账单
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	for _, ch := range j.channels {
		j.l.Info("开始执行定时对账",
			logger.String("channel", ch),
			logger.String("billDate", yesterday),
		)
		_, err := j.svc.RunReconciliation(ctx, ch, yesterday)
		if err != nil {
			j.l.Error("定时对账失败",
				logger.String("channel", ch),
				logger.String("billDate", yesterday),
				logger.Error(err),
			)
		}
	}
}
