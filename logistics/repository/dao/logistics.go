package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ==================== Models ====================

type FreightTemplate struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	TenantID      int64  `gorm:"not null;index:idx_tenant"`
	Name          string `gorm:"type:varchar(100);not null"`
	ChargeType    int32  `gorm:"not null"` // 1-按重量 2-按件数
	FreeThreshold int64  `gorm:"not null;default:0"`
	Ctime         int64  `gorm:"not null"`
	Utime         int64  `gorm:"not null"`
}

type FreightRule struct {
	ID              int64  `gorm:"primaryKey;autoIncrement"`
	TemplateID      int64  `gorm:"not null;index:idx_template"`
	Regions         string `gorm:"type:text"` // JSON
	FirstUnit       int32  `gorm:"not null"`
	FirstPrice      int64  `gorm:"not null"`
	AdditionalUnit  int32  `gorm:"not null"`
	AdditionalPrice int64  `gorm:"not null"`
}

type Shipment struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	TenantID    int64  `gorm:"not null;index:idx_tenant_status"`
	OrderID     int64  `gorm:"not null;uniqueIndex:uk_order"`
	CarrierCode string `gorm:"type:varchar(20);not null"`
	CarrierName string `gorm:"type:varchar(50);not null"`
	TrackingNo  string `gorm:"type:varchar(50);not null"`
	Status      int32  `gorm:"not null;default:1;index:idx_tenant_status,priority:2"` // 1-待发货 2-已发货 3-运输中 4-已签收
	Ctime       int64  `gorm:"not null"`
	Utime       int64  `gorm:"not null"`
}

type ShipmentTrack struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	ShipmentID  int64  `gorm:"not null;index:idx_shipment"`
	Description string `gorm:"type:varchar(500)"`
	Location    string `gorm:"type:varchar(200)"`
	TrackTime   int64  `gorm:"not null"`
}

// ==================== FreightTemplateDAO ====================

type FreightTemplateDAO interface {
	Insert(ctx context.Context, t FreightTemplate, rules []FreightRule) (FreightTemplate, error)
	Update(ctx context.Context, t FreightTemplate, rules []FreightRule) error
	FindById(ctx context.Context, id int64) (FreightTemplate, []FreightRule, error)
	ListByTenant(ctx context.Context, tenantId int64) ([]FreightTemplate, error)
	Delete(ctx context.Context, id, tenantId int64) error
	ListByTenantWithRules(ctx context.Context, tenantId int64) ([]FreightTemplate, []FreightRule, error)
}

type GORMFreightTemplateDAO struct {
	db *gorm.DB
}

func NewFreightTemplateDAO(db *gorm.DB) FreightTemplateDAO {
	return &GORMFreightTemplateDAO{db: db}
}

func (d *GORMFreightTemplateDAO) Insert(ctx context.Context, t FreightTemplate, rules []FreightRule) (FreightTemplate, error) {
	now := time.Now().UnixMilli()
	t.Ctime = now
	t.Utime = now
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&t).Error; err != nil {
			return err
		}
		for i := range rules {
			rules[i].TemplateID = t.ID
		}
		if len(rules) > 0 {
			return tx.Create(&rules).Error
		}
		return nil
	})
	return t, err
}

func (d *GORMFreightTemplateDAO) Update(ctx context.Context, t FreightTemplate, rules []FreightRule) error {
	t.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND tenant_id = ?", t.ID, t.TenantID).Updates(&t).Error; err != nil {
			return err
		}
		// 删除旧 rules 并重建
		if err := tx.Where("template_id = ?", t.ID).Delete(&FreightRule{}).Error; err != nil {
			return err
		}
		for i := range rules {
			rules[i].TemplateID = t.ID
		}
		if len(rules) > 0 {
			return tx.Create(&rules).Error
		}
		return nil
	})
}

func (d *GORMFreightTemplateDAO) FindById(ctx context.Context, id int64) (FreightTemplate, []FreightRule, error) {
	var t FreightTemplate
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if err != nil {
		return t, nil, err
	}
	var rules []FreightRule
	err = d.db.WithContext(ctx).Where("template_id = ?", id).Find(&rules).Error
	return t, rules, err
}

