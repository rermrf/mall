package ioc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/rermrf/mall/admin-bff/handler"
	ijwt "github.com/rermrf/mall/admin-bff/handler/jwt"
	"github.com/rermrf/mall/pkg/ginx"
	pkglogger "github.com/rermrf/mall/pkg/logger"
)

func issueAdminRefreshToken(t *testing.T, jwtHandler *ijwt.JWTHandler) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "unit-test")
	ctx.Request = req

	if err := jwtHandler.SetTokenHeaders(ctx, 123, 0); err != nil {
		t.Fatalf("issue admin tokens: %v", err)
	}
	refreshToken := recorder.Header().Get("X-Refresh-Token")
	if refreshToken == "" {
		t.Fatal("expected refresh token")
	}
	return refreshToken
}

func TestRefreshRouteAllowsRefreshTokenWithoutAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	jwtHandler := ijwt.NewJWTHandler(nil, rdb, "access-secret", "refresh-secret")
	refreshToken := issueAdminRefreshToken(t, jwtHandler)

	engine := InitGinServer(
		jwtHandler,
		&handler.UserHandler{},
		&handler.TenantHandler{},
		&handler.ProductHandler{},
		&handler.OrderHandler{},
		&handler.PaymentHandler{},
		&handler.NotificationHandler{},
		&handler.InventoryHandler{},
		&handler.MarketingHandler{},
		&handler.LogisticsHandler{},
		&handler.AccountHandler{},
		pkglogger.NewNopLogger(),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh-token", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Refresh-Token", refreshToken)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var res ginx.Result
	if err := json.NewDecoder(recorder.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.Code != 0 {
		t.Fatalf("expected successful refresh, got %+v", res)
	}
	if recorder.Header().Get("X-Jwt-Token") == "" {
		t.Fatal("expected refreshed access token header")
	}
}
