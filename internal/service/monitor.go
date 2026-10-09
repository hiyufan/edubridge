package service

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// 后台课表监控：
//   - 每个 keepalive 周期访问一次教务系统，让学校那边的登录会话不过期（这样用户不用反复输验证码）；
//   - 距上次完整检查超过 check 周期时，拉取完整课表并与快照比对，有加课/减课就推送通知；
//   - 发现教务登录已失效时停止监控该用户，并通知其重新登录。

const monitorMaxWeek = 20

// StartMonitor 启动后台监控（随 Close 停止）
func (s *JwService) StartMonitor(keepalive, check time.Duration) {
	go func() {
		lastCheck := make(map[string]time.Time)
		timer := time.NewTimer(30 * time.Second) // 启动后稍等再跑第一轮
		defer timer.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-timer.C:
			}
			s.monitorRound(lastCheck, check)
			timer.Reset(keepalive)
		}
	}()
	slog.Info("Schedule monitor started", "keepalive", keepalive, "check", check)
}

func (s *JwService) monitorRound(lastCheck map[string]time.Time, check time.Duration) {
	uids, err := monitorUsers()
	if err != nil {
		slog.Warn("Load monitor users failed", "err", err)
		return
	}
	active := make(map[string]bool, len(uids))
	for _, uid := range uids {
		active[uid] = true
		select {
		case <-s.stopCh:
			return
		default:
		}

		if time.Since(lastCheck[uid]) >= check {
			if s.checkUser(uid) {
				lastCheck[uid] = time.Now()
			}
		} else {
			s.keepaliveUser(uid)
		}
	}
	for uid := range lastCheck {
		if !active[uid] {
			delete(lastCheck, uid)
		}
	}
}

// checkUser 完整检查一次课表，返回是否成功
func (s *JwService) checkUser(uid string) bool {
	session, err := s.userSession(uid)
	if err != nil {
		s.handleMonitorErr(uid, err)
		return false
	}
	if _, err := s.refreshSchedule(session, monitorMaxWeek); err != nil {
		s.handleMonitorErr(uid, err)
		return false
	}
	return true
}

// keepaliveUser 轻量访问一次教务系统保持会话
func (s *JwService) keepaliveUser(uid string) {
	session, err := s.userSession(uid)
	if err != nil {
		s.handleMonitorErr(uid, err)
		return
	}
	if err := s.ping(session); err != nil {
		if errors.Is(err, ErrSessionExpired) {
			s.expireUser(uid)
		}
		s.handleMonitorErr(uid, err)
		return
	}
	s.persistCookies(session)
}

func (s *JwService) handleMonitorErr(uid string, err error) {
	if errors.Is(err, ErrSessionExpired) {
		// cookie 已不存在（例如在别处被清理）：确保不再监控
		if _, rmErr := removeMonitorUser(uid); rmErr != nil {
			slog.Warn("Remove monitor user failed", "uid", uid, "err", rmErr)
		}
		return
	}
	slog.Warn("Monitor user failed", "uid", uid, "err", err)
}

// ping 访问学生门户首页，检测并保持登录态
func (s *JwService) ping(session *Session) error {
	req, err := http.NewRequest("GET", baseURL+"/studentportal.php/Main/", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,*/*")
	req.Header.Set("Referer", baseURL+"/")

	resp, err := session.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if isLoginPage(resp.Request.URL.String(), string(body)) {
		return ErrSessionExpired
	}
	return nil
}
