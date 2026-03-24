//go:build wireinject

package main

import (
	"github.com/google/wire"
	pgrpc "github.com/rermrf/mall/payment/grpc"
	"github.com/rermrf/mall/payment/ioc"
	"github.com/rermrf/mall/payment/repository"
	"github.com/rermrf/mall/payment/repository/cache"
	"github.com/rermrf/mall/payment/repository/dao"
	"github.com/rermrf/mall/payment/service"
	"github.com/rermrf/mall/payment/service/channel"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitOrderClient,
	ioc.InitIdempotencyService,
	ioc.InitSnowflakeNode,
)

var paymentSet = wire.NewSet(
	dao.NewPaymentDAO,
	dao.NewReconciliationDAO,
	cache.NewPaymentCache,
	repository.NewPaymentRepository,
	channel.NewMockChannel,
	ioc.InitAlipayClient,
	channel.NewAlipayChannel,
	ioc.InitWechatConfig,
	ioc.InitWechatClient,
	channel.NewWechatChannel,
	service.NewOrderRefundSyncer,
	service.NewPaymentService,
	service.NewReconciliationService,
	service.NewReconciliationJob,
	pgrpc.NewPaymentGRPCServer,
	ioc.InitSyncProducer,
	ioc.InitProducer,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, paymentSet, wire.Struct(new(App), "*"))
	return new(App)
}
