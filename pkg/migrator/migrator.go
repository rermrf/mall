// Package migrator 提供数据库双写迁移支持。
// 用于在不停机的情况下从旧表/库迁移到新表/库。
package migrator

import (
	"context"

	"github.com/rermrf/emo/logger"
	"gorm.io/gorm"
)

// Migrator 双写迁移器
type Migrator[T any] struct {
	src    *gorm.DB
	dst    *gorm.DB
	l      logger.Logger
	entity T
}

// NewMigrator 创建迁移器，src 为源库，dst 为目标库
func NewMigrator[T any](src, dst *gorm.DB, l logger.Logger) *Migrator[T] {
	return &Migrator[T]{src: src, dst: dst, l: l}
}

// FullSync 全量同步：从 src 读取所有记录写入 dst（幂等，通过 upsert）
func (m *Migrator[T]) FullSync(ctx context.Context, batchSize int) error {
	var offset int
	for {
		var batch []T
		err := m.src.WithContext(ctx).Offset(offset).Limit(batchSize).Find(&batch).Error
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		err = m.dst.WithContext(ctx).Save(&batch).Error
		if err != nil {
			m.l.Error("迁移批次写入失败",
				logger.Error(err),
				logger.Int64("offset", int64(offset)),
			)
			return err
		}
		offset += len(batch)
	}
}
