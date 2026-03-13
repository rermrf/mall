//go:build wireinject

package main

import (
	"github.com/google/wire"
	igrpc "github.com/rermrf/mall/product/grpc"
	"github.com/rermrf/mall/product/ioc"
	"github.com/rermrf/mall/product/repository"
	"github.com/rermrf/mall/product/repository/cache"
	"github.com/rermrf/mall/product/repository/dao"
	"github.com/rermrf/mall/product/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitTenantClient,
)

var productSet = wire.NewSet(
	dao.NewProductDAO,
	dao.NewSKUDAO,
	dao.NewSpecDAO,
	dao.NewCategoryDAO,
	dao.NewBrandDAO,
	cache.NewProductCache,
	repository.NewProductRepository,
	repository.NewCategoryRepository,
	repository.NewBrandRepository,
	service.NewProductService,
	igrpc.NewProductGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, productSet, wire.Struct(new(App), "*"))
	return new(App)
}
