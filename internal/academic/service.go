// Package academic 业务用例层：登录、查询课表/成绩（带缓存）、与快照比对并发出通知、
// 处理教务登录失效。HTTP 层和后台任务都只通过这里访问教务数据。
package academic

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"jww/internal/jwclient"
	"jww/internal/notify"
	"jww/internal/session"
	"jww/internal/store"
)

// MaxWeek 一个学期最多的教学周
const MaxWeek = 20

const (
	scheduleCacheTTL = 5 * time.Minute
	scoreCacheTTL    = 15 * time.Minute
)

// Service 教务业务服务
type Service struct {
	sessions *session.Manager
	store    *store.Store
	notifier *notify.Notifier
	now      func() time.Time

	schedules cache[scheduleEntry]
	scores    cache[scoresEntry]
	recordMu  sync.Mutex // 串行化“读快照-比对-写快照”，避免重复通知
}

// New 创建业务服务
func New(sessions *session.Manager, st *store.Store, notifier *notify.Notifier) *Service {
	return &Service{sessions: sessions, store: st, notifier: notifier, now: time.Now}
}

// client 取得用户的教务会话
func (s *Service) client(ctx context.Context, uid string) (*jwclient.Client, error) {
	c, err := s.sessions.Get(ctx, uid)
	return c, s.handle(ctx, uid, err)
}

// done 一次教务请求结束：成功则持久化可能变化的 cookie，失败则统一处理
func (s *Service) done(ctx context.Context, uid string, c *jwclient.Client, err error) error {
	if err == nil {
		if perr := s.sessions.Persist(ctx, uid, c); perr != nil {
			slog.Warn("persist cookies failed", "uid", uid, "err", perr)
		}
		return nil
	}
	return s.handle(ctx, uid, err)
}

// handle 教务登录失效时清理登录态、停止监控，并通知用户重新登录（只通知一次）
func (s *Service) handle(ctx context.Context, uid string, err error) error {
	if !errors.Is(err, jwclient.ErrSessionExpired) {
		return err
	}
	s.schedules.delete(uid)
	s.scores.delete(uid)
	wasMonitored, dropErr := s.sessions.Drop(ctx, uid)
	if dropErr != nil {
		slog.Warn("drop session failed", "uid", uid, "err", dropErr)
	}
	if wasMonitored {
		slog.Info("jw session expired", "uid", uid)
		s.notifier.Notify(uid, notify.EventSessionExpired, "教务系统登录已失效，课表监控已暂停，请打开应用重新登录。", nil)
	}
	return err
}

// Captcha 获取登录验证码（sessionID 由前端保存，登录时带回）
func (s *Service) Captcha(ctx context.Context, sessionID string) ([]byte, string, error) {
	return s.sessions.Pending(sessionID).Captcha(ctx)
}

// Login 登录教务系统；成功后登录态持久化并开始后台监控
func (s *Service) Login(ctx context.Context, sessionID, username, password, captcha, loginType string) error {
	c := s.sessions.Pending(sessionID)
	if err := c.Login(ctx, username, password, captcha, loginType); err != nil {
		return err
	}
	return s.sessions.Bind(ctx, sessionID, username, c)
}

// Logout 用户主动退出：清理登录态并停止监控（不发通知）
func (s *Service) Logout(ctx context.Context, uid string) error {
	s.schedules.delete(uid)
	s.scores.delete(uid)
	_, err := s.sessions.Drop(ctx, uid)
	return err
}

// Keepalive 访问一次教务系统，保持学校那边的会话活跃
func (s *Service) Keepalive(ctx context.Context, uid string) error {
	c, err := s.client(ctx, uid)
	if err != nil {
		return err
	}
	return s.done(ctx, uid, c, c.Ping(ctx))
}

// cache 按学号缓存、带过期时间的简单缓存
type cache[T any] struct {
	mu    sync.Mutex
	items map[string]cached[T]
}

type cached[T any] struct {
	value  T
	expire time.Time
}

func (c *cache[T]) get(uid string, now time.Time) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.items[uid]
	if !ok || now.After(it.expire) {
		var zero T
		return zero, false
	}
	return it.value, true
}

func (c *cache[T]) set(uid string, v T, expire time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil {
		c.items = make(map[string]cached[T])
	}
	c.items[uid] = cached[T]{value: v, expire: expire}
}

func (c *cache[T]) delete(uid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, uid)
}
