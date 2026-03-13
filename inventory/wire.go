//go:build wireinject

package main

import (
	"github.com/google/wire"
	igrpc "github.com/rermrf/mall/inventory/grpc"
	"github.com/rermrf/mall/inventory/ioc"
	"github.com/rermrf/mall/inventory/repository"
	"github.com/rermrf/mall/inventory/repository/cache"
	"github.com/rermrf/mall/inventory/repository/dao"
	"github.com/rermrf/mall/inventory/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
)

var inventorySet = wire.NewSet(
	dao.NewInventoryDAO,
	cache.NewInventoryCache,
	repository.NewInventoryRepository,
	service.NewInventoryService,
	igrpc.NewInventoryGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitConsumerGroup,
	ioc.NewDeductExpireConsumer,
	ioc.InitConsumers,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, inventorySet, wire.Struct(new(App), "*"))
	return new(App)
}
