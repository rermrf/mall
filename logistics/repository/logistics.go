package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rermrf/mall/logistics/domain"
	"github.com/rermrf/mall/logistics/repository/cache"
	"github.com/rermrf/mall/logistics/repository/dao"
)

// ==================== Interface ====================

type LogisticsRepository interface {
	// 运费模板
	CreateFreightTemplate(ctx context.Context, t domain.FreightTemplate) (domain.FreightTemplate, error)
	UpdateFreightTemplate(ctx context.Context, t domain.FreightTemplate) error
	GetFreightTemplate(ctx context.Context, id int64) (domain.FreightTemplate, error)
	ListFreightTemplates(ctx context.Context, tenantId int64) ([]domain.FreightTemplate, error)
	DeleteFreightTemplate(ctx context.Context, id, tenantId int64) error
	GetTemplatesForCalculation(ctx context.Context, tenantId int64) ([]cache.CachedTemplate, error)
	// 物流单
	CreateShipment(ctx context.Context, s domain.Shipment) (domain.Shipment, error)
	GetShipment(ctx context.Context, id int64) (domain.Shipment, error)
	GetShipmentByOrder(ctx context.Context, orderId int64) (domain.Shipment, error)
	// 轨迹
	AddTrack(ctx context.Context, t domain.ShipmentTrack) (domain.ShipmentTrack, error)
	UpdateShipmentStatus(ctx context.Context, shipmentId int64, status int32) error
}

// ==================== Implementation ====================

type logisticsRepository struct {
	templateDAO dao.FreightTemplateDAO
	shipmentDAO dao.ShipmentDAO
	trackDAO    dao.ShipmentTrackDAO
	cache       cache.LogisticsCache
}

func NewLogisticsRepository(
	templateDAO dao.FreightTemplateDAO,
	shipmentDAO dao.ShipmentDAO,
	trackDAO dao.ShipmentTrackDAO,
	c cache.LogisticsCache,
) LogisticsRepository {
	return &logisticsRepository{
		templateDAO: templateDAO,
		shipmentDAO: shipmentDAO,
		trackDAO:    trackDAO,
		cache:       c,
	}
}

// ==================== 运费模板 ====================

func (r *logisticsRepository) CreateFreightTemplate(ctx context.Context, t domain.FreightTemplate) (domain.FreightTemplate, error) {
	dt, rules := r.templateToDAO(t)
	dt, err := r.templateDAO.Insert(ctx, dt, rules)
	if err != nil {
		return domain.FreightTemplate{}, err
	}
	// 失效缓存
	_ = r.cache.DeleteTemplates(ctx, t.TenantID)
	return r.templateToDomain(dt, rules), nil
}

func (r *logisticsRepository) UpdateFreightTemplate(ctx context.Context, t domain.FreightTemplate) error {
	dt, rules := r.templateToDAO(t)
	err := r.templateDAO.Update(ctx, dt, rules)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteTemplates(ctx, t.TenantID)
	return nil
}

func (r *logisticsRepository) GetFreightTemplate(ctx context.Context, id int64) (domain.FreightTemplate, error) {
	t, rules, err := r.templateDAO.FindById(ctx, id)
	if err != nil {
		return domain.FreightTemplate{}, err
	}
	return r.templateToDomain(t, rules), nil
}

func (r *logisticsRepository) ListFreightTemplates(ctx context.Context, tenantId int64) ([]domain.FreightTemplate, error) {
	templates, err := r.templateDAO.ListByTenant(ctx, tenantId)
	if err != nil {
		return nil, err
	}
	result := make([]domain.FreightTemplate, 0, len(templates))
	for _, t := range templates {
		result = append(result, r.templateToDomain(t, nil))
	}
	return result, nil
}

func (r *logisticsRepository) DeleteFreightTemplate(ctx context.Context, id, tenantId int64) error {
	err := r.templateDAO.Delete(ctx, id, tenantId)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteTemplates(ctx, tenantId)
	return nil
}

