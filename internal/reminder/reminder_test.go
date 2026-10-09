package reminder_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"jww/internal/model"
	"jww/internal/notify"
	"jww/internal/reminder"
	"jww/internal/testenv"
	"jww/internal/timetable"
)

var ctx = context.Background()

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, timetable.CST)
	if err != nil {
		panic(err)
	}
	return t
}

func setup(t *testing.T, settings model.ReminderSettings) (*reminder.Scheduler, *testenv.Webhook) {
	t.Helper()
	e := testenv.New(t)
	const uid = "u1"
	e.SubscribeHook(t, uid)
	weeks := make([]int, 16)
	for i := range weeks {
		weeks[i] = i + 1
	}
	// 2026-09-07 是周一，第 1 周
	e.Store.SaveSnapshot(ctx, uid, &model.FullSchedule{SemesterStart: "2026-09-07", Courses: []model.Course{
		{Name: "大学英语", Teacher: "赵老师", Room: "C301", DayOfWeek: 1, PeriodStart: 3, Periods: 2, Weeks: weeks},
		{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 2, Weeks: weeks},
		{Name: "体育", DayOfWeek: 2, PeriodStart: 5, Periods: 2, Weeks: []int{1}},
	}}, true)
	s := reminder.New(e.Store, e.Notifier)
	if err := s.Save(ctx, uid, &settings); err != nil {
		t.Fatal(err)
	}
	return s, e.Hook
}

func TestDaily(t *testing.T) {
	s, hook := setup(t, model.ReminderSettings{Daily: true, DailyTime: "07:00"})

	s.Tick(ctx, at("2026-09-14 06:59"))
	hook.None(t)
	s.Tick(ctx, at("2026-09-14 07:00"))
	p := hook.Next(t)
	if want := "今天（周一）有 2 节课：\n08:00-09:40 高等数学 @A101\n10:00-11:40 大学英语 @C301"; p.Event != notify.EventReminder || p.Text != want {
		t.Fatalf("got %q / %q", p.Event, p.Text)
	}
	s.Tick(ctx, at("2026-09-14 07:03")) // 不重复
	hook.None(t)
	s.Tick(ctx, at("2026-09-15 07:00")) // 周二第 2 周没课
	hook.None(t)
	s.Tick(ctx, at("2026-09-21 07:30")) // 错过窗口不补发
	hook.None(t)
	s.Tick(ctx, time.Date(2026, 9, 27, 23, 0, 0, 0, time.UTC)) // UTC 23:00 = 北京时间周一 07:00
	if p := hook.Next(t); !strings.HasPrefix(p.Text, "今天（周一）") {
		t.Fatalf("utc tick: %q", p.Text)
	}
}

func TestDailyTomorrow(t *testing.T) {
	s, hook := setup(t, model.ReminderSettings{Daily: true, DailyTime: "21:00", DailyTomorrow: true})
	s.Tick(ctx, at("2026-09-06 21:00"))
	if p := hook.Next(t); !strings.HasPrefix(p.Text, "明天（周一）有 2 节课") {
		t.Fatalf("text = %q", p.Text)
	}
	s.Tick(ctx, at("2026-09-07 21:00"))
	if p := hook.Next(t); !strings.Contains(p.Text, "14:00-15:40 体育") {
		t.Fatalf("text = %q", p.Text)
	}
}

func TestBeforeClass(t *testing.T) {
	s, hook := setup(t, model.ReminderSettings{BeforeClass: true, BeforeMinutes: 15})
	s.Tick(ctx, at("2026-09-14 07:44"))
	hook.None(t)
	s.Tick(ctx, at("2026-09-14 07:45"))
	if p := hook.Next(t); p.Text != "15 分钟后上课：08:00-09:40 高等数学 @A101（王老师）" {
		t.Fatalf("text = %q", p.Text)
	}
	s.Tick(ctx, at("2026-09-14 07:46"))
	hook.None(t)
	s.Tick(ctx, at("2026-09-14 09:47"))
	if p := hook.Next(t); !strings.Contains(p.Text, "大学英语") {
		t.Fatalf("text = %q", p.Text)
	}
	s.Tick(ctx, at("2026-09-14 10:01"))
	hook.None(t)
}

func TestDisabled(t *testing.T) {
	s, hook := setup(t, model.ReminderSettings{BeforeClass: true})
	s.Save(ctx, "u1", &model.ReminderSettings{})
	s.Tick(ctx, at("2026-09-14 07:45"))
	hook.None(t)
}

func TestValidate(t *testing.T) {
	for _, r := range []model.ReminderSettings{{DailyTime: "7点"}, {BeforeMinutes: 500}, {BeforeMinutes: -1}} {
		if err := reminder.Validate(&r); err == nil {
			t.Errorf("%+v should be invalid", r)
		}
	}
	r := model.ReminderSettings{}
	if err := reminder.Validate(&r); err != nil || r.DailyTime != "07:00" || r.BeforeMinutes != 15 {
		t.Fatalf("defaults: %+v, %v", r, err)
	}
}
