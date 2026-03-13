//go:build wireinject

package main

import (
	"github.com/google/wire"
	lgrpc "github.com/rermrf/mall/logistics/grpc"
	"github.com/rermrf/mall/logistics/ioc"
	"github.com/rermrf/mall/logistics/repository"
	"github.com/rermrf/mall/logistics/repository/cache"
	"github.com/rermrf/mall/logistics/repository/dao"
	"github.com/rermrf/mall/logistics/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
)

var logisticsSet = wire.NewSet(
	dao.NewFreightTemplateDAO,
	dao.NewShipmentDAO,
	dao.NewShipmentTrackDAO,
	cache.NewLogisticsCache,
	repository.NewLogisticsRepository,
	service.NewLogisticsService,
	lgrpc.NewLogisticsGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, logisticsSet, wire.Struct(new(App), "*"))
	return new(App)
}
