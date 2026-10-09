package store

import (
	"context"
	"net/http"
)

// SaveCookies 保存教务登录 cookie（不过期，失效时由上层删除）
func (s *Store) SaveCookies(ctx context.Context, uid string, cookies map[string][]*http.Cookie) error {
	return s.setJSON(ctx, keyCookies+uid, cookies, 0)
}

// Cookies 读取教务登录 cookie
func (s *Store) Cookies(ctx context.Context, uid string) (map[string][]*http.Cookie, error) {
	var cookies map[string][]*http.Cookie
	err := s.getJSON(ctx, keyCookies+uid, &cookies)
	return cookies, err
}

// DeleteCookies 删除教务登录 cookie
func (s *Store) DeleteCookies(ctx context.Context, uid string) error {
	return s.rdb.Del(ctx, keyCookies+uid).Err()
}

// AddMonitored 把用户加入后台监控
func (s *Store) AddMonitored(ctx context.Context, uid string) error {
	return s.rdb.SAdd(ctx, keyMonitoredUsers, uid).Err()
}

// RemoveMonitored 移出后台监控，返回此前是否在监控中
func (s *Store) RemoveMonitored(ctx context.Context, uid string) (bool, error) {
	n, err := s.rdb.SRem(ctx, keyMonitoredUsers, uid).Result()
	return n > 0, err
}

// MonitoredUsers 所有处于监控中的学号
func (s *Store) MonitoredUsers(ctx context.Context) ([]string, error) {
	return s.rdb.SMembers(ctx, keyMonitoredUsers).Result()
}

// IsMonitored 用户是否处于监控中
func (s *Store) IsMonitored(ctx context.Context, uid string) (bool, error) {
	return s.rdb.SIsMember(ctx, keyMonitoredUsers, uid).Result()
}
