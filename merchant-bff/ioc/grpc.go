package ioc

import (
	"fmt"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"
	logisticsv1 "github.com/rermrf/mall/api/proto/gen/logistics/v1"
	marketingv1 "github.com/rermrf/mall/api/proto/gen/marketing/v1"
	notificationv1 "github.com/rermrf/mall/api/proto/gen/notification/v1"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
)

func InitEtcdClient() *clientv3.Client {
	var cfg struct {
		Addrs []string `yaml:"addrs"`
	}
	err := viper.UnmarshalKey("etcd", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 etcd 配置失败: %w", err))
	}
	client, err := clientv3.New(clientv3.Config{
		Endpoints: cfg.Addrs,
	})
	if err != nil {
		panic(fmt.Errorf("连接 etcd 失败: %w", err))
	}
	return client
}

func initServiceConn(etcdClient *clientv3.Client, serviceName string) *grpc.ClientConn {
	etcdResolver, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(fmt.Errorf("创建 etcd resolver 失败: %w", err))
	}

	opts := append([]grpc.DialOption{
		grpc.WithResolvers(etcdResolver),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(tenantx.GRPCUnaryClientInterceptor()),
	}, grpcx.DefaultClientDialOptions()...)
	conn, err := grpc.NewClient(
		"etcd:///service/"+serviceName,
		opts...,
	)
	if err != nil {
		panic(fmt.Errorf("连接 gRPC 服务 %s 失败: %w", serviceName, err))
	}
	return conn
}

func InitUserClient(etcdClient *clientv3.Client) userv1.UserServiceClient {
	conn := initServiceConn(etcdClient, "user")
	return userv1.NewUserServiceClient(conn)
}

func InitTenantClient(etcdClient *clientv3.Client) tenantv1.TenantServiceClient {
	conn := initServiceConn(etcdClient, "tenant")
	return tenantv1.NewTenantServiceClient(conn)
}

func InitInventoryClient(etcdClient *clientv3.Client) inventoryv1.InventoryServiceClient {
	conn := initServiceConn(etcdClient, "inventory")
	return inventoryv1.NewInventoryServiceClient(conn)
}

func InitOrderClient(etcdClient *clientv3.Client) orderv1.OrderServiceClient {
	conn := initServiceConn(etcdClient, "order")
	return orderv1.NewOrderServiceClient(conn)
}

func InitPaymentClient(etcdClient *clientv3.Client) paymentv1.PaymentServiceClient {
	conn := initServiceConn(etcdClient, "payment")
	return paymentv1.NewPaymentServiceClient(conn)
}

func InitMarketingClient(etcdClient *clientv3.Client) marketingv1.MarketingServiceClient {
	conn := initServiceConn(etcdClient, "marketing")
	return marketingv1.NewMarketingServiceClient(conn)
}

func InitLogisticsClient(etcdClient *clientv3.Client) logisticsv1.LogisticsServiceClient {
	conn := initServiceConn(etcdClient, "logistics")
	return logisticsv1.NewLogisticsServiceClient(conn)
}

func InitNotificationClient(etcdClient *clientv3.Client) notificationv1.NotificationServiceClient {
	conn := initServiceConn(etcdClient, "notification")
	return notificationv1.NewNotificationServiceClient(conn)
}

func InitProductClient(etcdClient *clientv3.Client) productv1.ProductServiceClient {
	conn := initServiceConn(etcdClient, "product")
	return productv1.NewProductServiceClient(conn)
}
