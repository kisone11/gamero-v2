// Package auth 提供 JWT Access Token 和 Refresh Token 的生成、解析功能。
// Access Token 有效期 2 小时，Refresh Token 有效期 7 天。
// 支持通过 Redis 维护 Token 黑名单，实现强制登出功能。
package auth

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/pkg/cache"
	"github.com/gamero/gamero/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// blacklistKeyPrefix Token 黑名单 Redis key 前缀
	blacklistKeyPrefix = "token:blacklist:"
	// tokenTypeAccess Access Token 类型标识
	tokenTypeAccess = "access"
	// tokenTypeRefresh Refresh Token 类型标识
	tokenTypeRefresh = "refresh"
)

// Claims JWT 自定义 Claims
type Claims struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Type     string `json:"type"` // "access" 或 "refresh"
	jwt.RegisteredClaims
}

// TokenPair Access Token 和 Refresh Token 对
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"` // Access Token 过期时间
}

// Manager JWT 管理器
type Manager struct {
	cfg config.JWTConfig
}

// global 是全局 JWT 管理器
var global *Manager

// Init 初始化全局 JWT 管理器
func Init(cfg config.JWTConfig) *Manager {
	// 密鑰为空时快速失败：密鑰为空等价于无验证，容许任意伪造 JWT
	if cfg.AccessSecret == "" {
		panic("auth: jwt.access_secret 未配置，拒绝展开（空密鑰可伪造任意 token）")
	}
	if cfg.RefreshSecret == "" {
		panic("auth: jwt.refresh_secret 未配置，拒绝展开")
	}
	// 防止使用配置模板常量直接部署到生产环境
	if strings.Contains(cfg.AccessSecret, "change-in-production") {
		panic("auth: jwt.access_secret 使用了占位符默认密鑰，生产环境必须替换（openssl rand -base64 64）")
	}
	global = &Manager{cfg: cfg}
	return global
}

// Get 返回全局 JWT 管理器
func Get() *Manager {
	if global == nil {
		panic("JWT 未初始化，请先调用 auth.Init()")
	}
	return global
}

// GenerateTokenPair 生成 Access Token 和 Refresh Token 对
func (m *Manager) GenerateTokenPair(userID uint64, username, role string) (*TokenPair, error) {
	now := time.Now()

	// 生成 Access Token
	accessExpire := now.Add(m.cfg.AccessExpireDuration())
	accessClaims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		Type:     tokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpire),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "gamero",
			Subject:   fmt.Sprintf("%d", userID),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(m.cfg.AccessSecret))
	if err != nil {
		return nil, fmt.Errorf("生成 Access Token 失败: %w", err)
	}

	// 生成 Refresh Token
	refreshExpire := now.Add(m.cfg.RefreshExpireDuration())
	refreshClaims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		Type:     tokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpire),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "gamero",
			Subject:   fmt.Sprintf("%d", userID),
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(m.cfg.RefreshSecret))
	if err != nil {
		return nil, fmt.Errorf("生成 Refresh Token 失败: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpire,
	}, nil
}

// ParseAccessToken 解析并验证 Access Token
func (m *Manager) ParseAccessToken(tokenStr string) (*Claims, error) {
	return m.parseToken(tokenStr, m.cfg.AccessSecret, tokenTypeAccess)
}

// ParseRefreshToken 解析并验证 Refresh Token
func (m *Manager) ParseRefreshToken(tokenStr string) (*Claims, error) {
	return m.parseToken(tokenStr, m.cfg.RefreshSecret, tokenTypeRefresh)
}

// parseToken 解析并验证 Token
func (m *Manager) parseToken(tokenStr, secret, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非法的签名算法: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors2IsExpired(err) {
			return nil, errors.CodeError(errors.CodeTokenExpired)
		}
		return nil, errors.CodeError(errors.CodeTokenInvalid)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.CodeError(errors.CodeTokenInvalid)
	}

	// 验证 token 类型
	if claims.Type != expectedType {
		return nil, errors.CodeError(errors.CodeTokenInvalid)
	}

	return claims, nil
}

// RevokeToken 将 Token 加入黑名单（强制登出）
func (m *Manager) RevokeToken(ctx context.Context, tokenStr string, expireAt time.Time) error {
	ttl := time.Until(expireAt)
	if ttl <= 0 {
		// Token 已过期，无需加入黑名单
		return nil
	}

	key := blacklistKeyPrefix + tokenStr
	return cache.Set(ctx, key, "1", ttl)
}

// IsTokenRevoked 检查 Token 是否在黑名单中
func (m *Manager) IsTokenRevoked(ctx context.Context, tokenStr string) (bool, error) {
	key := blacklistKeyPrefix + tokenStr
	return cache.Exists(ctx, key)
}

// errors2IsExpired 判断 jwt 错误是否为 Token 过期
func errors2IsExpired(err error) bool {
	if err == nil {
		return false
	}
	return stderrors.Is(err, jwt.ErrTokenExpired)
}

// ===== 包级别便捷函数 =====

// GenerateTokenPair 使用全局管理器生成 Token 对
func GenerateTokenPair(userID uint64, username, role string) (*TokenPair, error) {
	return Get().GenerateTokenPair(userID, username, role)
}

// ParseAccessToken 使用全局管理器解析 Access Token
func ParseAccessToken(tokenStr string) (*Claims, error) {
	return Get().ParseAccessToken(tokenStr)
}

// ParseRefreshToken 使用全局管理器解析 Refresh Token
func ParseRefreshToken(tokenStr string) (*Claims, error) {
	return Get().ParseRefreshToken(tokenStr)
}

// RevokeToken 使用全局管理器吊销 Token
func RevokeToken(ctx context.Context, tokenStr string, expireAt time.Time) error {
	return Get().RevokeToken(ctx, tokenStr, expireAt)
}

// IsTokenRevoked 使用全局管理器检查 Token 是否已吊销
func IsTokenRevoked(ctx context.Context, tokenStr string) (bool, error) {
	return Get().IsTokenRevoked(ctx, tokenStr)
}
