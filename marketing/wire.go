//go:build wireinject

package main

import (
	"github.com/google/wire"
	mgrpc "github.com/rermrf/mall/marketing/grpc"
	"github.com/rermrf/mall/marketing/ioc"
	"github.com/rermrf/mall/marketing/repository"
	"github.com/rermrf/mall/marketing/repository/cache"
	"github.com/rermrf/mall/marketing/repository/dao"
	"github.com/rermrf/mall/marketing/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitOrderClient,
)

var marketingSet = wire.NewSet(
	dao.NewCouponDAO,
	dao.NewUserCouponDAO,
	dao.NewSeckillDAO,
	dao.NewPromotionDAO,
	cache.NewMarketingCache,
	repository.NewMarketingRepository,
	service.NewMarketingService,
	mgrpc.NewMarketingGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitGRPCServer,
	ioc.InitConsumerGroup,
	ioc.NewOrderCancelledConsumer,
	ioc.InitConsumers,
)

func InitApp() *App {
	wire.Build(thirdPartySet, marketingSet, wire.Struct(new(App), "*"))
	return new(App)
}
