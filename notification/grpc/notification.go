package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	notificationv1 "github.com/rermrf/mall/api/proto/gen/notification/v1"
	"github.com/rermrf/mall/notification/domain"
	"github.com/rermrf/mall/notification/service"
)

type NotificationGRPCServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	svc service.NotificationService
}

func NewNotificationGRPCServer(svc service.NotificationService) *NotificationGRPCServer {
	return &NotificationGRPCServer{svc: svc}
}

func (s *NotificationGRPCServer) Register(server *grpc.Server) {
	notificationv1.RegisterNotificationServiceServer(server, s)
}

func (s *NotificationGRPCServer) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	n, err := s.svc.SendNotification(ctx, req.GetUserId(), req.GetTenantId(), req.GetTemplateCode(), req.GetChannel(), req.GetParams())
	if err != nil {
		return nil, err
	}
	return &notificationv1.SendNotificationResponse{Id: n.ID}, nil
}

func (s *NotificationGRPCServer) BatchSendNotification(ctx context.Context, req *notificationv1.BatchSendNotificationRequest) (*notificationv1.BatchSendNotificationResponse, error) {
	success, fail, err := s.svc.BatchSendNotification(ctx, req.GetUserIds(), req.GetTenantId(), req.GetTemplateCode(), req.GetChannel(), req.GetParams())
	if err != nil {
		return nil, err
	}
	return &notificationv1.BatchSendNotificationResponse{SuccessCount: success, FailCount: fail}, nil
}

func (s *NotificationGRPCServer) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	ns, total, err := s.svc.ListNotifications(ctx, req.GetUserId(), req.GetChannel(), req.GetUnreadOnly(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	pbNs := make([]*notificationv1.Notification, 0, len(ns))
	for _, n := range ns {
		pbNs = append(pbNs, toNotificationDTO(n))
	}
	return &notificationv1.ListNotificationsResponse{Notifications: pbNs, Total: total}, nil
}

func (s *NotificationGRPCServer) MarkRead(ctx context.Context, req *notificationv1.MarkReadRequest) (*notificationv1.MarkReadResponse, error) {
	if err := s.svc.MarkRead(ctx, req.GetId(), req.GetUserId()); err != nil {
		return nil, err
	}
	return &notificationv1.MarkReadResponse{}, nil
}

func (s *NotificationGRPCServer) MarkAllRead(ctx context.Context, req *notificationv1.MarkAllReadRequest) (*notificationv1.MarkAllReadResponse, error) {
	if err := s.svc.MarkAllRead(ctx, req.GetUserId()); err != nil {
		return nil, err
	}
	return &notificationv1.MarkAllReadResponse{}, nil
}

func (s *NotificationGRPCServer) GetUnreadCount(ctx context.Context, req *notificationv1.GetUnreadCountRequest) (*notificationv1.GetUnreadCountResponse, error) {
	count, err := s.svc.GetUnreadCount(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &notificationv1.GetUnreadCountResponse{Count: count}, nil
}

func (s *NotificationGRPCServer) CreateTemplate(ctx context.Context, req *notificationv1.CreateTemplateRequest) (*notificationv1.CreateTemplateResponse, error) {
	t := req.GetTemplate()
	tmpl, err := s.svc.CreateTemplate(ctx, domain.NotificationTemplate{
		TenantID: t.GetTenantId(), Code: t.GetCode(), Channel: t.GetChannel(),
		Title: t.GetTitle(), Content: t.GetContent(), Status: t.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &notificationv1.CreateTemplateResponse{Id: tmpl.ID}, nil
}

func (s *NotificationGRPCServer) UpdateTemplate(ctx context.Context, req *notificationv1.UpdateTemplateRequest) (*notificationv1.UpdateTemplateResponse, error) {
	t := req.GetTemplate()
	if err := s.svc.UpdateTemplate(ctx, domain.NotificationTemplate{
		ID: t.GetId(), Title: t.GetTitle(), Content: t.GetContent(), Status: t.GetStatus(),
	}); err != nil {
		return nil, err
	}
	return &notificationv1.UpdateTemplateResponse{}, nil
}

func (s *NotificationGRPCServer) DeleteTemplate(ctx context.Context, req *notificationv1.DeleteTemplateRequest) (*notificationv1.DeleteTemplateResponse, error) {
	if err := s.svc.DeleteTemplate(ctx, req.GetId(), req.GetTenantId()); err != nil {
		return nil, err
	}
	return &notificationv1.DeleteTemplateResponse{}, nil
}

func (s *NotificationGRPCServer) ListTemplates(ctx context.Context, req *notificationv1.ListTemplatesRequest) (*notificationv1.ListTemplatesResponse, error) {
	templates, err := s.svc.ListTemplates(ctx, req.GetTenantId(), req.GetChannel())
	if err != nil {
		return nil, err
	}
	pbTemplates := make([]*notificationv1.NotificationTemplate, 0, len(templates))
	for _, t := range templates {
		pbTemplates = append(pbTemplates, toTemplateDTO(t))
	}
	return &notificationv1.ListTemplatesResponse{Templates: pbTemplates}, nil
}

func toNotificationDTO(n domain.Notification) *notificationv1.Notification {
	return &notificationv1.Notification{
		Id: n.ID, UserId: n.UserID, TenantId: n.TenantID, Channel: n.Channel,
		Title: n.Title, Content: n.Content, IsRead: n.IsRead, Status: n.Status,
		Ctime: timestamppb.New(n.Ctime),
	}
}

func toTemplateDTO(t domain.NotificationTemplate) *notificationv1.NotificationTemplate {
	return &notificationv1.NotificationTemplate{
		Id: t.ID, TenantId: t.TenantID, Code: t.Code, Channel: t.Channel,
		Title: t.Title, Content: t.Content, Status: t.Status,
		Ctime: timestamppb.New(t.Ctime),
	}
}
