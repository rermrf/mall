package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ReconciliationBatchModel struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	BatchNo       string `gorm:"type:varchar(64);uniqueIndex:uk_batch_no"`
	Channel       string `gorm:"type:varchar(32)"`
	BillDate      string `gorm:"type:varchar(10);index:idx_bill_date"`
	Status        int32  // 1=处理中 2=已完成 3=失败
	TotalChannel  int32
	TotalLocal    int32
	TotalMatch    int32
	TotalMismatch int32
	ChannelAmount int64
	LocalAmount   int64
	ErrorMsg      string `gorm:"type:varchar(512)"`
	Ctime         int64
	Utime         int64
}

func (ReconciliationBatchModel) TableName() string { return "reconciliation_batches" }

type ReconciliationDetailModel struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	BatchId        int64  `gorm:"index:idx_batch_id"`
	PaymentNo      string `gorm:"type:varchar(64)"`
	ChannelTradeNo string `gorm:"type:varchar(128)"`
	Type           int32  // 1=本地多 2=渠道多 3=金额不一致 4=状态不一致
	LocalAmount    int64
	ChannelAmount  int64
	LocalStatus    int32
	ChannelStatus  string `gorm:"type:varchar(32)"`
	Handled        bool
	Remark         string `gorm:"type:varchar(512)"`
	Ctime          int64
}

func (ReconciliationDetailModel) TableName() string { return "reconciliation_details" }

type ReconciliationDAO interface {
	CreateBatch(ctx context.Context, batch ReconciliationBatchModel) (ReconciliationBatchModel, error)
	UpdateBatch(ctx context.Context, id int64, updates map[string]any) error
	FindBatchByChannelAndDate(ctx context.Context, channel, billDate string) (ReconciliationBatchModel, error)
	ListBatches(ctx context.Context, offset, limit int) ([]ReconciliationBatchModel, int64, error)
	GetBatch(ctx context.Context, id int64) (ReconciliationBatchModel, error)
	CreateDetails(ctx context.Context, details []ReconciliationDetailModel) error
	ListDetails(ctx context.Context, batchId int64, offset, limit int) ([]ReconciliationDetailModel, int64, error)
}

type GORMReconciliationDAO struct {
	db *gorm.DB
}

func NewReconciliationDAO(db *gorm.DB) ReconciliationDAO {
	return &GORMReconciliationDAO{db: db}
}

func (d *GORMReconciliationDAO) CreateBatch(ctx context.Context, batch ReconciliationBatchModel) (ReconciliationBatchModel, error) {
	now := time.Now().UnixMilli()
	batch.Ctime = now
	batch.Utime = now
	err := d.db.WithContext(ctx).Create(&batch).Error
	return batch, err
}

func (d *GORMReconciliationDAO) UpdateBatch(ctx context.Context, id int64, updates map[string]any) error {
	updates["utime"] = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Model(&ReconciliationBatchModel{}).Where("id = ?", id).Updates(updates).Error
}

func (d *GORMReconciliationDAO) FindBatchByChannelAndDate(ctx context.Context, channel, billDate string) (ReconciliationBatchModel, error) {
	var batch ReconciliationBatchModel
	err := d.db.WithContext(ctx).Where("channel = ? AND bill_date = ? AND status = ?", channel, billDate, 2).First(&batch).Error
	return batch, err
}

func (d *GORMReconciliationDAO) ListBatches(ctx context.Context, offset, limit int) ([]ReconciliationBatchModel, int64, error) {
	var batches []ReconciliationBatchModel
	var total int64
	query := d.db.WithContext(ctx).Model(&ReconciliationBatchModel{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&batches).Error
	return batches, total, err
}

func (d *GORMReconciliationDAO) GetBatch(ctx context.Context, id int64) (ReconciliationBatchModel, error) {
	var batch ReconciliationBatchModel
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&batch).Error
	return batch, err
}

func (d *GORMReconciliationDAO) CreateDetails(ctx context.Context, details []ReconciliationDetailModel) error {
	if len(details) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	for i := range details {
		details[i].Ctime = now
	}
	return d.db.WithContext(ctx).CreateInBatches(details, 100).Error
}

func (d *GORMReconciliationDAO) ListDetails(ctx context.Context, batchId int64, offset, limit int) ([]ReconciliationDetailModel, int64, error) {
	var details []ReconciliationDetailModel
	var total int64
	query := d.db.WithContext(ctx).Model(&ReconciliationDetailModel{}).Where("batch_id = ?", batchId)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id ASC").Offset(offset).Limit(limit).Find(&details).Error
	return details, total, err
}
