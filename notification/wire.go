//go:build wireinject

package main

import (
	"github.com/google/wire"
	ngrpc "github.com/rermrf/mall/notification/grpc"
	"github.com/rermrf/mall/notification/ioc"
	"github.com/rermrf/mall/notification/repository"
	"github.com/rermrf/mall/notification/repository/cache"
	"github.com/rermrf/mall/notification/repository/dao"
	"github.com/rermrf/mall/notification/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitOrderClient,
)

var notificationSet = wire.NewSet(
	dao.NewNotificationTemplateDAO,
	dao.NewNotificationDAO,
	cache.NewNotificationCache,
	repository.NewNotificationRepository,
	service.NewNotificationService,
	ngrpc.NewNotificationGRPCServer,
	ioc.InitSmsProvider,
	ioc.InitEmailProvider,
	ioc.InitGRPCServer,
	ioc.InitConsumerGroup,
	ioc.NewUserRegisteredConsumer,
	ioc.NewOrderPaidConsumer,
	ioc.NewOrderShippedConsumer,
	ioc.NewInventoryAlertConsumer,
	ioc.NewTenantApprovedConsumer,
	ioc.NewTenantPlanChangedConsumer,
	ioc.NewOrderCompletedConsumer,
	ioc.InitConsumers,
)

func InitApp() *App {
	wire.Build(thirdPartySet, notificationSet, wire.Struct(new(App), "*"))
	return new(App)
}
