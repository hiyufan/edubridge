// Package session 管理教务登录会话（jwclient.Client）的生命周期。
//
//   - 登录前（获取验证码阶段）的会话以前端 sessionID 为键暂存在内存，空闲 30 分钟丢弃；
//   - 登录成功后按学号存放，cookie 持久化到 Redis，并把用户加入后台监控；
//   - 内存中长时间未使用的已登录会话会被淘汰，下次使用时从 Redis 无感恢复。
package session

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"jww/internal/jwclient"
	"jww/internal/store"
)

const idleTTL = 30 * time.Minute

type entry struct {
	client   *jwclient.Client
	lastUsed time.Time
}

// Manager 会话管理器
type Manager struct {
	baseURL string
	store   *store.Store

	mu      sync.Mutex
	pending map[string]*entry // 前端 sessionID -> 未登录会话
	users   map[string]*entry // 学号 -> 已登录会话
}

// New 创建会话管理器
func New(baseURL string, st *store.Store) *Manager {
	return &Manager{
		baseURL: baseURL,
		store:   st,
		pending: make(map[string]*entry),
		users:   make(map[string]*entry),
	}
}

// Pending 获取（或新建）登录前的会话，验证码和登录请求必须使用同一个会话
func (m *Manager) Pending(sessionID string) *jwclient.Client {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.pending[sessionID]
	if !ok {
		e = &entry{client: jwclient.New(m.baseURL)}
		m.pending[sessionID] = e
	}
	e.lastUsed = time.Now()
	return e.client
}

// Bind 登录成功后把会话转为该学号的会话，持久化 cookie 并加入后台监控
func (m *Manager) Bind(ctx context.Context, sessionID, uid string, c *jwclient.Client) error {
	m.mu.Lock()
	delete(m.pending, sessionID)
	m.users[uid] = &entry{client: c, lastUsed: time.Now()}
	m.mu.Unlock()

	c.Jar().TakeDirty()
	if err := m.store.SaveCookies(ctx, uid, c.Jar().Export()); err != nil {
		return err
	}
	return m.store.AddMonitored(ctx, uid)
}

// Get 获取学号对应的已登录会话；没有登录态时返回 jwclient.ErrSessionExpired
func (m *Manager) Get(ctx context.Context, uid string) (*jwclient.Client, error) {
	m.mu.Lock()
	if e, ok := m.users[uid]; ok {
		e.lastUsed = time.Now()
		m.mu.Unlock()
		return e.client, nil
	}
	m.mu.Unlock()

	cookies, err := m.store.Cookies(ctx, uid)
	if errors.Is(err, store.ErrNotFound) {
		return nil, jwclient.ErrSessionExpired
	}
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.users[uid]; ok { // 并发恢复时以先到者为准
		return e.client, nil
	}
	c := jwclient.New(m.baseURL)
	c.Jar().Import(cookies)
	m.users[uid] = &entry{client: c, lastUsed: time.Now()}
	return c, nil
}

// Persist cookie 有变化时写回 Redis
func (m *Manager) Persist(ctx context.Context, uid string, c *jwclient.Client) error {
	if !c.Jar().TakeDirty() {
		return nil
	}
	return m.store.SaveCookies(ctx, uid, c.Jar().Export())
}

// Drop 丢弃学号的登录态并移出后台监控，返回此前是否在监控中
func (m *Manager) Drop(ctx context.Context, uid string) (wasMonitored bool, err error) {
	m.mu.Lock()
	delete(m.users, uid)
	m.mu.Unlock()

	wasMonitored, err = m.store.RemoveMonitored(ctx, uid)
	if delErr := m.store.DeleteCookies(ctx, uid); err == nil {
		err = delErr
	}
	return wasMonitored, err
}

// Run 定期淘汰空闲会话，直到 ctx 结束
func (m *Manager) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.evict(ctx, time.Now().Add(-idleTTL))
		}
	}
}

func (m *Manager) evict(ctx context.Context, before time.Time) {
	m.mu.Lock()
	for id, e := range m.pending {
		if e.lastUsed.Before(before) {
			delete(m.pending, id)
		}
	}
	evicted := make(map[string]*jwclient.Client)
	for uid, e := range m.users {
		if e.lastUsed.Before(before) {
			delete(m.users, uid)
			evicted[uid] = e.client
		}
	}
	m.mu.Unlock()

	for uid, c := range evicted {
		if err := m.Persist(ctx, uid, c); err != nil {
			slog.Warn("persist evicted session failed", "uid", uid, "err", err)
		}
	}
}
