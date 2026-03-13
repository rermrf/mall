package ijwt

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/pkg/ginx"
)


type Claims struct {
	Uid       int64  `json:"uid"`
	TenantId  int64  `json:"tenant_id"`
	UserAgent string `json:"user_agent"`
	jwt.RegisteredClaims
}

type LoginReq struct {
	TenantId int64  `json:"tenant_id"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type RefreshReq struct{}

type LogoutReq struct{}

type JWTHandler struct {
	userClient    userv1.UserServiceClient
	rdb           redis.Cmdable
	accessSecret  []byte
	refreshSecret []byte
}

func NewJWTHandler(userClient userv1.UserServiceClient, rdb redis.Cmdable, accessSecret string, refreshSecret string) *JWTHandler {
	return &JWTHandler{
		userClient:    userClient,
		rdb:           rdb,
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
	}
}

func (h *JWTHandler) Login(ctx *gin.Context, req LoginReq) (ginx.Result, error) {
	resp, err := h.userClient.Login(ctx.Request.Context(), &userv1.LoginRequest{
		TenantId: req.TenantId,
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("调用用户服务登录失败: %w", err)
	}

	err = h.SetTokenHeaders(ctx, resp.User.GetId(), req.TenantId)
	if err != nil {
		return ginx.Result{}, fmt.Errorf("生成 token 失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "登录成功",
	}, nil
}

func (h *JWTHandler) Refresh(ctx *gin.Context, _ RefreshReq) (ginx.Result, error) {
	refreshToken := ctx.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		return ginx.Result{Code: 401001, Msg: "缺少 refresh token"}, nil
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return h.refreshSecret, nil
	})
	if err != nil || !token.Valid {
		return ginx.Result{Code: 401001, Msg: "refresh token 无效或已过期"}, nil
	}

	if claims.ID != "" && h.IsTokenBlacklisted(ctx.Request.Context(), claims.ID) {
		return ginx.Result{Code: 401002, Msg: "token 已失效，请重新登录"}, nil
	}

	err = h.SetTokenHeaders(ctx, claims.Uid, claims.TenantId)
	if err != nil {
		return ginx.Result{}, fmt.Errorf("生成 token 失败: %w", err)
	}

	return ginx.Result{
		Code: 0,
		Msg:  "刷新成功",
	}, nil
}

func (h *JWTHandler) Logout(ctx *gin.Context, _ LogoutReq) (ginx.Result, error) {
	accessToken := extractTokenFromHeader(ctx)
	if accessToken != "" {
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(accessToken, claims, func(t *jwt.Token) (interface{}, error) {
			return h.accessSecret, nil
		})
		if err == nil && token.Valid && claims.ID != "" {
			ttl := time.Until(claims.ExpiresAt.Time)
			if ttl > 0 {
				h.rdb.Set(ctx.Request.Context(), "jwt:blacklist:"+claims.ID, "1", ttl)
			}
		}
	}
	refreshToken := ctx.GetHeader("X-Refresh-Token")
	if refreshToken != "" {
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
			return h.refreshSecret, nil
		})
		if err == nil && token.Valid && claims.ID != "" {
			ttl := time.Until(claims.ExpiresAt.Time)
			if ttl > 0 {
				h.rdb.Set(ctx.Request.Context(), "jwt:blacklist:"+claims.ID, "1", ttl)
			}
		}
	}
	return ginx.Result{Code: 0, Msg: "登出成功"}, nil
}

func (h *JWTHandler) SetTokenHeaders(ctx *gin.Context, uid int64, tenantId int64) error {
	now := time.Now()
	ua := ctx.GetHeader("User-Agent")

	accessClaims := Claims{
		Uid:       uid,
		TenantId:  tenantId,
		UserAgent: ua,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(h.accessSecret)
	if err != nil {
		return err
	}

	refreshClaims := Claims{
		Uid:       uid,
		TenantId:  tenantId,
		UserAgent: ua,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(h.refreshSecret)
	if err != nil {
		return err
	}

	ctx.Header("X-Jwt-Token", accessStr)
	ctx.Header("X-Refresh-Token", refreshStr)
	return nil
}

func (h *JWTHandler) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return h.accessSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token 无效")
	}
	return claims, nil
}

func (h *JWTHandler) IsTokenBlacklisted(ctx context.Context, jti string) bool {
	val, err := h.rdb.Exists(ctx, "jwt:blacklist:"+jti).Result()
	if err != nil {
		return false
	}
	return val > 0
}

func extractTokenFromHeader(ctx *gin.Context) string {
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
		return authHeader[len(prefix):]
	}
	return ""
}
