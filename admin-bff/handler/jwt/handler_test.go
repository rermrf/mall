package ijwt

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

func issueTokens(t *testing.T, h *JWTHandler, tenantID int64) (string, string) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "unit-test")
	ctx.Request = req

	if err := h.SetTokenHeaders(ctx, 123, tenantID); err != nil {
		t.Fatalf("issue tokens: %v", err)
	}

	accessToken := recorder.Header().Get("X-Jwt-Token")
	refreshToken := recorder.Header().Get("X-Refresh-Token")
	if accessToken == "" || refreshToken == "" {
		t.Fatalf("expected both access and refresh tokens, got access=%q refresh=%q", accessToken, refreshToken)
	}
	return accessToken, refreshToken
}

func parseClaims(t *testing.T, tokenStr string, secret []byte) *Claims {
	t.Helper()

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if !token.Valid {
		t.Fatal("expected valid token")
	}
	return claims
}

func newRedisClient(t *testing.T) redis.Cmdable {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

func TestLogoutAllowsMissingRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rdb := newRedisClient(t)
	handler := NewJWTHandler(nil, rdb, "access-secret", "refresh-secret")
	accessToken, _ := issueTokens(t, handler, 0)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("POST", "/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	ctx.Request = req

	res, err := handler.Logout(ctx, LogoutReq{})
	if err != nil {
		t.Fatalf("logout returned error: %v", err)
	}
	if res.Code != 0 {
		t.Fatalf("expected success when refresh token is missing, got %+v", res)
	}
}

func TestLogoutBlacklistsAccessAndRefreshTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rdb := newRedisClient(t)
	handler := NewJWTHandler(nil, rdb, "access-secret", "refresh-secret")
	accessToken, refreshToken := issueTokens(t, handler, 0)

	accessClaims := parseClaims(t, accessToken, handler.accessSecret)
	refreshClaims := parseClaims(t, refreshToken, handler.refreshSecret)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("POST", "/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Refresh-Token", refreshToken)
	ctx.Request = req

	res, err := handler.Logout(ctx, LogoutReq{})
	if err != nil {
		t.Fatalf("logout returned error: %v", err)
	}
	if res.Code != 0 {
		t.Fatalf("expected successful logout, got %+v", res)
	}

	if !handler.IsTokenBlacklisted(context.Background(), accessClaims.ID) {
		t.Fatalf("expected access token %s to be blacklisted", accessClaims.ID)
	}
	if !handler.IsTokenBlacklisted(context.Background(), refreshClaims.ID) {
		t.Fatalf("expected refresh token %s to be blacklisted", refreshClaims.ID)
	}
}
