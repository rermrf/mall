package ioc

import (
	"github.com/rermrf/emo/logger"
	"go.uber.org/zap"
)

func InitLogger() logger.Logger {
	l, _ := zap.NewDevelopment()
	return logger.NewZapLogger(l)
}
