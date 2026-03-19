package middleware

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
)

type tenantTestServer struct {
	tenantv1.UnimplementedTenantServiceServer
	getShopByDomain func(context.Context, *tenantv1.GetShopByDomainRequest) (*tenantv1.GetShopByDomainResponse, error)
}

func (s *tenantTestServer) GetShopByDomain(ctx context.Context, req *tenantv1.GetShopByDomainRequest) (*tenantv1.GetShopByDomainResponse, error) {
	if s.getShopByDomain == nil {
		return nil, errors.New("getShopByDomain not implemented")
	}
	return s.getShopByDomain(ctx, req)
}

func newTenantClient(t *testing.T, fn func(context.Context, *tenantv1.GetShopByDomainRequest) (*tenantv1.GetShopByDomainResponse, error)) tenantv1.TenantServiceClient {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	tenantv1.RegisterTenantServiceServer(server, &tenantTestServer{getShopByDomain: fn})

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("new tenant grpc client: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	return tenantv1.NewTenantServiceClient(conn)
}

func newTenantRedis(t *testing.T) *redis.Client {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

func TestTenantResolveCachesNotFoundAsNegativeResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	client := newTenantClient(t, func(context.Context, *tenantv1.GetShopByDomainRequest) (*tenantv1.GetShopByDomainResponse, error) {
		return nil, status.Error(codes.NotFound, "店铺不存在")
	})
	rdb := newTenantRedis(t)

	engine := gin.New()
	engine.Use(NewTenantResolve(client, rdb).Build())
	engine.GET("/ping", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://shop.example.com/ping", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}

	val, err := rdb.Get(t.Context(), tenantDomainKeyPrefix+"shop.example.com").Result()
	if err != nil {
		t.Fatalf("expected negative cache entry: %v", err)
	}
	if val != tenantNegativeValue {
		t.Fatalf("expected negative cache value %q, got %q", tenantNegativeValue, val)
	}
}

func TestTenantResolveDoesNotNegativeCacheInternalErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	client := newTenantClient(t, func(context.Context, *tenantv1.GetShopByDomainRequest) (*tenantv1.GetShopByDomainResponse, error) {
		return nil, status.Error(codes.Internal, "内部错误: boom")
	})
	rdb := newTenantRedis(t)

	engine := gin.New()
	engine.Use(NewTenantResolve(client, rdb).Build())
	engine.GET("/ping", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://shop.example.com/ping", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for tenant service errors, got %d", recorder.Code)
	}

	if _, err := rdb.Get(t.Context(), tenantDomainKeyPrefix+"shop.example.com").Result(); err == nil {
		t.Fatal("expected no negative cache entry for internal error")
	}
}
