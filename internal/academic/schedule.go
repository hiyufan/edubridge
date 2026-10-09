package academic

import (
	"context"
	"errors"
	"log/slog"

	"jww/internal/model"
	"jww/internal/notify"
	"jww/internal/store"
	"jww/internal/timetable"
)

type scheduleEntry struct {
	schedule *model.FullSchedule
}

// WeekSchedule 单周课表；week<=0 表示当前周（按学期起始日计算真实当前周）
func (s *Service) WeekSchedule(ctx context.Context, uid string, week int) (*model.Schedule, error) {
	c, err := s.client(ctx, uid)
	if err != nil {
		return nil, err
	}
	sched, err := c.WeekSchedule(ctx, week)
	if err == nil && week <= 0 {
		// 入口页可能是任意一周，先用它推算学期起始日，再取真实当前周
		current := 1
		if sched.SemesterStart != "" {
			current = timetable.CurrentWeek(sched.SemesterStart, MaxWeek, s.now())
		}
		if sched.Week != current || len(sched.Courses) == 0 {
			sched, err = c.WeekSchedule(ctx, current)
		}
		if err == nil {
			sched.CurrentWeek = current
		}
	} else if err == nil && sched.SemesterStart != "" {
		sched.CurrentWeek = timetable.CurrentWeek(sched.SemesterStart, MaxWeek, s.now())
	}
	if err := s.done(ctx, uid, c, err); err != nil {
		return nil, err
	}
	return sched, nil
}

// FullSchedule 全学期课表（缓存 5 分钟）
func (s *Service) FullSchedule(ctx context.Context, uid string, maxWeek int) (*model.FullSchedule, error) {
	if e, ok := s.schedules.get(uid, s.now()); ok && e.schedule.TotalWeeks == maxWeek {
		cp := *e.schedule
		cp.Courses = append([]model.Course(nil), e.schedule.Courses...)
		return &cp, nil
	}
	return s.fetchSchedule(ctx, uid, maxWeek)
}

// RefreshSchedule 跳过缓存重新拉取全学期课表并检测变动（后台监控用）
func (s *Service) RefreshSchedule(ctx context.Context, uid string) error {
	_, err := s.fetchSchedule(ctx, uid, MaxWeek)
	return err
}

func (s *Service) fetchSchedule(ctx context.Context, uid string, maxWeek int) (*model.FullSchedule, error) {
	c, err := s.client(ctx, uid)
	if err != nil {
		return nil, err
	}
	full, complete, err := c.FullSchedule(ctx, maxWeek)
	if err := s.done(ctx, uid, c, err); err != nil {
		return nil, err
	}
	if full.SemesterStart != "" {
		full.CurrentWeek = timetable.CurrentWeek(full.SemesterStart, maxWeek, s.now())
	}
	s.schedules.set(uid, scheduleEntry{full}, s.now().Add(scheduleCacheTTL))
	s.recordSchedule(ctx, uid, full, complete)
	return full, nil
}

// recordSchedule 与上一次完整课表比对，有变动则通知，然后保存为新快照。
// 拉取不完整时不比对（缺的周会被误判成“停课”），只在还没有快照时保存一份供 iCal 使用。
func (s *Service) recordSchedule(ctx context.Context, uid string, cur *model.FullSchedule, complete bool) {
	s.recordMu.Lock()
	defer s.recordMu.Unlock()

	prev, prevComplete, err := s.store.Snapshot(ctx, uid)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		slog.Warn("load snapshot failed", "uid", uid, "err", err)
		return
	}
	if !complete {
		if prev == nil {
			if err := s.store.SaveSnapshot(ctx, uid, cur, false); err != nil {
				slog.Warn("save snapshot failed", "uid", uid, "err", err)
			}
		}
		return
	}

	if prev != nil && prevComplete && prev.Semester == cur.Semester {
		if d := timetable.Diff(prev, cur, cur.CurrentWeek, s.now()); !d.Empty() {
			slog.Info("schedule changed", "uid", uid, "added", len(d.Added), "removed", len(d.Removed), "changed", len(d.Changed))
			if err := s.store.SaveLatestDiff(ctx, uid, d); err != nil {
				slog.Warn("save diff failed", "uid", uid, "err", err)
			}
			s.notifier.Notify(uid, notify.EventScheduleDiff, timetable.DiffSummary(d), d)
		}
	}
	if err := s.store.SaveSnapshot(ctx, uid, cur, true); err != nil {
		slog.Warn("save snapshot failed", "uid", uid, "err", err)
	}
}

// Conflicts 全学期课表中的时间冲突
func (s *Service) Conflicts(ctx context.Context, uid string) ([]model.ConflictPair, error) {
	full, err := s.FullSchedule(ctx, uid, MaxWeek)
	if err != nil {
		return nil, err
	}
	return timetable.Conflicts(full.Courses), nil
}

// LatestDiff 最近一次检测到的课表变动；没有时返回 nil
func (s *Service) LatestDiff(ctx context.Context, uid string) (*model.ScheduleDiff, error) {
	d, err := s.store.LatestDiff(ctx, uid)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	return d, err
}