func (r *logisticsRepository) GetTemplatesForCalculation(ctx context.Context, tenantId int64) ([]cache.CachedTemplate, error) {
	// cache-aside: try cache first
	cached, err := r.cache.GetTemplates(ctx, tenantId)
	if err == nil {
		return cached, nil
	}
	// cache miss (including redis.Nil), fall back to DB
	if err != redis.Nil {
		// log unexpected error but proceed to DB
	}
	templates, rules, err := r.templateDAO.ListByTenantWithRules(ctx, tenantId)
	if err != nil {
		return nil, err
	}
	// build rule map: templateID -> []rule
	ruleMap := make(map[int64][]dao.FreightRule)
	for _, rule := range rules {
		ruleMap[rule.TemplateID] = append(ruleMap[rule.TemplateID], rule)
	}
	// build CachedTemplate slice
	result := make([]cache.CachedTemplate, 0, len(templates))
	for _, t := range templates {
		ct := cache.CachedTemplate{
			ID:            t.ID,
			TenantID:      t.TenantID,
			Name:          t.Name,
			ChargeType:    t.ChargeType,
			FreeThreshold: t.FreeThreshold,
		}
		for _, rule := range ruleMap[t.ID] {
			ct.Rules = append(ct.Rules, cache.CachedFreightRule{
				ID:              rule.ID,
				TemplateID:      rule.TemplateID,
				Regions:         rule.Regions,
				FirstUnit:       rule.FirstUnit,
				FirstPrice:      rule.FirstPrice,
				AdditionalUnit:  rule.AdditionalUnit,
				AdditionalPrice: rule.AdditionalPrice,
			})
		}
		result = append(result, ct)
	}
	// write back to cache (best-effort)
	_ = r.cache.SetTemplates(ctx, tenantId, result)
	return result, nil
}

// ==================== 物流单 ====================

func (r *logisticsRepository) CreateShipment(ctx context.Context, s domain.Shipment) (domain.Shipment, error) {
	ds := r.shipmentToDAO(s)
	ds, err := r.shipmentDAO.Insert(ctx, ds)
	if err != nil {
		return domain.Shipment{}, err
	}
	return r.shipmentToDomain(ds, nil), nil
}

func (r *logisticsRepository) GetShipment(ctx context.Context, id int64) (domain.Shipment, error) {
	ds, err := r.shipmentDAO.FindById(ctx, id)
	if err != nil {
		return domain.Shipment{}, err
	}
	tracks, err := r.trackDAO.ListByShipment(ctx, id)
	if err != nil {
		return domain.Shipment{}, err
	}
	return r.shipmentToDomain(ds, tracks), nil
}

func (r *logisticsRepository) GetShipmentByOrder(ctx context.Context, orderId int64) (domain.Shipment, error) {
	ds, err := r.shipmentDAO.FindByOrderId(ctx, orderId)
	if err != nil {
		return domain.Shipment{}, err
	}
	tracks, err := r.trackDAO.ListByShipment(ctx, ds.ID)
	if err != nil {
		return domain.Shipment{}, err
	}
	return r.shipmentToDomain(ds, tracks), nil
}

// ==================== 轨迹 ====================

func (r *logisticsRepository) AddTrack(ctx context.Context, t domain.ShipmentTrack) (domain.ShipmentTrack, error) {
	dt := r.trackToDAO(t)
	dt, err := r.trackDAO.Insert(ctx, dt)
	if err != nil {
		return domain.ShipmentTrack{}, err
	}
	return r.trackToDomain(dt), nil
}

func (r *logisticsRepository) UpdateShipmentStatus(ctx context.Context, shipmentId int64, status int32) error {
	return r.shipmentDAO.UpdateStatus(ctx, shipmentId, status)
}

// ==================== Converters ====================

