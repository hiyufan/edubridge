// Package auth 签发和校验本系统的登录凭证：短期 access token（2 小时）+ 可轮换的 refresh token（30 天）。
// 凭证里只有学号；教务系统的登录态由 session 包按学号管理。
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"jww/internal/store"
)

// ErrInvalidToken 凭证无效、过期或已撤销
var ErrInvalidToken = errors.New("登录已过期，请重新登录")

const (
	AccessTTL  = 2 * time.Hour
	RefreshTTL = 30 * 24 * time.Hour
)

// Claims JWT 内容
type Claims struct {
	UID     string `json:"uid"`
	TokenID string `json:"tokenId,omitempty"` // 仅 refresh token
	jwt.RegisteredClaims
}

// Service 凭证服务
type Service struct {
	accessSecret  []byte
	refreshSecret []byte
	store         *store.Store
}

// New 创建凭证服务（两个密钥必须不同）
func New(accessSecret, refreshSecret string, st *store.Store) *Service {
	return &Service{accessSecret: []byte(accessSecret), refreshSecret: []byte(refreshSecret), store: st}
}

func sign(c *Claims, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	c.IssuedAt = jwt.NewNumericDate(now)
	c.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(secret)
}

func parse(token string, secret []byte) (*Claims, error) {
	c := &Claims{}
	t, err := jwt.ParseWithClaims(token, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil || !t.Valid || c.UID == "" {
		return nil, ErrInvalidToken
	}
	return c, nil
}

func newTokenID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Issue 登录成功后签发一对凭证
func (s *Service) Issue(ctx context.Context, uid string) (access, refresh string, err error) {
	access, err = sign(&Claims{UID: uid}, s.accessSecret, AccessTTL)
	if err != nil {
		return "", "", err
	}
	id := newTokenID()
	refresh, err = sign(&Claims{UID: uid, TokenID: id}, s.refreshSecret, RefreshTTL)
	if err != nil {
		return "", "", err
	}
	if err := s.store.SaveRefreshToken(ctx, id, uid, time.Now().Add(RefreshTTL)); err != nil {
		return "", "", fmt.Errorf("save refresh token: %w", err)
	}
	return access, refresh, nil
}

// Refresh 用 refresh token 换一对新凭证；旧 refresh token 立即作废
func (s *Service) Refresh(ctx context.Context, refreshToken string) (uid, access, refresh string, err error) {
	c, err := parse(refreshToken, s.refreshSecret)
	if err != nil || c.TokenID == "" {
		return "", "", "", ErrInvalidToken
	}
	owner, err := s.store.RefreshTokenUser(ctx, c.TokenID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && owner != c.UID) {
		return "", "", "", ErrInvalidToken
	}
	if err != nil {
		return "", "", "", err
	}
	if err := s.store.DeleteRefreshToken(ctx, c.TokenID, c.UID); err != nil {
		return "", "", "", err
	}
	access, refresh, err = s.Issue(ctx, c.UID)
	return c.UID, access, refresh, err
}

// Revoke 作废 refresh token（退出登录），返回其学号；无效的 token 直接忽略
func (s *Service) Revoke(ctx context.Context, refreshToken string) (uid string, err error) {
	c, err := parse(refreshToken, s.refreshSecret)
	if err != nil || c.TokenID == "" {
		return "", nil
	}
	return c.UID, s.store.DeleteRefreshToken(ctx, c.TokenID, c.UID)
}

// Verify 校验 access token，返回学号
func (s *Service) Verify(accessToken string) (string, error) {
	c, err := parse(accessToken, s.accessSecret)
	if err != nil {
		return "", err
	}
	return c.UID, nil
}
