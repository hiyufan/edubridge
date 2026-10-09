package service

import (
	"strings"
	"testing"
	"time"

	"jww/internal/model"
)

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, cst)
	if err != nil {
		panic(err)
	}
	return t
}

func setupReminder(t *testing.T, uid string, settings *ReminderSettings) (*JwService, *webhookRecorder) {
	t.Helper()
	setupRedis(t)
	hook := newWebhookRecorder(t)
	SaveWebhook(uid, &WebhookEntry{URL: hook.srv.URL})

	weeks := make([]int, 16)
	for i := range weeks {
		weeks[i] = i + 1
	}
	// 2026-09-07 是周一，第 1 周
	saveScheduleSnapshot(uid, &model.FullSchedule{
		SemesterStart: "2026-09-07",
		Courses: []model.Course{
			{Name: "大学英语", Teacher: "赵老师", Room: "C301", DayOfWeek: 1, PeriodStart: 3, Periods: 2, Weeks: weeks},
			{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 2, Weeks: weeks},
			{Name: "体育", DayOfWeek: 2, PeriodStart: 5, Periods: 2, Weeks: []int{1}},
		},
	}, true)
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := SaveReminderSettings(uid, settings); err != nil {
		t.Fatal(err)
	}
	svc := NewJwService()
	t.Cleanup(svc.Close)
	return svc, hook
}

func TestDailyReminder(t *testing.T) {
	svc, hook := setupReminder(t, "u1", &ReminderSettings{Daily: true, DailyTime: "07:00"})

	svc.runReminders(at("2026-09-14 06:59")) // 还没到点
	hook.none(t)

	svc.runReminders(at("2026-09-14 07:00")) // 周一，第 2 周
	p := hook.next(t)
	want := "今天（周一）有 2 节课：\n08:00-09:40 高等数学 @A101\n10:00-11:40 大学英语 @C301"
	if p.Event != EventReminder || p.Text != want {
		t.Fatalf("got %q / %q", p.Event, p.Text)
	}

	svc.runReminders(at("2026-09-14 07:03")) // 同一天不重复
	hook.none(t)

	svc.runReminders(at("2026-09-15 07:00")) // 周二第 2 周没有体育
	hook.none(t)

	svc.runReminders(at("2026-09-21 07:30")) // 超过 10 分钟窗口（例如服务停机）不再补发
	hook.none(t)
}

func TestDailyReminderTomorrow(t *testing.T) {
	svc, hook := setupReminder(t, "u2", &ReminderSettings{Daily: true, DailyTime: "21:00", DailyTomorrow: true})

	svc.runReminders(at("2026-09-06 21:00")) // 周日晚上，明天是第 1 周周一
	p := hook.next(t)
	if !strings.HasPrefix(p.Text, "明天（周一）有 2 节课") {
		t.Fatalf("text = %q", p.Text)
	}

	svc.runReminders(at("2026-09-07 21:00")) // 周一晚上，明天第 1 周周二有体育
	if p := hook.next(t); !strings.Contains(p.Text, "14:00-15:40 体育") {
		t.Fatalf("text = %q", p.Text)
	}
}

func TestBeforeClassReminder(t *testing.T) {
	svc, hook := setupReminder(t, "u3", &ReminderSettings{BeforeClass: true, BeforeMinutes: 15})

	svc.runReminders(at("2026-09-14 07:44"))
	hook.none(t)

	svc.runReminders(at("2026-09-14 07:45"))
	if p := hook.next(t); p.Text != "15 分钟后上课：08:00-09:40 高等数学 @A101（王老师）" {
		t.Fatalf("text = %q", p.Text)
	}
	svc.runReminders(at("2026-09-14 07:46"))
	hook.none(t)

	svc.runReminders(at("2026-09-14 09:47"))
	if p := hook.next(t); !strings.Contains(p.Text, "大学英语") {
		t.Fatalf("text = %q", p.Text)
	}

	svc.runReminders(at("2026-09-14 10:01")) // 已经开始上课
	hook.none(t)
}

func TestReminderDisabledRemovesUser(t *testing.T) {
	svc, hook := setupReminder(t, "u4", &ReminderSettings{BeforeClass: true})
	SaveReminderSettings("u4", &ReminderSettings{DailyTime: "07:00", BeforeMinutes: 15})
	svc.runReminders(at("2026-09-14 07:45"))
	hook.none(t)
}

func TestReminderValidate(t *testing.T) {
	for _, r := range []ReminderSettings{{DailyTime: "7点"}, {BeforeMinutes: 500}, {BeforeMinutes: -1}} {
		if err := r.Validate(); err == nil {
			t.Errorf("%+v should be invalid", r)
		}
	}
	r := ReminderSettings{}
	if err := r.Validate(); err != nil || r.DailyTime != "07:00" || r.BeforeMinutes != 15 {
		t.Fatalf("defaults: %+v, %v", r, err)
	}
}
