package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// ---- iCal 订阅 token ----

// SaveICalToken 保存用户的 iCal token，同一用户只保留最新一个
func (s *Store) SaveICalToken(ctx context.Context, uid, token string, ttl time.Duration) error {
	if old, err := s.rdb.Get(ctx, keyICalUser+uid).Result(); err == nil {
		s.rdb.Del(ctx, keyICalToken+old)
	}
	_, err := s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Set(ctx, keyICalToken+token, uid, ttl)
		p.Set(ctx, keyICalUser+uid, token, ttl)
		return nil
	})
	return err
}

// ICalTokenUser 由 token 查学号
func (s *Store) ICalTokenUser(ctx context.Context, token string) (string, error) {
	uid, err := s.rdb.Get(ctx, keyICalToken+token).Result()
	return uid, notFound(err)
}

// UserICalToken 用户当前的 iCal token 及过期时间
func (s *Store) UserICalToken(ctx context.Context, uid string) (string, time.Time, error) {
	token, err := s.rdb.Get(ctx, keyICalUser+uid).Result()
	if err != nil {
		return "", time.Time{}, notFound(err)
	}
	ttl, err := s.rdb.TTL(ctx, keyICalUser+uid).Result()
	if err != nil {
		return "", time.Time{}, err
	}
	return token, time.Now().Add(ttl), nil
}

// ---- 登录 refresh token ----

// SaveRefreshToken 记录一个有效的 refresh token
func (s *Store) SaveRefreshToken(ctx context.Context, tokenID, uid string, expiresAt time.Time) error {
	_, err := s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.HSet(ctx, keyRefreshToken+tokenID, "user_id", uid, "expires_at", expiresAt.Unix())
		p.ExpireAt(ctx, keyRefreshToken+tokenID, expiresAt)
		p.SAdd(ctx, keyRefreshByUser+uid, tokenID)
		p.ExpireAt(ctx, keyRefreshByUser+uid, expiresAt)
		return nil
	})
	return err
}

// RefreshTokenUser refresh token 对应的学号；已撤销或过期返回 ErrNotFound
func (s *Store) RefreshTokenUser(ctx context.Context, tokenID string) (string, error) {
	uid, err := s.rdb.HGet(ctx, keyRefreshToken+tokenID, "user_id").Result()
	return uid, notFound(err)
}

// DeleteRefreshToken 撤销一个 refresh token
func (s *Store) DeleteRefreshToken(ctx context.Context, tokenID, uid string) error {
	_, err := s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Del(ctx, keyRefreshToken+tokenID)
		p.SRem(ctx, keyRefreshByUser+uid, tokenID)
		return nil
	})
	return err
}
