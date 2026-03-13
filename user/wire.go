//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/rermrf/mall/user/grpc"
	"github.com/rermrf/mall/user/ioc"
	"github.com/rermrf/mall/user/repository"
	"github.com/rermrf/mall/user/repository/cache"
	"github.com/rermrf/mall/user/repository/dao"
	"github.com/rermrf/mall/user/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
)

var userSet = wire.NewSet(
	dao.NewUserDAO,
	dao.NewRoleDAO,
	dao.NewAddressDAO,
	cache.NewUserCache,
	repository.NewUserRepository,
	service.NewUserService,
	grpc.NewUserGRPCServer,

	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitConsumerGroup,
	ioc.NewUserRegisteredConsumer,
	ioc.InitConsumers,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, userSet, wire.Struct(new(App), "*"))
	return new(App)
}
