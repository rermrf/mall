//go:build wireinject

package main

import (
	"github.com/google/wire"
	cgrpc "github.com/rermrf/mall/cart/grpc"
	"github.com/rermrf/mall/cart/ioc"
	"github.com/rermrf/mall/cart/repository"
	"github.com/rermrf/mall/cart/repository/cache"
	"github.com/rermrf/mall/cart/repository/dao"
	"github.com/rermrf/mall/cart/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitLogger,
	ioc.InitEtcdClient,
)

var cartSet = wire.NewSet(
	dao.NewCartDAO,
	cache.NewCartCache,
	repository.NewCartRepository,
	service.NewCartService,
	cgrpc.NewCartGRPCServer,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, cartSet, wire.Struct(new(App), "*"))
	return new(App)
}
