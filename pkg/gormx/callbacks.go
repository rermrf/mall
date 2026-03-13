package gormx

import (
	"github.com/rermrf/emo/logger"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Callbacks 注册 GORM 回调，用于日志和可观测性扩展
type Callbacks struct {
	L logger.Logger
}

// LogSlowSQL 注册慢查询日志回调（阈值由 GORM 全局配置 SlowThreshold 控制）
func (c *Callbacks) LogSlowSQL(db *gorm.DB) {
	// GORM 自带慢查询日志，通过 gorm.Config.Logger 配置
	// 这里作为示例提供一个快捷封装
	db.Config.Logger = gormlogger.Default.LogMode(gormlogger.Warn)
}