func (d *GORMFreightTemplateDAO) ListByTenant(ctx context.Context, tenantId int64) ([]FreightTemplate, error) {
	var templates []FreightTemplate
	err := d.db.WithContext(ctx).Where("tenant_id = ?", tenantId).Order("id DESC").Find(&templates).Error
	return templates, err
}

func (d *GORMFreightTemplateDAO) Delete(ctx context.Context, id, tenantId int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND tenant_id = ?", id, tenantId).Delete(&FreightTemplate{}).Error; err != nil {
			return err
		}
		return tx.Where("template_id = ?", id).Delete(&FreightRule{}).Error
	})
}

func (d *GORMFreightTemplateDAO) ListByTenantWithRules(ctx context.Context, tenantId int64) ([]FreightTemplate, []FreightRule, error) {
	var templates []FreightTemplate
	err := d.db.WithContext(ctx).Where("tenant_id = ?", tenantId).Order("id DESC").Find(&templates).Error
	if err != nil {
		return nil, nil, err
	}
	if len(templates) == 0 {
		return templates, nil, nil
	}
	ids := make([]int64, len(templates))
	for i := range templates {
		ids[i] = templates[i].ID
	}
	var rules []FreightRule
	err = d.db.WithContext(ctx).Where("template_id IN ?", ids).Find(&rules).Error
	return templates, rules, err
}

// ==================== ShipmentDAO ====================

type ShipmentDAO interface {
	Insert(ctx context.Context, s Shipment) (Shipment, error)
	FindById(ctx context.Context, id int64) (Shipment, error)
	FindByOrderId(ctx context.Context, orderId int64) (Shipment, error)
	UpdateStatus(ctx context.Context, id int64, status int32) error
}

type GORMShipmentDAO struct {
	db *gorm.DB
}

func NewShipmentDAO(db *gorm.DB) ShipmentDAO {
	return &GORMShipmentDAO{db: db}
}

func (d *GORMShipmentDAO) Insert(ctx context.Context, s Shipment) (Shipment, error) {
	now := time.Now().UnixMilli()
	s.Ctime = now
	s.Utime = now
	err := d.db.WithContext(ctx).Create(&s).Error
	return s, err
}

func (d *GORMShipmentDAO) FindById(ctx context.Context, id int64) (Shipment, error) {
	var s Shipment
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	return s, err
}

func (d *GORMShipmentDAO) FindByOrderId(ctx context.Context, orderId int64) (Shipment, error) {
	var s Shipment
	err := d.db.WithContext(ctx).Where("order_id = ?", orderId).First(&s).Error
	return s, err
}

func (d *GORMShipmentDAO) UpdateStatus(ctx context.Context, id int64, status int32) error {
	return d.db.WithContext(ctx).Model(&Shipment{}).Where("id = ?", id).
		Updates(map[string]any{
			"status": status,
			"utime":  time.Now().UnixMilli(),
		}).Error
}

// ==================== ShipmentTrackDAO ====================

type ShipmentTrackDAO interface {
	Insert(ctx context.Context, t ShipmentTrack) (ShipmentTrack, error)
	ListByShipment(ctx context.Context, shipmentId int64) ([]ShipmentTrack, error)
}

type GORMShipmentTrackDAO struct {
	db *gorm.DB
}

func NewShipmentTrackDAO(db *gorm.DB) ShipmentTrackDAO {
	return &GORMShipmentTrackDAO{db: db}
}

func (d *GORMShipmentTrackDAO) Insert(ctx context.Context, t ShipmentTrack) (ShipmentTrack, error) {
	err := d.db.WithContext(ctx).Create(&t).Error
	return t, err
}

func (d *GORMShipmentTrackDAO) ListByShipment(ctx context.Context, shipmentId int64) ([]ShipmentTrack, error) {
	var tracks []ShipmentTrack
	err := d.db.WithContext(ctx).Where("shipment_id = ?", shipmentId).
		Order("track_time DESC").Find(&tracks).Error
	return tracks, err
}
