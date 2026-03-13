package ioc

import (
	"context"
	"fmt"
	"strconv"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/notification/events"
	"github.com/rermrf/mall/notification/service"
	"github.com/rermrf/mall/pkg/saramax"
	"github.com/rermrf/mall/pkg/tenantx"
	"github.com/spf13/viper"
)

func InitKafka() sarama.Client {
	type Config struct {
		Addrs []string `yaml:"addrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("kafka", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Kafka 配置失败: %w", err))
	}
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	client, err := sarama.NewClient(cfg.Addrs, saramaCfg)
	if err != nil {
		panic(fmt.Errorf("连接 Kafka 失败: %w", err))
	}
	return client
}

func InitConsumerGroup(client sarama.Client) sarama.ConsumerGroup {
	cg, err := sarama.NewConsumerGroupFromClient("notification-svc", client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka ConsumerGroup 失败: %w", err))
	}
	return cg
}

func NewUserRegisteredConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
) *events.UserRegisteredConsumer {
	return events.NewUserRegisteredConsumer(cg, l, func(ctx context.Context, evt events.UserRegisteredEvent) error {
		ctx = tenantx.WithTenantID(ctx, evt.TenantId)
		l.Info("收到用户注册事件", logger.Int64("userId", evt.UserId))
		params := map[string]string{
			"phone": evt.Phone,
		}
		// 发送欢迎短信
		_, _ = svc.SendNotification(ctx, evt.UserId, evt.TenantId, "welcome_sms", 1, params)
		// 发送欢迎邮件
		_, _ = svc.SendNotification(ctx, evt.UserId, evt.TenantId, "welcome_email", 2, params)
		return nil
	})
}

func NewOrderPaidConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
	orderClient orderv1.OrderServiceClient,
) *events.OrderPaidConsumer {
	return events.NewOrderPaidConsumer(cg, l, func(ctx context.Context, evt events.OrderPaidEvent) error {
		l.Info("收到订单支付事件", logger.String("orderNo", evt.OrderNo))
		orderResp, err := orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderNo: evt.OrderNo})
		if err == nil {
			ctx = tenantx.WithTenantID(ctx, orderResp.GetOrder().GetTenantId())
		}
		if err != nil {
			l.Error("获取订单详情失败", logger.String("orderNo", evt.OrderNo), logger.Error(err))
			return nil
		}
		order := orderResp.GetOrder()
		params := map[string]string{
			"OrderNo":   evt.OrderNo,
			"PaymentNo": evt.PaymentNo,
		}
		_, _ = svc.SendNotification(ctx, order.GetTenantId(), order.GetTenantId(), "order_paid_inapp", 3, params)
		return nil
	})
}

func NewOrderShippedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
	orderClient orderv1.OrderServiceClient,
) *events.OrderShippedConsumer {
	return events.NewOrderShippedConsumer(cg, l, func(ctx context.Context, evt events.OrderShippedEvent) error {
		ctx = tenantx.WithTenantID(ctx, evt.TenantId)
		l.Info("收到订单发货事件",
			logger.Int64("orderId", evt.OrderId),
			logger.String("trackingNo", evt.TrackingNo))
		if evt.OrderNo == "" {
			l.Warn("order_shipped 事件缺少 OrderNo，无法查询订单详情",
				logger.Int64("orderId", evt.OrderId))
			return nil
		}
		orderResp, err := orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderNo: evt.OrderNo})
		if err != nil {
			l.Error("获取订单详情失败", logger.String("orderNo", evt.OrderNo), logger.Error(err))
			return nil
		}
		order := orderResp.GetOrder()
		params := map[string]string{
			"OrderNo":     evt.OrderNo,
			"TrackingNo":  evt.TrackingNo,
			"CarrierName": evt.CarrierName,
		}
		_, _ = svc.SendNotification(ctx, order.GetBuyerId(), evt.TenantId, "order_shipped_inapp", 3, params)
		return nil
	})
}

func NewInventoryAlertConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
) *events.InventoryAlertConsumer {
	return events.NewInventoryAlertConsumer(cg, l, func(ctx context.Context, evt events.InventoryAlertEvent) error {
		ctx = tenantx.WithTenantID(ctx, evt.TenantID)
		l.Info("收到库存预警事件", logger.Int64("skuId", evt.SKUID))
		params := map[string]string{
			"SKUID":     strconv.FormatInt(evt.SKUID, 10),
			"Available": strconv.FormatInt(int64(evt.Available), 10),
			"Threshold": strconv.FormatInt(int64(evt.Threshold), 10),
		}
		// 使用 tenantId 作为 userId 是合理近似：多租户场景下商家管理员 ID 通常等于 tenantId
		_, _ = svc.SendNotification(ctx, evt.TenantID, evt.TenantID, "inventory_alert_inapp", 3, params)
		_, _ = svc.SendNotification(ctx, evt.TenantID, evt.TenantID, "inventory_alert_email", 2, params)
		return nil
	})
}

func NewTenantApprovedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
) *events.TenantApprovedConsumer {
	return events.NewTenantApprovedConsumer(cg, l, func(ctx context.Context, evt events.TenantApprovedEvent) error {
		ctx = tenantx.WithTenantID(ctx, evt.TenantId)
		l.Info("收到租户审核通过事件", logger.Int64("tenantId", evt.TenantId))
		params := map[string]string{
			"TenantName": evt.Name,
		}
		// 使用 tenantId 作为 userId 是合理近似：多租户场景下商家管理员 ID 通常等于 tenantId
		_, _ = svc.SendNotification(ctx, evt.TenantId, evt.TenantId, "tenant_approved_inapp", 3, params)
		_, _ = svc.SendNotification(ctx, evt.TenantId, evt.TenantId, "tenant_approved_email", 2, params)
		return nil
	})
}

func NewTenantPlanChangedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
) *events.TenantPlanChangedConsumer {
	return events.NewTenantPlanChangedConsumer(cg, l, func(ctx context.Context, evt events.TenantPlanChangedEvent) error {
		ctx = tenantx.WithTenantID(ctx, evt.TenantId)
		l.Info("收到租户套餐变更事件", logger.Int64("tenantId", evt.TenantId))
		params := map[string]string{
			"OldPlanId": strconv.FormatInt(evt.OldPlanId, 10),
			"NewPlanId": strconv.FormatInt(evt.NewPlanId, 10),
		}
		// 使用 tenantId 作为 userId 是合理近似：多租户场景下商家管理员 ID 通常等于 tenantId
		_, _ = svc.SendNotification(ctx, evt.TenantId, evt.TenantId, "tenant_plan_changed_inapp", 3, params)
		_, _ = svc.SendNotification(ctx, evt.TenantId, evt.TenantId, "tenant_plan_changed_email", 2, params)
		return nil
	})
}

func NewOrderCompletedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
	orderClient orderv1.OrderServiceClient,
) *events.OrderCompletedConsumer {
	return events.NewOrderCompletedConsumer(cg, l, func(ctx context.Context, evt events.OrderCompletedEvent) error {
		ctx = tenantx.WithTenantID(ctx, evt.TenantID)
		l.Info("收到订单完成事件", logger.String("orderNo", evt.OrderNo))
		orderResp, err := orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderNo: evt.OrderNo})
		if err != nil {
			l.Error("获取订单详情失败", logger.String("orderNo", evt.OrderNo), logger.Error(err))
			return nil
		}
		order := orderResp.GetOrder()
		params := map[string]string{
			"OrderNo": evt.OrderNo,
		}
		_, _ = svc.SendNotification(ctx, order.GetBuyerId(), evt.TenantID, "order_completed_inapp", 3, params)
		return nil
	})
}

func InitConsumers(
	userRegistered *events.UserRegisteredConsumer,
	orderPaid *events.OrderPaidConsumer,
	orderShipped *events.OrderShippedConsumer,
	inventoryAlert *events.InventoryAlertConsumer,
	tenantApproved *events.TenantApprovedConsumer,
	tenantPlanChanged *events.TenantPlanChangedConsumer,
	orderCompleted *events.OrderCompletedConsumer,
) []saramax.Consumer {
	return []saramax.Consumer{userRegistered, orderPaid, orderShipped, inventoryAlert, tenantApproved, tenantPlanChanged, orderCompleted}
}
