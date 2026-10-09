// Package monitor 后台定时任务：对每个处于监控中的用户
//   - 每个 keepalive 周期访问一次教务系统，让学校那边的登录不过期；
//   - 距上次完整检查超过 check 周期时，重新拉取课表和成绩，变动由 academic 比对并通知。
//
// 本包只负责调度和记录运行状态，业务逻辑都在 academic。
package monitor

import (
	"context"
	"log/slog"
	"time"

	"jww/internal/academic"
	"jww/internal/store"
)

const userTimeout = 2 * time.Minute

// Monitor 后台监控
type Monitor struct {
	academic  *academic.Service
	store     *store.Store
	keepalive time.Duration
	check     time.Duration
	lastCheck map[string]time.Time // 只在 Run 的 goroutine 中访问
}

// New 创建后台监控
func New(a *academic.Service, st *store.Store, keepalive, check time.Duration) *Monitor {
	return &Monitor{academic: a, store: st, keepalive: keepalive, check: check, lastCheck: make(map[string]time.Time)}
}

// Run 运行直到 ctx 结束；启动 30 秒后执行第一轮
func (m *Monitor) Run(ctx context.Context) {
	slog.Info("monitor started", "keepalive", m.keepalive, "check", m.check)
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		m.Round(ctx)
		timer.Reset(m.keepalive)
	}
}

// Round 对所有监控中的用户执行一轮保活或完整检查
func (m *Monitor) Round(ctx context.Context) {
	uids, err := m.store.MonitoredUsers(ctx)
	if err != nil {
		slog.Warn("load monitored users failed", "err", err)
		return
	}
	active := make(map[string]bool, len(uids))
	for _, uid := range uids {
		if ctx.Err() != nil {
			return
		}
		active[uid] = true
		if time.Since(m.lastCheck[uid]) >= m.check {
			if m.CheckUser(ctx, uid) == nil {
				m.lastCheck[uid] = time.Now()
			}
		} else {
			m.KeepaliveUser(ctx, uid)
		}
	}
	for uid := range m.lastCheck {
		if !active[uid] {
			delete(m.lastCheck, uid)
		}
	}
}

// CheckUser 完整检查一次课表和成绩
func (m *Monitor) CheckUser(ctx context.Context, uid string) error {
	ctx, cancel := context.WithTimeout(ctx, userTimeout)
	defer cancel()
	err := m.academic.RefreshSchedule(ctx, uid)
	if err == nil {
		err = m.academic.RefreshScores(ctx, uid)
	}
	m.record(ctx, uid, store.MonitorCheck, err)
	return err
}

// KeepaliveUser 保活一次
func (m *Monitor) KeepaliveUser(ctx context.Context, uid string) error {
	ctx, cancel := context.WithTimeout(ctx, userTimeout)
	defer cancel()
	err := m.academic.Keepalive(ctx, uid)
	m.record(ctx, uid, store.MonitorKeepalive, err)
	return err
}

func (m *Monitor) record(ctx context.Context, uid, kind string, result error) {
	if result != nil {
		slog.Warn("monitor "+kind+" failed", "uid", uid, "err", result)
	}
	if err := m.store.RecordMonitorResult(ctx, uid, kind, result); err != nil {
		slog.Warn("record monitor status failed", "uid", uid, "err", err)
	}
}

// Status 用户的监控运行情况
type Status struct {
	Monitoring         bool       `json:"monitoring"`
	LastCheck          *time.Time `json:"lastCheck"`
	LastCheckError     string     `json:"lastCheckError"`
	LastKeepalive      *time.Time `json:"lastKeepalive"`
	LastKeepaliveError string     `json:"lastKeepaliveError"`
	KeepaliveMinutes   int        `json:"keepaliveMinutes"`
	CheckMinutes       int        `json:"checkMinutes"`
}

// Status 读取用户的监控运行情况
func (m *Monitor) Status(ctx context.Context, uid string) (*Status, error) {
	monitored, err := m.store.IsMonitored(ctx, uid)
	if err != nil {
		return nil, err
	}
	st, err := m.store.MonitorStatus(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &Status{
		Monitoring: monitored,
		LastCheck:  st.LastCheck, LastCheckError: st.LastCheckError,
		LastKeepalive: st.LastKeepalive, LastKeepaliveError: st.LastKeepaliveError,
		KeepaliveMinutes: int(m.keepalive.Minutes()), CheckMinutes: int(m.check.Minutes()),
	}, nil
}
