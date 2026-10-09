// Package store 封装所有 Redis 读写。上层只通过这里的方法访问持久化数据，
// key 命名集中在本包内，方便查找和迁移。
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound 数据不存在或已过期
var ErrNotFound = errors.New("not found")

// Store Redis 存储
type Store struct {
	rdb *redis.Client
}

// New 包装已有的 Redis 客户端（测试中可传入 miniredis）
func New(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

// Open 连接 Redis 并检查连通性
func Open(ctx context.Context, addr, password string, db int) (*Store, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	return New(rdb), nil
}

// Close 关闭连接
func (s *Store) Close() error {
	return s.rdb.Close()
}

func notFound(err error) error {
	if errors.Is(err, redis.Nil) {
		return ErrNotFound
	}
	return err
}

func (s *Store) getJSON(ctx context.Context, key string, v any) error {
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return notFound(err)
	}
	return json.Unmarshal(data, v)
}

func (s *Store) setJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, key, data, ttl).Err()
}
