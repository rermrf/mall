//go:build wireinject

package main

import (
	"github.com/google/wire"
	sgrpc "github.com/rermrf/mall/search/grpc"
	"github.com/rermrf/mall/search/ioc"
	"github.com/rermrf/mall/search/repository"
	"github.com/rermrf/mall/search/repository/cache"
	"github.com/rermrf/mall/search/repository/dao"
	"github.com/rermrf/mall/search/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitES,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitProductClient,
)

var searchSet = wire.NewSet(
	dao.NewSearchDAO,
	cache.NewSearchCache,
	repository.NewSearchRepository,
	service.NewSearchService,
	sgrpc.NewSearchGRPCServer,
	ioc.InitGRPCServer,
	ioc.InitConsumerGroup,
	ioc.NewProductUpdatedConsumer,
	ioc.NewProductStatusChangedConsumer,
	ioc.InitConsumers,
)

func InitApp() *App {
	wire.Build(thirdPartySet, searchSet, wire.Struct(new(App), "*"))
	return new(App)
}
