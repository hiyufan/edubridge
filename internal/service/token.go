package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"jww/internal/model"
	"jww/pkg/database"
)

type TokenService struct {
	db    *gorm.DB
	redis *redis.Client
}

var tokenService *TokenService

func GetTokenService() *TokenService {
	if tokenService == nil {
		tokenService = &TokenService{
			db:    database.GetDB(),
			redis: database.GetRedis(),
		}
	}
	return tokenService
}

const refreshTokenPrefix = "refresh_token:"
const refreshTokenUserKey = "refresh_token:user:"

func (s *TokenService) StoreRefreshToken(userID, tokenID, device, userAgent string, expiresAt time.Time) error {
	ctx := context.Background()

	tokenData := map[string]interface{}{
		"user_id":    userID,
		"token_id":   tokenID,
		"device":     device,
		"user_agent": userAgent,
		"expires_at": expiresAt.Unix(),
	}

	err := s.redis.HSet(ctx, refreshTokenPrefix+tokenID, tokenData).Err()
	if err != nil {
		return err
	}

	err = s.redis.ExpireAt(ctx, refreshTokenPrefix+tokenID, expiresAt).Err()
	if err != nil {
		return err
	}

	err = s.redis.SAdd(ctx, refreshTokenUserKey+userID, tokenID).Err()
	if err != nil {
		return err
	}
	s.redis.ExpireAt(ctx, refreshTokenUserKey+userID, expiresAt)

	dbToken := &model.RefreshToken{
		UserID:    userID,
		TokenID:   tokenID,
		Device:    device,
		UserAgent: userAgent,
		ExpiresAt: expiresAt,
	}
	if err := s.db.Create(dbToken).Error; err != nil {
		return err
	}

	return nil
}

func (s *TokenService) ValidateRefreshToken(tokenID string) (userID string, err error) {
	ctx := context.Background()

	exists, err := s.redis.Exists(ctx, refreshTokenPrefix+tokenID).Result()
	if err != nil {
		return "", err
	}

	if exists == 0 {
		var dbToken model.RefreshToken
		if err := s.db.Where("token_id = ? AND revoked = ? AND expires_at > ?", tokenID, false, time.Now()).First(&dbToken).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", errors.New("token not found or expired")
			}
			return "", err
		}
		return dbToken.UserID, nil
	}

	userID, err = s.redis.HGet(ctx, refreshTokenPrefix+tokenID, "user_id").Result()
	if err != nil {
		return "", err
	}

	expiresAtStr, err := s.redis.HGet(ctx, refreshTokenPrefix+tokenID, "expires_at").Result()
	if err != nil {
		return "", err
	}

	var expiresAt int64
	fmt.Sscanf(expiresAtStr, "%d", &expiresAt)
	if time.Now().Unix() > expiresAt {
		s.RevokeRefreshToken(tokenID, userID)
		return "", errors.New("token expired")
	}

	return userID, nil
}

func (s *TokenService) RevokeRefreshToken(tokenID, userID string) error {
	ctx := context.Background()

	if err := s.redis.Del(ctx, refreshTokenPrefix+tokenID).Err(); err != nil {
		return err
	}

	if userID != "" {
		s.redis.SRem(ctx, refreshTokenUserKey+userID, tokenID)
	}

	if err := s.db.Model(&model.RefreshToken{}).Where("token_id = ?", tokenID).Update("revoked", true).Error; err != nil {
		return err
	}

	return nil
}

func (s *TokenService) RevokeAllUserTokens(userID string) error {
	ctx := context.Background()

	tokenIDs, err := s.redis.SMembers(ctx, refreshTokenUserKey+userID).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	for _, tokenID := range tokenIDs {
		s.redis.Del(ctx, refreshTokenPrefix+tokenID)
	}

	s.redis.Del(ctx, refreshTokenUserKey+userID)

	if err := s.db.Model(&model.RefreshToken{}).Where("user_id = ?", userID).Update("revoked", true).Error; err != nil {
		return err
	}

	return nil
}

func (s *TokenService) RotateRefreshToken(oldTokenID, userID, device, userAgent string, expiresAt time.Time) (newTokenID string, err error) {
	newTokenID = fmt.Sprintf("%s_%d", userID, time.Now().UnixNano())

	if err := s.StoreRefreshToken(userID, newTokenID, device, userAgent, expiresAt); err != nil {
		return "", err
	}

	s.RevokeRefreshToken(oldTokenID, userID)

	return newTokenID, nil
}
