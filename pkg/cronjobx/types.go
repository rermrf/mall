// Package cronjobx 重新导出 github.com/rermrf/emo/cronjobx，
// 提供定时任务构建器。
package cronjobx

import "github.com/rermrf/emo/cronjobx"

type (
	CronJobBuilder = cronjobx.CronJobBuilder
	Job            = cronjobx.Job
)

var NewCronJobBuilder = cronjobx.NewCronJobBuilder
