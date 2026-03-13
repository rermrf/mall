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
	"github.com/rermrf/mall/pkg/tenantx"
)


type Claims struct {
	Uid       int64  `json:"uid"`
	TenantId  int64  `json:"tenant_id"`
	UserAgent string `json:"user_agent"`
	jwt.RegisteredClaims
}

type LoginReq struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginByPhoneReq struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type OAuthLoginReq struct {
	Provider string `json:"provider"`
	Code     string `json:"code"`
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
	tenantId := tenantx.GetTenantID(ctx.Request.Context())
	resp, err := h.userClient.Login(ctx.Request.Context(), &userv1.LoginRequest{
		TenantId: tenantId,
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "登录失败", ginx.UserErrMappings...)
	}
	err = h.SetTokenHeaders(ctx, resp.User.GetId(), tenantId)
	if err != nil {
		return ginx.Result{}, fmt.Errorf("生成 token 失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "登录成功"}, nil
}

func (h *JWTHandler) LoginByPhone(ctx *gin.Context, req LoginByPhoneReq) (ginx.Result, error) {
	tenantId := tenantx.GetTenantID(ctx.Request.Context())
	resp, err := h.userClient.LoginByPhone(ctx.Request.Context(), &userv1.LoginByPhoneRequest{
		TenantId: tenantId,
		Phone:    req.Phone,
		Code:     req.Code,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "手机登录失败", ginx.UserErrMappings...)
	}
	err = h.SetTokenHeaders(ctx, resp.User.GetId(), tenantId)
	if err != nil {
		return ginx.Result{}, fmt.Errorf("生成 token 失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "登录成功"}, nil
}

func (h *JWTHandler) OAuthLogin(ctx *gin.Context, req OAuthLoginReq) (ginx.Result, error) {
	tenantId := tenantx.GetTenantID(ctx.Request.Context())
	resp, err := h.userClient.OAuthLogin(ctx.Request.Context(), &userv1.OAuthLoginRequest{
		TenantId: tenantId,
		Provider: req.Provider,
		Code:     req.Code,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "第三方登录失败", ginx.UserErrMappings...)
	}
	err = h.SetTokenHeaders(ctx, resp.User.GetId(), tenantId)
	if err != nil {
		return ginx.Result{}, fmt.Errorf("生成 token 失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "登录成功", Data: map[string]any{"is_new": resp.GetIsNew()}}, nil
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
		return ginx.Result{Code: 401001, Msg: "token 已失效"}, nil
	}
	err = h.SetTokenHeaders(ctx, claims.Uid, claims.TenantId)
	if err != nil {
		return ginx.Result{}, fmt.Errorf("生成 token 失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "刷新成功"}, nil
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
		Uid: uid, TenantId: tenantId, UserAgent: ua,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	accessStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(h.accessSecret)
	if err != nil {
		return err
	}

	refreshClaims := Claims{
		Uid: uid, TenantId: tenantId, UserAgent: ua,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	refreshStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(h.refreshSecret)
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
