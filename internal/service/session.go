package service

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// ErrSessionExpired 教务系统登录态已失效（需要用户重新输入验证码登录）
var ErrSessionExpired = errors.New("教务系统登录已失效，请重新登录")

// 会话模型：
//   - 登录前（获取验证码阶段）的会话按前端 sessionID 存在 s.sessions，30 分钟过期；
//   - 登录成功后会话转移到 s.users，按学号存放，cookie 持久化到 Redis，
//     sessionID -> 学号 的映射也写入 Redis，因此服务重启、内存淘汰后都能无感恢复；
//   - 后台监控（monitor.go）定时访问教务系统，让学校那边的会话保持活跃。

func newSession() *Session {
	cookieJar := &jar{}
	client := resty.New()
	client.SetBaseURL(baseURL)
	client.SetTimeout(20 * time.Second)
	client.SetCookieJar(cookieJar)
	client.SetHeader("User-Agent", userAgent)
	client.SetHeader("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	client.SetHeader("Connection", "keep-alive")
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(10))

	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 20 * time.Second,
		},
		Timeout: 20 * time.Second,
		Jar:     cookieJar,
	}

	return &Session{
		Client:     client,
		HttpClient: httpClient,
		CookieJar:  cookieJar,
		ExpireTime: time.Now().Add(serviceSessionTTL),
	}
}

// getSession 获取或创建登录前会话（验证码、登录请求使用）
func (s *JwService) getSession(sessionID string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists || time.Now().After(session.ExpireTime) {
		session = newSession()
		s.sessions[sessionID] = session
	}
	return session
}

// bindLogin 登录成功后把会话转为按学号存放，并持久化
func (s *JwService) bindLogin(sessionID, uid string, session *Session) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	session.UID = uid
	session.ExpireTime = time.Now().Add(serviceSessionTTL)
	s.users[uid] = session
	s.sidUID[sessionID] = uid
	s.mu.Unlock()

	if err := saveJwCookies(uid, session.CookieJar.Export()); err != nil {
		slog.Error("Save jw cookies failed", "uid", uid, "err", err)
	}
	if err := bindSessionID(sessionID, uid); err != nil {
		slog.Error("Bind sessionID failed", "uid", uid, "err", err)
	}
	if err := addMonitorUser(uid); err != nil {
		slog.Error("Add monitor user failed", "uid", uid, "err", err)
	}
}

// resolveUID 由前端 sessionID 找到学号（内存 -> Redis）
func (s *JwService) resolveUID(sessionID string) (string, error) {
	s.mu.RLock()
	uid, ok := s.sidUID[sessionID]
	s.mu.RUnlock()
	if ok {
		return uid, nil
	}

	uid, err := lookupSessionID(sessionID)
	if errors.Is(err, ErrNotFound) {
		return "", ErrSessionExpired
	}
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.sidUID[sessionID] = uid
	s.mu.Unlock()
	return uid, nil
}

// userSession 获取学号对应的已登录会话，内存中没有则从 Redis 恢复 cookie
func (s *JwService) userSession(uid string) (*Session, error) {
	s.mu.Lock()
	session, ok := s.users[uid]
	if ok {
		session.ExpireTime = time.Now().Add(serviceSessionTTL)
		s.mu.Unlock()
		return session, nil
	}
	s.mu.Unlock()

	cookies, err := loadJwCookies(uid)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrSessionExpired
	}
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.users[uid]; ok { // 并发恢复时以先到者为准
		return session, nil
	}
	session = newSession()
	session.UID = uid
	session.CookieJar.Import(cookies)
	s.users[uid] = session
	return session, nil
}

// checkSession 由前端 sessionID 获取已登录会话
func (s *JwService) checkSession(sessionID string) (*Session, error) {
	uid, err := s.resolveUID(sessionID)
	if err != nil {
		return nil, err
	}
	return s.userSession(uid)
}

// UIDForSession 由前端 sessionID 获取学号
func (s *JwService) UIDForSession(sessionID string) (string, error) {
	return s.resolveUID(sessionID)
}

// persistCookies cookie 有变化时写回 Redis
func (s *JwService) persistCookies(session *Session) {
	if session.UID == "" || !session.CookieJar.TakeDirty() {
		return
	}
	if err := saveJwCookies(session.UID, session.CookieJar.Export()); err != nil {
		slog.Warn("Persist jw cookies failed", "uid", session.UID, "err", err)
	}
}

// expireUser 教务登录态失效：清理会话、停止监控，并通知用户重新登录
func (s *JwService) expireUser(uid string) {
	s.mu.Lock()
	delete(s.users, uid)
	delete(s.scheduleCache, uid)
	for sid, u := range s.sidUID {
		if u == uid {
			delete(s.sidUID, sid)
		}
	}
	s.mu.Unlock()

	wasMonitored, err := removeMonitorUser(uid)
	if err != nil {
		slog.Warn("Remove monitor user failed", "uid", uid, "err", err)
	}
	if err := deleteJwCookies(uid); err != nil {
		slog.Warn("Delete jw cookies failed", "uid", uid, "err", err)
	}
	if wasMonitored {
		slog.Info("Jw session expired", "uid", uid)
		notifyUser(uid, EventSessionExpired, "教务系统登录已失效，课表监控已暂停，请打开应用重新登录。", nil)
	}
}

// Logout 用户主动退出：清理会话并停止监控（不发通知）
func (s *JwService) Logout(uid string) {
	if _, err := removeMonitorUser(uid); err != nil {
		slog.Warn("Remove monitor user failed", "uid", uid, "err", err)
	}
	s.expireUser(uid)
}

// isLoginPage 判断教务系统返回的是否是登录页（会话已失效）
func isLoginPage(finalURL, body string) bool {
	if strings.Contains(finalURL, "/Index/login") || strings.HasSuffix(strings.TrimRight(finalURL, "/"), "/studentportal.php") {
		return true
	}
	return strings.Contains(body, "Index/checkLogin") || strings.Contains(body, "Public/verify")
}

// cleanup 定期把长时间未使用的会话移出内存（已登录会话仍在 Redis，可随时恢复）
func (s *JwService) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, session := range s.sessions {
				if now.After(session.ExpireTime) {
					delete(s.sessions, id)
				}
			}
			var evicted []*Session
			for uid, session := range s.users {
				if now.After(session.ExpireTime) {
					delete(s.users, uid)
					evicted = append(evicted, session)
				}
			}
			for sid, uid := range s.sidUID {
				if _, ok := s.users[uid]; !ok {
					delete(s.sidUID, sid)
				}
			}
			for id, cache := range s.scheduleCache {
				if now.After(cache.Expire) {
					delete(s.scheduleCache, id)
				}
			}
			s.mu.Unlock()

			for _, session := range evicted {
				s.persistCookies(session)
			}
		}
	}
}
