package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	logisticsv1 "github.com/rermrf/mall/api/proto/gen/logistics/v1"
	"github.com/rermrf/mall/logistics/domain"
	"github.com/rermrf/mall/logistics/service"
)

type LogisticsGRPCServer struct {
	logisticsv1.UnimplementedLogisticsServiceServer
	svc service.LogisticsService
}

func NewLogisticsGRPCServer(svc service.LogisticsService) *LogisticsGRPCServer {
	return &LogisticsGRPCServer{svc: svc}
}

func (s *LogisticsGRPCServer) Register(server *grpc.Server) {
	logisticsv1.RegisterLogisticsServiceServer(server, s)
}

// ==================== 运费模板 ====================

func (s *LogisticsGRPCServer) CreateFreightTemplate(ctx context.Context, req *logisticsv1.CreateFreightTemplateRequest) (*logisticsv1.CreateFreightTemplateResponse, error) {
	t := req.GetTemplate()
	rules := make([]domain.FreightRule, 0, len(t.GetRules()))
	for _, r := range t.GetRules() {
		rules = append(rules, domain.FreightRule{
			Regions: r.GetRegions(), FirstUnit: r.GetFirstUnit(), FirstPrice: r.GetFirstPrice(),
			AdditionalUnit: r.GetAdditionalUnit(), AdditionalPrice: r.GetAdditionalPrice(),
		})
	}
	template, err := s.svc.CreateFreightTemplate(ctx, domain.FreightTemplate{
		TenantID: t.GetTenantId(), Name: t.GetName(),
		ChargeType: t.GetChargeType(), FreeThreshold: t.GetFreeThreshold(),
		Rules: rules,
	})
	if err != nil {
		return nil, err
	}
	return &logisticsv1.CreateFreightTemplateResponse{Id: template.ID}, nil
}

func (s *LogisticsGRPCServer) UpdateFreightTemplate(ctx context.Context, req *logisticsv1.UpdateFreightTemplateRequest) (*logisticsv1.UpdateFreightTemplateResponse, error) {
	t := req.GetTemplate()
	rules := make([]domain.FreightRule, 0, len(t.GetRules()))
	for _, r := range t.GetRules() {
		rules = append(rules, domain.FreightRule{
			Regions: r.GetRegions(), FirstUnit: r.GetFirstUnit(), FirstPrice: r.GetFirstPrice(),
			AdditionalUnit: r.GetAdditionalUnit(), AdditionalPrice: r.GetAdditionalPrice(),
		})
	}
	err := s.svc.UpdateFreightTemplate(ctx, domain.FreightTemplate{
		ID: t.GetId(), TenantID: t.GetTenantId(), Name: t.GetName(),
		ChargeType: t.GetChargeType(), FreeThreshold: t.GetFreeThreshold(),
		Rules: rules,
	})
	if err != nil {
		return nil, err
	}
	return &logisticsv1.UpdateFreightTemplateResponse{}, nil
}

