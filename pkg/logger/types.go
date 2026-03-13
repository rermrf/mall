// Package logger 重新导出 github.com/rermrf/emo/logger，
// 项目中统一使用本包引用，避免直接依赖外部路径。
package logger

import "github.com/rermrf/emo/logger"

// 重新导出类型
type (
	Logger    = logger.Logger
	Field     = logger.Field
	NopLogger = logger.NopLogger
	ZapLogger = logger.ZapLogger
)

// 重新导出构造函数
var (
	NewNopLogger = logger.NewNopLogger
	NewZapLogger = logger.NewZapLogger
)

// 重新导出 Field 构造器
var (
	String = logger.String
	Int64  = logger.Int64
	Int32  = logger.Int32
	Bool   = logger.Bool
	Error  = logger.Error
	Any    = logger.Any
)
