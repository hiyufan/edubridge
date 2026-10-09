package monitor_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"jww/internal/academic"
	"jww/internal/jwclient"
	"jww/internal/jwclient/jwtest"
	"jww/internal/model"
	"jww/internal/monitor"
	"jww/internal/notify"
	"jww/internal/session"
	"jww/internal/testenv"
)

var ctx = context.Background()

func newMonitor(e *testenv.Env) *monitor.Monitor {
	return monitor.New(e.Academic, e.Store, 10*time.Minute, time.Hour)
}

func TestScheduleChangesAndExpiry(t *testing.T) {
	e := testenv.New(t)
	uid := e.Login(t)
	m := newMonitor(e)

	// 第一次只建立基线
	if err := m.CheckUser(ctx, uid); err != nil {
		t.Fatal(err)
	}
	e.Hook.None(t)

	// 无变化不通知
	m.CheckUser(ctx, uid)
	e.Hook.None(t)

	// 第 3 周停课、第 4 周加课
	e.JW.SetCourses(func(week int) []model.Course {
		list := []model.Course{{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 1}}
		switch week {
		case 3:
			list = nil
		case 4:
			list = append(list, model.Course{Name: "大学物理", Teacher: "李老师", Room: "B202", DayOfWeek: 3, PeriodStart: 1, Periods: 1})
		}
		return list
	})
	if err := m.CheckUser(ctx, uid); err != nil {
		t.Fatal(err)
	}
	p := e.Hook.Next(t)
	if p.Event != notify.EventScheduleDiff ||
		!strings.Contains(p.Text, "+ 大学物理 周三 第1节 @B202（第4周）") ||
		!strings.Contains(p.Text, "- 高等数学 周一 第1节 @A101（第3周）") {
		t.Fatalf("notification = %+v", p)
	}
	if d, err := e.Academic.LatestDiff(ctx, uid); err != nil || len(d.Added) != 1 || len(d.Removed) != 1 {
		t.Fatalf("latest diff = %+v, %v", d, err)
	}

	// 保活正常不通知
	if err := m.KeepaliveUser(ctx, uid); err != nil {
		t.Fatal(err)
	}
	e.Hook.None(t)

	// 学校踢下线：保活发现后停止监控并通知一次
	e.JW.KickAll()
	if err := m.KeepaliveUser(ctx, uid); !errors.Is(err, jwclient.ErrSessionExpired) {
		t.Fatalf("keepalive: %v", err)
	}
	if p := e.Hook.Next(t); p.Event != notify.EventSessionExpired {
		t.Fatalf("event = %q", p.Event)
	}
	if ok, _ := e.Store.IsMonitored(ctx, uid); ok {
		t.Fatal("should stop monitoring")
	}
	if _, err := e.Academic.FullSchedule(ctx, uid, 20); !errors.Is(err, jwclient.ErrSessionExpired) {
		t.Fatalf("after expiry: %v", err)
	}
	m.KeepaliveUser(ctx, uid)
	e.Hook.None(t) // 不重复通知

	status, _ := e.Store.MonitorStatus(ctx, uid)
	history, _ := e.Store.History(ctx, uid, 10)
	if status.LastCheck == nil || status.LastCheckError != "" || status.LastKeepaliveError == "" {
		t.Fatalf("status = %+v", status)
	}
	if len(history) != 2 || history[0].Event != notify.EventSessionExpired || history[1].Event != notify.EventScheduleDiff {
		t.Fatalf("history = %+v", history)
	}

	// 重新登录后自动恢复监控
	e.Login(t)
	if ok, _ := e.Store.IsMonitored(ctx, uid); !ok {
		t.Fatal("should resume monitoring after login")
	}
}

func TestIncompleteFetchDoesNotNotify(t *testing.T) {
	e := testenv.New(t)
	uid := e.Login(t)
	m := newMonitor(e)
	m.CheckUser(ctx, uid)

	e.JW.FailWeek(5) // 第 5 周拉取失败，不能当成停课
	m.CheckUser(ctx, uid)
	e.Hook.None(t)

	e.JW.FailWeek(0)
	m.CheckUser(ctx, uid)
	e.Hook.None(t)
}

func TestNewScores(t *testing.T) {
	e := testenv.New(t)
	var rows []map[string]any
	for i := 0; i < 12; i++ {
		rows = append(rows, jwtest.ScoreRow(fmt.Sprintf("课程%d", i), "80", "3.0", "2"))
	}
	e.JW.SetScores(append(rows, jwtest.ScoreRow("体育", "", "", "1")))
	uid := e.Login(t)
	m := newMonitor(e)

	m.CheckUser(ctx, uid)
	e.Hook.None(t)

	rows[3] = jwtest.ScoreRow("课程3", "85", "3.5", "2")
	e.JW.SetScores(append(rows, jwtest.ScoreRow("体育", "优秀", "4.0", "1"), jwtest.ScoreRow("高等数学", "92", "4.2", "4")))
	m.CheckUser(ctx, uid)
	p := e.Hook.Next(t)
	for _, want := range []string{"《高等数学》92（绩点 4.2，学分 4）", "《体育》优秀", "《课程3》85（绩点 3.5，学分 2） [由 80 修改]"} {
		if p.Event != notify.EventScoreNew || !strings.Contains(p.Text, want) {
			t.Errorf("missing %q in %+v", want, p)
		}
	}
	m.CheckUser(ctx, uid)
	e.Hook.None(t)
}

func TestIncompleteScoreBaseline(t *testing.T) {
	e := testenv.New(t)
	var rows []map[string]any
	for i := 0; i < 20; i++ {
		rows = append(rows, jwtest.ScoreRow(fmt.Sprintf("课程%d", i), "80", "3.0", "2"))
	}
	e.JW.SetScores(rows)
	uid := e.Login(t)
	m := newMonitor(e)

	e.JW.FailScorePage(2)
	m.CheckUser(ctx, uid)
	e.JW.FailScorePage(0)
	m.CheckUser(ctx, uid) // 第 2 页的老成绩不是新成绩
	e.Hook.None(t)

	e.JW.SetScores(append(rows, jwtest.ScoreRow("新课", "90", "4.0", "3")))
	m.CheckUser(ctx, uid)
	if p := e.Hook.Next(t); !strings.Contains(p.Text, "《新课》90") {
		t.Fatalf("text = %s", p.Text)
	}
}

func TestRoundAndRestart(t *testing.T) {
	e := testenv.New(t)
	uid := e.Login(t)

	// 模拟服务重启：新的会话管理器和业务服务，只靠 Redis 恢复
	sessions := session.New(e.JW.URL, e.Store)
	a := academic.New(sessions, e.Store, e.Notifier)
	m := monitor.New(a, e.Store, 10*time.Minute, time.Hour)

	before := e.JW.Requests()
	m.Round(ctx) // 第一轮做完整检查
	full := e.JW.Requests() - before
	m.Round(ctx) // 第二轮只保活
	if keep := e.JW.Requests() - before - full; full < 20 || keep != 1 {
		t.Fatalf("requests: check=%d keepalive=%d", full, keep)
	}
	if status, _ := e.Store.MonitorStatus(ctx, uid); status.LastCheck == nil || status.LastKeepalive == nil {
		t.Fatalf("status = %+v", status)
	}
}
