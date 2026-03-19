//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/rermrf/mall/merchant-bff/handler"
	"github.com/rermrf/mall/merchant-bff/ioc"
)

var thirdPartySet = wire.NewSet(
	ioc.InitEtcdClient,
	ioc.InitLogger,
	ioc.InitRedis,
	ioc.InitUserClient,
	ioc.InitTenantClient,
	ioc.InitInventoryClient,
	ioc.InitOrderClient,
	ioc.InitPaymentClient,
	ioc.InitMarketingClient,
	ioc.InitLogisticsClient,
	ioc.InitNotificationClient,
	ioc.InitProductClient,
	ioc.InitAccountClient,
)

var handlerSet = wire.NewSet(
	ioc.InitJWTHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	handler.NewInventoryHandler,
	handler.NewOrderHandler,
	handler.NewPaymentHandler,
	handler.NewMarketingHandler,
	handler.NewLogisticsHandler,
	handler.NewNotificationHandler,
	handler.NewProductHandler,
	handler.NewAccountHandler,
	ioc.InitGinServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, handlerSet, wire.Struct(new(App), "*"))
	return new(App)
}