func (r *logisticsRepository) templateToDAO(t domain.FreightTemplate) (dao.FreightTemplate, []dao.FreightRule) {
	dt := dao.FreightTemplate{
		ID:            t.ID,
		TenantID:      t.TenantID,
		Name:          t.Name,
		ChargeType:    t.ChargeType,
		FreeThreshold: t.FreeThreshold,
	}
	rules := r.rulesToDAO(t.Rules)
	return dt, rules
}

func (r *logisticsRepository) templateToDomain(t dao.FreightTemplate, rules []dao.FreightRule) domain.FreightTemplate {
	domainRules := make([]domain.FreightRule, 0, len(rules))
	for _, rule := range rules {
		domainRules = append(domainRules, r.ruleToDomain(rule))
	}
	return domain.FreightTemplate{
		ID:            t.ID,
		TenantID:      t.TenantID,
		Name:          t.Name,
		ChargeType:    t.ChargeType,
		FreeThreshold: t.FreeThreshold,
		Rules:         domainRules,
		Ctime:         time.UnixMilli(t.Ctime),
		Utime:         time.UnixMilli(t.Utime),
	}
}

func (r *logisticsRepository) rulesToDAO(rules []domain.FreightRule) []dao.FreightRule {
	result := make([]dao.FreightRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, dao.FreightRule{
			ID:              rule.ID,
			TemplateID:      rule.TemplateID,
			Regions:         rule.Regions,
			FirstUnit:       rule.FirstUnit,
			FirstPrice:      rule.FirstPrice,
			AdditionalUnit:  rule.AdditionalUnit,
			AdditionalPrice: rule.AdditionalPrice,
		})
	}
	return result
}

func (r *logisticsRepository) ruleToDomain(rule dao.FreightRule) domain.FreightRule {
	return domain.FreightRule{
		ID:              rule.ID,
		TemplateID:      rule.TemplateID,
		Regions:         rule.Regions,
		FirstUnit:       rule.FirstUnit,
		FirstPrice:      rule.FirstPrice,
		AdditionalUnit:  rule.AdditionalUnit,
		AdditionalPrice: rule.AdditionalPrice,
	}
}

func (r *logisticsRepository) shipmentToDAO(s domain.Shipment) dao.Shipment {
	return dao.Shipment{
		ID:          s.ID,
		TenantID:    s.TenantID,
		OrderID:     s.OrderID,
		CarrierCode: s.CarrierCode,
		CarrierName: s.CarrierName,
		TrackingNo:  s.TrackingNo,
		Status:      s.Status,
	}
}

func (r *logisticsRepository) shipmentToDomain(s dao.Shipment, tracks []dao.ShipmentTrack) domain.Shipment {
	domainTracks := make([]domain.ShipmentTrack, 0, len(tracks))
	for _, t := range tracks {
		domainTracks = append(domainTracks, r.trackToDomain(t))
	}
	return domain.Shipment{
		ID:          s.ID,
		TenantID:    s.TenantID,
		OrderID:     s.OrderID,
		CarrierCode: s.CarrierCode,
		CarrierName: s.CarrierName,
		TrackingNo:  s.TrackingNo,
		Status:      s.Status,
		Tracks:      domainTracks,
		Ctime:       time.UnixMilli(s.Ctime),
		Utime:       time.UnixMilli(s.Utime),
	}
}

func (r *logisticsRepository) trackToDAO(t domain.ShipmentTrack) dao.ShipmentTrack {
	return dao.ShipmentTrack{
		ID:          t.ID,
		ShipmentID:  t.ShipmentID,
		Description: t.Description,
		Location:    t.Location,
		TrackTime:   t.TrackTime,
	}
}

func (r *logisticsRepository) trackToDomain(t dao.ShipmentTrack) domain.ShipmentTrack {
	return domain.ShipmentTrack{
		ID:          t.ID,
		ShipmentID:  t.ShipmentID,
		Description: t.Description,
		Location:    t.Location,
		TrackTime:   t.TrackTime,
	}
}