func (s *LogisticsGRPCServer) GetFreightTemplate(ctx context.Context, req *logisticsv1.GetFreightTemplateRequest) (*logisticsv1.GetFreightTemplateResponse, error) {
	t, err := s.svc.GetFreightTemplate(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &logisticsv1.GetFreightTemplateResponse{Template: toFreightTemplateDTO(t)}, nil
}

func (s *LogisticsGRPCServer) ListFreightTemplates(ctx context.Context, req *logisticsv1.ListFreightTemplatesRequest) (*logisticsv1.ListFreightTemplatesResponse, error) {
	templates, err := s.svc.ListFreightTemplates(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	pbTemplates := make([]*logisticsv1.FreightTemplate, 0, len(templates))
	for _, t := range templates {
		pbTemplates = append(pbTemplates, toFreightTemplateDTO(t))
	}
	return &logisticsv1.ListFreightTemplatesResponse{Templates: pbTemplates}, nil
}

func (s *LogisticsGRPCServer) DeleteFreightTemplate(ctx context.Context, req *logisticsv1.DeleteFreightTemplateRequest) (*logisticsv1.DeleteFreightTemplateResponse, error) {
	err := s.svc.DeleteFreightTemplate(ctx, req.GetId(), req.GetTenantId())
	if err != nil {
		return nil, err
	}
	return &logisticsv1.DeleteFreightTemplateResponse{}, nil
}

// ==================== 运费计算 ====================

func (s *LogisticsGRPCServer) CalculateFreight(ctx context.Context, req *logisticsv1.CalculateFreightRequest) (*logisticsv1.CalculateFreightResponse, error) {
	items := make([]service.FreightCalcItem, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		items = append(items, service.FreightCalcItem{
			SkuID: item.GetSkuId(), Quantity: item.GetQuantity(), Weight: item.GetWeight(),
		})
	}
	freight, freeShipping, err := s.svc.CalculateFreight(ctx, req.GetTenantId(), req.GetProvince(), items)
	if err != nil {
		return nil, err
	}
	return &logisticsv1.CalculateFreightResponse{Freight: freight, FreeShipping: freeShipping}, nil
}

// ==================== 物流 ====================

func (s *LogisticsGRPCServer) CreateShipment(ctx context.Context, req *logisticsv1.CreateShipmentRequest) (*logisticsv1.CreateShipmentResponse, error) {
	shipment, err := s.svc.CreateShipment(ctx, domain.Shipment{
		TenantID: req.GetTenantId(), OrderID: req.GetOrderId(),
		CarrierCode: req.GetCarrierCode(), CarrierName: req.GetCarrierName(),
		TrackingNo: req.GetTrackingNo(),
	})
	if err != nil {
		return nil, err
	}
	return &logisticsv1.CreateShipmentResponse{Id: shipment.ID}, nil
}

func (s *LogisticsGRPCServer) GetShipment(ctx context.Context, req *logisticsv1.GetShipmentRequest) (*logisticsv1.GetShipmentResponse, error) {
	shipment, err := s.svc.GetShipment(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &logisticsv1.GetShipmentResponse{Shipment: toShipmentDTO(shipment)}, nil
}

func (s *LogisticsGRPCServer) GetShipmentByOrder(ctx context.Context, req *logisticsv1.GetShipmentByOrderRequest) (*logisticsv1.GetShipmentByOrderResponse, error) {
	shipment, err := s.svc.GetShipmentByOrder(ctx, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &logisticsv1.GetShipmentByOrderResponse{Shipment: toShipmentDTO(shipment)}, nil
}

func (s *LogisticsGRPCServer) AddTrack(ctx context.Context, req *logisticsv1.AddTrackRequest) (*logisticsv1.AddTrackResponse, error) {
	err := s.svc.AddTrack(ctx, req.GetShipmentId(), req.GetDescription(), req.GetLocation(), req.GetTrackTime())
	if err != nil {
		return nil, err
	}
	return &logisticsv1.AddTrackResponse{}, nil
}

// ==================== DTO Converters ====================

func toFreightTemplateDTO(t domain.FreightTemplate) *logisticsv1.FreightTemplate {
	rules := make([]*logisticsv1.FreightRule, 0, len(t.Rules))
	for _, r := range t.Rules {
		rules = append(rules, &logisticsv1.FreightRule{
			Id: r.ID, TemplateId: r.TemplateID, Regions: r.Regions,
			FirstUnit: r.FirstUnit, FirstPrice: r.FirstPrice,
			AdditionalUnit: r.AdditionalUnit, AdditionalPrice: r.AdditionalPrice,
		})
	}
	return &logisticsv1.FreightTemplate{
		Id: t.ID, TenantId: t.TenantID, Name: t.Name,
		ChargeType: t.ChargeType, FreeThreshold: t.FreeThreshold,
		Rules: rules,
		Ctime: timestamppb.New(t.Ctime), Utime: timestamppb.New(t.Utime),
	}
}

func toShipmentDTO(s domain.Shipment) *logisticsv1.Shipment {
	tracks := make([]*logisticsv1.ShipmentTrack, 0, len(s.Tracks))
	for _, t := range s.Tracks {
		tracks = append(tracks, &logisticsv1.ShipmentTrack{
			Id: t.ID, ShipmentId: t.ShipmentID, Description: t.Description,
			Location: t.Location, TrackTime: t.TrackTime,
		})
	}
	return &logisticsv1.Shipment{
		Id: s.ID, TenantId: s.TenantID, OrderId: s.OrderID,
		CarrierCode: s.CarrierCode, CarrierName: s.CarrierName,
		TrackingNo: s.TrackingNo, Status: s.Status,
		Tracks: tracks,
		Ctime:  timestamppb.New(s.Ctime), Utime: timestamppb.New(s.Utime),
	}
}
