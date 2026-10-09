// Package reminder 上课提醒：基于 Redis 中的课表快照（后台监控定时刷新），按北京时间推送
//   - 每日课表：每天在设定时间推送当天（或明天）的课；
//   - 课前提醒：每节课开始前 N 分钟推送。
//
// 同一条提醒只发一次；服务停机错过的提醒不补发。
package reminder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"jww/internal/model"
	"jww/internal/notify"
	"jww/internal/store"
	"jww/internal/timetable"
)

const (
	dailyWindow  = 10 * time.Minute // 到点后多久内仍可发送
	beforeWindow = 5 * time.Minute
)

// Defaults 未设置时的默认值（均关闭）
func Defaults() *model.ReminderSettings {
	return &model.ReminderSettings{DailyTime: "07:00", BeforeMinutes: 15}
}

// ValidationError 设置不合法，信息可直接展示
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// Validate 校验并补全默认值
func Validate(r *model.ReminderSettings) error {
	if r.DailyTime == "" {
		r.DailyTime = "07:00"
	}
	if _, err := time.Parse("15:04", r.DailyTime); err != nil {
		return &ValidationError{"提醒时间格式应为 HH:MM"}
	}
	if r.BeforeMinutes == 0 {
		r.BeforeMinutes = 15
	}
	if r.BeforeMinutes < 1 || r.BeforeMinutes > 180 {
		return &ValidationError{"课前提醒时间应在 1 到 180 分钟之间"}
	}
	return nil
}

// Scheduler 上课提醒调度器
type Scheduler struct {
	store    *store.Store
	notifier *notify.Notifier
}

// New 创建调度器
func New(st *store.Store, n *notify.Notifier) *Scheduler {
	return &Scheduler{store: st, notifier: n}
}

// Settings 用户的提醒设置（未设置时返回默认值）
func (s *Scheduler) Settings(ctx context.Context, uid string) (*model.ReminderSettings, error) {
	r, err := s.store.Reminder(ctx, uid)
	if errors.Is(err, store.ErrNotFound) {
		return Defaults(), nil
	}
	return r, err
}

// Save 校验并保存提醒设置
func (s *Scheduler) Save(ctx context.Context, uid string, r *model.ReminderSettings) error {
	if err := Validate(r); err != nil {
		return err
	}
	return s.store.SaveReminder(ctx, uid, r)
}

// Run 每 30 秒检查一次，直到 ctx 结束
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.Tick(ctx, now)
		}
	}
}

// Tick 发送 now 时刻到点的提醒
func (s *Scheduler) Tick(ctx context.Context, now time.Time) {
	uids, err := s.store.ReminderUsers(ctx)
	if err != nil {
		slog.Warn("load reminder users failed", "err", err)
		return
	}
	for _, uid := range uids {
		s.remind(ctx, uid, now.In(timetable.CST))
	}
}

func (s *Scheduler) remind(ctx context.Context, uid string, now time.Time) {
	settings, err := s.Settings(ctx, uid)
	if err != nil {
		slog.Warn("load reminder settings failed", "uid", uid, "err", err)
		return
	}
	schedule, _, err := s.store.Snapshot(ctx, uid)
	if err != nil {
		return // 还没有课表
	}
	today := now.Format("2006-01-02")

	if settings.Daily {
		t, _ := time.Parse("15:04", settings.DailyTime)
		target := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, timetable.CST)
		if inWindow(now, target, dailyWindow) && s.markSent(ctx, uid, "daily:"+today) {
			day := now
			if settings.DailyTomorrow {
				day = now.AddDate(0, 0, 1)
			}
			if courses := timetable.ClassesOn(schedule, day); len(courses) > 0 {
				s.notifier.Notify(uid, notify.EventReminder, dailyText(courses, day, settings.DailyTomorrow), nil)
			}
		}
	}

	if settings.BeforeClass {
		lead := time.Duration(settings.BeforeMinutes) * time.Minute
		for _, c := range timetable.ClassesOn(schedule, now) {
			start, ok := timetable.ClassStart(now, c.PeriodStart)
			if !ok || !now.Before(start) || !inWindow(now, start.Add(-lead), beforeWindow) {
				continue
			}
			if s.markSent(ctx, uid, fmt.Sprintf("class:%s:%d", today, c.PeriodStart)) {
				s.notifier.Notify(uid, notify.EventReminder, beforeText(c, settings.BeforeMinutes), nil)
			}
		}
	}
}

func inWindow(now, target time.Time, window time.Duration) bool {
	return !now.Before(target) && now.Sub(target) < window
}

func (s *Scheduler) markSent(ctx context.Context, uid, id string) bool {
	first, err := s.store.MarkReminderSent(ctx, uid, id)
	if err != nil {
		slog.Warn("mark reminder sent failed", "uid", uid, "err", err)
		return false
	}
	return first
}

func describe(c model.Course) string {
	from, to := timetable.PeriodTimeRange(c.PeriodStart, c.Periods)
	s := c.Name
	if from != "" {
		s = fmt.Sprintf("%s-%s %s", from, to, c.Name)
	}
	if c.Room != "" {
		s += " @" + c.Room
	}
	return s
}

func dailyText(courses []model.Course, day time.Time, tomorrow bool) string {
	label := "今天"
	if tomorrow {
		label = "明天"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s（%s）有 %d 节课：", label, timetable.WeekdayName(timetable.Weekday(day)), len(courses))
	for _, c := range courses {
		b.WriteString("\n" + describe(c))
	}
	return b.String()
}

func beforeText(c model.Course, minutes int) string {
	text := fmt.Sprintf("%d 分钟后上课：%s", minutes, describe(c))
	if c.Teacher != "" {
		text += "（" + c.Teacher + "）"
	}
	return text
}
