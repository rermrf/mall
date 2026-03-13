//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/rermrf/mall/consumer-bff/handler"
	"github.com/rermrf/mall/consumer-bff/ioc"
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
	ioc.InitCartClient,
	ioc.InitProductClient,
	ioc.InitSearchClient,
	ioc.InitMarketingClient,
	ioc.InitLogisticsClient,
	ioc.InitNotificationClient,
)

var handlerSet = wire.NewSet(
	ioc.InitJWTHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	handler.NewInventoryHandler,
	handler.NewOrderHandler,
	handler.NewPaymentHandler,
	handler.NewCartHandler,
	handler.NewSearchHandler,
	handler.NewMarketingHandler,
	handler.NewLogisticsHandler,
	handler.NewNotificationHandler,
	ioc.InitGinServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, handlerSet, wire.Struct(new(App), "*"))
	return new(App)
}
