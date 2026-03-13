//go:build wireinject

package main

import (
	"github.com/google/wire"
	ogrpc "github.com/rermrf/mall/order/grpc"
	"github.com/rermrf/mall/order/ioc"
	"github.com/rermrf/mall/order/repository"
	"github.com/rermrf/mall/order/repository/cache"
	"github.com/rermrf/mall/order/repository/dao"
	"github.com/rermrf/mall/order/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitProductClient,
	ioc.InitInventoryClient,
	ioc.InitPaymentClient,
	ioc.InitUserClient,
	ioc.InitIdempotencyService,
	ioc.InitSnowflakeNode,
)

var orderSet = wire.NewSet(
	dao.NewOrderDAO,
	cache.NewOrderCache,
	repository.NewOrderRepository,
	service.NewOrderService,
	ogrpc.NewOrderGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitConsumerGroup,
	ioc.NewOrderPaidConsumer,
	ioc.NewOrderCloseDelayConsumer,
	ioc.NewSeckillSuccessConsumer,
	ioc.InitConsumers,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, orderSet, wire.Struct(new(App), "*"))
	return new(App)
}
