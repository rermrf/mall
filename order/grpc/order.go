package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/order/domain"
	"github.com/rermrf/mall/order/service"
)

type OrderGRPCServer struct {
	orderv1.UnimplementedOrderServiceServer
	svc service.OrderService
}

func NewOrderGRPCServer(svc service.OrderService) *OrderGRPCServer {
	return &OrderGRPCServer{svc: svc}
}

func (s *OrderGRPCServer) Register(server *grpc.Server) {
	orderv1.RegisterOrderServiceServer(server, s)
}

func (s *OrderGRPCServer) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	items := make([]service.CreateOrderItemReq, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		items = append(items, service.CreateOrderItemReq{
			SKUID:    item.GetSkuId(),
			Quantity: item.GetQuantity(),
		})
	}
	orderNo, payAmount, err := s.svc.CreateOrder(ctx, service.CreateOrderReq{
		BuyerID:   req.GetBuyerId(),
		TenantID:  req.GetTenantId(),
		Items:     items,
		AddressID: req.GetAddressId(),
		CouponID:  req.GetCouponId(),
		Remark:    req.GetRemark(),
	})
	if err != nil {
		return nil, err
	}
	return &orderv1.CreateOrderResponse{OrderNo: orderNo, PayAmount: payAmount}, nil
}

func (s *OrderGRPCServer) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	order, err := s.svc.GetOrder(ctx, req.GetOrderNo())
	if err != nil {
		return nil, err
	}
	return &orderv1.GetOrderResponse{Order: s.toOrderDTO(order)}, nil
}

func (s *OrderGRPCServer) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	orders, total, err := s.svc.ListOrders(ctx, req.GetBuyerId(), req.GetTenantId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*orderv1.Order, 0, len(orders))
	for _, o := range orders {
		dtos = append(dtos, s.toOrderDTO(o))
	}
	return &orderv1.ListOrdersResponse{Orders: dtos, Total: total}, nil
}

func (s *OrderGRPCServer) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.CancelOrderResponse, error) {
	err := s.svc.CancelOrder(ctx, req.GetOrderNo(), req.GetBuyerId())
	if err != nil {
		return nil, err
	}
	return &orderv1.CancelOrderResponse{}, nil
}

func (s *OrderGRPCServer) ConfirmReceive(ctx context.Context, req *orderv1.ConfirmReceiveRequest) (*orderv1.ConfirmReceiveResponse, error) {
	err := s.svc.ConfirmReceive(ctx, req.GetOrderNo(), req.GetBuyerId())
	if err != nil {
		return nil, err
	}
	return &orderv1.ConfirmReceiveResponse{}, nil
}

func (s *OrderGRPCServer) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.UpdateOrderStatusResponse, error) {
	err := s.svc.UpdateOrderStatus(ctx, req.GetOrderNo(), req.GetStatus(), req.GetOperatorId(), req.GetOperatorType(), req.GetRemark())
	if err != nil {
		return nil, err
	}
	return &orderv1.UpdateOrderStatusResponse{}, nil
}

func (s *OrderGRPCServer) ApplyRefund(ctx context.Context, req *orderv1.ApplyRefundRequest) (*orderv1.ApplyRefundResponse, error) {
	refundNo, err := s.svc.ApplyRefund(ctx, req.GetOrderNo(), req.GetBuyerId(), req.GetType(), req.GetRefundAmount(), req.GetReason())
	if err != nil {
		return nil, err
	}
	return &orderv1.ApplyRefundResponse{RefundNo: refundNo}, nil
}

func (s *OrderGRPCServer) HandleRefund(ctx context.Context, req *orderv1.HandleRefundRequest) (*orderv1.HandleRefundResponse, error) {
	err := s.svc.HandleRefund(ctx, req.GetRefundNo(), req.GetTenantId(), req.GetApproved(), req.GetReason())
	if err != nil {
		return nil, err
	}
	return &orderv1.HandleRefundResponse{}, nil
}

func (s *OrderGRPCServer) GetRefundOrder(ctx context.Context, req *orderv1.GetRefundOrderRequest) (*orderv1.GetRefundOrderResponse, error) {
	refund, err := s.svc.GetRefundOrder(ctx, req.GetRefundNo())
	if err != nil {
		return nil, err
	}
	return &orderv1.GetRefundOrderResponse{RefundOrder: s.toRefundDTO(refund)}, nil
}

func (s *OrderGRPCServer) ListRefundOrders(ctx context.Context, req *orderv1.ListRefundOrdersRequest) (*orderv1.ListRefundOrdersResponse, error) {
	refunds, total, err := s.svc.ListRefundOrders(ctx, req.GetTenantId(), req.GetBuyerId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	dtos := make([]*orderv1.RefundOrder, 0, len(refunds))
	for _, r := range refunds {
		dtos = append(dtos, s.toRefundDTO(r))
	}
	return &orderv1.ListRefundOrdersResponse{RefundOrders: dtos, Total: total}, nil
}

func (s *OrderGRPCServer) toOrderDTO(o domain.Order) *orderv1.Order {
	items := make([]*orderv1.OrderItem, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, &orderv1.OrderItem{
			Id:           item.ID,
			OrderId:      item.OrderID,
			ProductId:    item.ProductID,
			SkuId:        item.SKUID,
			ProductName:  item.ProductName,
			SkuSpec:      item.SKUSpec,
			ProductImage: item.Image,
			Price:        item.Price,
			Quantity:     item.Quantity,
			TotalAmount:  item.Subtotal,
		})
	}
	return &orderv1.Order{
		Id:              o.ID,
		TenantId:        o.TenantID,
		OrderNo:         o.OrderNo,
		BuyerId:         o.BuyerID,
		Status:          int32(o.Status),
		TotalAmount:     o.TotalAmount,
		DiscountAmount:  o.DiscountAmount,
		FreightAmount:   o.FreightAmount,
		PayAmount:       o.PayAmount,
		CouponId:        o.CouponID,
		ReceiverName:    o.ReceiverName,
		ReceiverPhone:   o.ReceiverPhone,
		ReceiverAddress: o.ReceiverAddress,
		Remark:          o.Remark,
		PayTime:         o.PaidAt,
		ShipTime:        o.ShippedAt,
		ReceiveTime:     o.ReceivedAt,
		CloseTime:       o.ClosedAt,
		Items:           items,
		Ctime:           timestamppb.New(o.Ctime),
		Utime:           timestamppb.New(o.Utime),
	}
}

func (s *OrderGRPCServer) toRefundDTO(r domain.RefundOrder) *orderv1.RefundOrder {
	return &orderv1.RefundOrder{
		Id:           r.ID,
		TenantId:     r.TenantID,
		OrderId:      r.OrderID,
		RefundNo:     r.RefundNo,
		BuyerId:      r.BuyerID,
		Type:         r.Type,
		Status:       int32(r.Status),
		RefundAmount: r.RefundAmount,
		Reason:       r.Reason,
		Ctime:        timestamppb.New(r.Ctime),
		Utime:        timestamppb.New(r.Utime),
	}
}
