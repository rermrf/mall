package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/logistics/domain"
	"github.com/rermrf/mall/logistics/events"
	"github.com/rermrf/mall/logistics/repository"
	"github.com/rermrf/mall/logistics/repository/cache"
)

var (
	ErrTemplateNotFound = errors.New("运费模板不存在")
	ErrShipmentNotFound = errors.New("物流单不存在")
	ErrNoFreightRule    = errors.New("无匹配运费规则")
)

type FreightCalcItem struct {
	SkuID    int64
	Quantity int32
	Weight   int32 // 克
}

type LogisticsService interface {
	// 运费模板
	CreateFreightTemplate(ctx context.Context, t domain.FreightTemplate) (domain.FreightTemplate, error)
	UpdateFreightTemplate(ctx context.Context, t domain.FreightTemplate) error
	GetFreightTemplate(ctx context.Context, id int64) (domain.FreightTemplate, error)
	ListFreightTemplates(ctx context.Context, tenantId int64) ([]domain.FreightTemplate, error)
	DeleteFreightTemplate(ctx context.Context, id, tenantId int64) error
	// 运费计算
	CalculateFreight(ctx context.Context, tenantId int64, province string, items []FreightCalcItem) (freight int64, freeShipping bool, err error)
	// 物流
	CreateShipment(ctx context.Context, s domain.Shipment) (domain.Shipment, error)
	GetShipment(ctx context.Context, id int64) (domain.Shipment, error)
	GetShipmentByOrder(ctx context.Context, orderId int64) (domain.Shipment, error)
	AddTrack(ctx context.Context, shipmentId int64, description, location string, trackTime int64) error
}

type logisticsService struct {
	repo     repository.LogisticsRepository
	producer events.Producer
	l        logger.Logger
}

func NewLogisticsService(repo repository.LogisticsRepository, producer events.Producer, l logger.Logger) LogisticsService {
	return &logisticsService{repo: repo, producer: producer, l: l}
}

// ==================== 运费模板 ====================

func (s *logisticsService) CreateFreightTemplate(ctx context.Context, t domain.FreightTemplate) (domain.FreightTemplate, error) {
	return s.repo.CreateFreightTemplate(ctx, t)
}

func (s *logisticsService) UpdateFreightTemplate(ctx context.Context, t domain.FreightTemplate) error {
	return s.repo.UpdateFreightTemplate(ctx, t)
}

func (s *logisticsService) GetFreightTemplate(ctx context.Context, id int64) (domain.FreightTemplate, error) {
	return s.repo.GetFreightTemplate(ctx, id)
}

func (s *logisticsService) ListFreightTemplates(ctx context.Context, tenantId int64) ([]domain.FreightTemplate, error) {
	return s.repo.ListFreightTemplates(ctx, tenantId)
}

func (s *logisticsService) DeleteFreightTemplate(ctx context.Context, id, tenantId int64) error {
	return s.repo.DeleteFreightTemplate(ctx, id, tenantId)
}

// ==================== 运费计算 ====================

func (s *logisticsService) CalculateFreight(ctx context.Context, tenantId int64, province string, items []FreightCalcItem) (int64, bool, error) {
	templates, err := s.repo.GetTemplatesForCalculation(ctx, tenantId)
	if err != nil {
		return 0, false, err
	}
	// 没有运费模板，默认包邮
	if len(templates) == 0 {
		return 0, true, nil
	}

	// 汇总数量和重量
	var totalQty int32
	var totalWeight int32
	for _, item := range items {
		totalQty += item.Quantity
		totalWeight += item.Weight * item.Quantity
	}

	for _, tpl := range templates {
		// 匹配省份规则
		rule, found := matchRule(tpl.Rules, province)
		if !found {
			continue
		}
		// 根据计费方式计算运费
		var freight int64
		switch tpl.ChargeType {
		case 1: // 按件数
			freight = calcByPiece(rule, totalQty)
		case 2: // 按重量
			freight = calcByWeight(rule, totalWeight)
		default:
			continue
		}
		return freight, freight == 0, nil
	}

	// 没有匹配规则，默认包邮
	return 0, true, nil
}

// matchRule 匹配省份运费规则，支持 "*" 通配符作为默认规则
func matchRule(rules []cache.CachedFreightRule, province string) (cache.CachedFreightRule, bool) {
	var defaultRule cache.CachedFreightRule
	hasDefault := false

	for _, rule := range rules {
		var regions []string
		if err := json.Unmarshal([]byte(rule.Regions), &regions); err != nil {
			continue
		}
		for _, r := range regions {
			if r == province {
				return rule, true
			}
			if r == "*" {
				defaultRule = rule
				hasDefault = true
			}
		}
	}

	if hasDefault {
		return defaultRule, true
	}
	return cache.CachedFreightRule{}, false
}

// calcByPiece 按件数计算运费: 首件价格 + ceil((数量 - 首件数) / 续件数) * 续件价格
func calcByPiece(rule cache.CachedFreightRule, totalQty int32) int64 {
	if totalQty <= rule.FirstUnit {
		return rule.FirstPrice
	}
	extra := totalQty - rule.FirstUnit
	if rule.AdditionalUnit <= 0 {
		return rule.FirstPrice
	}
	additionalSteps := int64(math.Ceil(float64(extra) / float64(rule.AdditionalUnit)))
	return rule.FirstPrice + additionalSteps*rule.AdditionalPrice
}

// calcByWeight 按重量计算运费: 首重价格 + ceil((重量 - 首重) / 续重) * 续重价格
func calcByWeight(rule cache.CachedFreightRule, totalWeight int32) int64 {
	if totalWeight <= rule.FirstUnit {
		return rule.FirstPrice
	}
	extra := totalWeight - rule.FirstUnit
	if rule.AdditionalUnit <= 0 {
		return rule.FirstPrice
	}
	additionalSteps := int64(math.Ceil(float64(extra) / float64(rule.AdditionalUnit)))
	return rule.FirstPrice + additionalSteps*rule.AdditionalPrice
}

// ==================== 物流 ====================

func (s *logisticsService) CreateShipment(ctx context.Context, shipment domain.Shipment) (domain.Shipment, error) {
	shipment.Status = 1 // 已发货
	result, err := s.repo.CreateShipment(ctx, shipment)
	if err != nil {
		return domain.Shipment{}, err
	}

	// 发送物流发货事件（失败仅记录日志，不影响主流程）
	evt := events.OrderShippedEvent{
		OrderId:     result.OrderID,
		TenantId:    result.TenantID,
		CarrierCode: result.CarrierCode,
		CarrierName: result.CarrierName,
		TrackingNo:  result.TrackingNo,
	}
	if produceErr := s.producer.ProduceOrderShipped(ctx, evt); produceErr != nil {
		s.l.Error("发送物流发货事件失败", logger.Error(produceErr))
	}

	return result, nil
}

func (s *logisticsService) GetShipment(ctx context.Context, id int64) (domain.Shipment, error) {
	return s.repo.GetShipment(ctx, id)
}

func (s *logisticsService) GetShipmentByOrder(ctx context.Context, orderId int64) (domain.Shipment, error) {
	return s.repo.GetShipmentByOrder(ctx, orderId)
}

func (s *logisticsService) AddTrack(ctx context.Context, shipmentId int64, description, location string, trackTime int64) error {
	track := domain.ShipmentTrack{
		ShipmentID:  shipmentId,
		Description: description,
		Location:    location,
		TrackTime:   trackTime,
	}
	_, err := s.repo.AddTrack(ctx, track)
	return err
}
