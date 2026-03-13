//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/rermrf/mall/tenant/grpc"
	"github.com/rermrf/mall/tenant/ioc"
	"github.com/rermrf/mall/tenant/repository"
	"github.com/rermrf/mall/tenant/repository/cache"
	"github.com/rermrf/mall/tenant/repository/dao"
	"github.com/rermrf/mall/tenant/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
)

var tenantSet = wire.NewSet(
	dao.NewTenantDAO,
	dao.NewPlanDAO,
	dao.NewQuotaDAO,
	dao.NewShopDAO,
	cache.NewTenantCache,
	repository.NewTenantRepository,
	service.NewTenantService,
	grpc.NewTenantGRPCServer,

	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, tenantSet, wire.Struct(new(App), "*"))
	return new(App)
}
