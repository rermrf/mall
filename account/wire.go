//go:build wireinject

package main

import (
	"github.com/google/wire"
	agrpc "github.com/rermrf/mall/account/grpc"
	"github.com/rermrf/mall/account/ioc"
	"github.com/rermrf/mall/account/repository"
	"github.com/rermrf/mall/account/repository/cache"
	"github.com/rermrf/mall/account/repository/dao"
	"github.com/rermrf/mall/account/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitSnowflakeNode,
)

var accountSet = wire.NewSet(
	dao.NewAccountDAO,
	cache.NewAccountCache,
	repository.NewAccountRepository,
	service.NewAccountService,
	agrpc.NewAccountGRPCServer,
	ioc.InitConsumerGroup,
	ioc.NewOrderCompletedConsumer,
	ioc.InitConsumers,
	ioc.InitTenantClient,
	ioc.InitTenantIntegration,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, accountSet, wire.Struct(new(App), "*"))
	return new(App)
}
