//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/rermrf/mall/admin-bff/handler"
	"github.com/rermrf/mall/admin-bff/ioc"
)

var thirdPartySet = wire.NewSet(
	ioc.InitEtcdClient,
	ioc.InitLogger,
	ioc.InitRedis,
	ioc.InitUserClient,
	ioc.InitTenantClient,
	ioc.InitProductClient,
	ioc.InitOrderClient,
	ioc.InitPaymentClient,
	ioc.InitNotificationClient,
	ioc.InitInventoryClient,
	ioc.InitMarketingClient,
	ioc.InitLogisticsClient,
	ioc.InitAccountClient,
)

var handlerSet = wire.NewSet(
	ioc.InitJWTHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	handler.NewProductHandler,
	handler.NewOrderHandler,
	handler.NewPaymentHandler,
	handler.NewNotificationHandler,
	handler.NewInventoryHandler,
	handler.NewMarketingHandler,
	handler.NewLogisticsHandler,
	handler.NewAccountHandler,
	handler.NewReconciliationHandler,
	ioc.InitGinServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, handlerSet, wire.Struct(new(App), "*"))
	return new(App)
}
