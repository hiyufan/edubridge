package timetable

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"jww/internal/model"
)

func occurrences(s *model.FullSchedule, fromWeek int) map[string]model.Occurrence {
	out := make(map[string]model.Occurrence)
	for _, c := range s.Courses {
		for _, w := range c.Weeks {
			if w < fromWeek {
				continue
			}
			out[fmt.Sprintf("%d|%d|%d|%s", w, c.DayOfWeek, c.PeriodStart, c.Name)] = model.Occurrence{
				Week: w, DayOfWeek: c.DayOfWeek, PeriodStart: c.PeriodStart, Periods: c.Periods,
				Name: c.Name, Teacher: c.Teacher, Room: c.Room,
			}
		}
	}
	return out
}

func lessOcc(a, b model.Occurrence) bool {
	if a.Week != b.Week {
		return a.Week < b.Week
	}
	if a.DayOfWeek != b.DayOfWeek {
		return a.DayOfWeek < b.DayOfWeek
	}
	return a.PeriodStart < b.PeriodStart
}

// Diff 按“某周某节课”比较两次完整课表，只看 fromWeek 及以后
func Diff(old, cur *model.FullSchedule, fromWeek int, now time.Time) *model.ScheduleDiff {
	oldOcc, newOcc := occurrences(old, fromWeek), occurrences(cur, fromWeek)
	d := &model.ScheduleDiff{DetectedAt: now}
	for k, o := range newOcc {
		prev, ok := oldOcc[k]
		switch {
		case !ok:
			d.Added = append(d.Added, o)
		case prev.Room != o.Room || prev.Teacher != o.Teacher || prev.Periods != o.Periods:
			d.Changed = append(d.Changed, model.OccurrenceChange{Old: prev, New: o})
		}
	}
	for k, o := range oldOcc {
		if _, ok := newOcc[k]; !ok {
			d.Removed = append(d.Removed, o)
		}
	}
	sort.Slice(d.Added, func(i, j int) bool { return lessOcc(d.Added[i], d.Added[j]) })
	sort.Slice(d.Removed, func(i, j int) bool { return lessOcc(d.Removed[i], d.Removed[j]) })
	sort.Slice(d.Changed, func(i, j int) bool { return lessOcc(d.Changed[i].New, d.Changed[j].New) })
	return d
}

func describeSlot(o model.Occurrence) string {
	periods := fmt.Sprintf("第%d节", o.PeriodStart)
	if o.Periods > 1 {
		periods = fmt.Sprintf("第%d-%d节", o.PeriodStart, o.PeriodStart+o.Periods-1)
	}
	s := fmt.Sprintf("%s %s %s", o.Name, WeekdayName(o.DayOfWeek), periods)
	if o.Room != "" {
		s += " @" + o.Room
	}
	return s
}

// groupByWeeks 同一时段同一门课的多周合并成一行：「高等数学 周三 第3-4节 @A101（第8、9周）」
func groupByWeeks(list []model.Occurrence) []string {
	var order []string
	weeks := make(map[string][]string)
	for _, o := range list {
		desc := describeSlot(o)
		if _, ok := weeks[desc]; !ok {
			order = append(order, desc)
		}
		weeks[desc] = append(weeks[desc], fmt.Sprint(o.Week))
	}
	lines := make([]string, len(order))
	for i, desc := range order {
		lines[i] = fmt.Sprintf("%s（第%s周）", desc, strings.Join(weeks[desc], "、"))
	}
	return lines
}

// DiffSummary 适合推送的中文描述
func DiffSummary(d *model.ScheduleDiff) string {
	if d.Empty() {
		return "课表无变动"
	}
	var b strings.Builder
	b.WriteString("课表有变动：")
	if len(d.Added) > 0 {
		b.WriteString("\n【加课】")
		for _, l := range groupByWeeks(d.Added) {
			b.WriteString("\n+ " + l)
		}
	}
	if len(d.Removed) > 0 {
		b.WriteString("\n【减课/停课】")
		for _, l := range groupByWeeks(d.Removed) {
			b.WriteString("\n- " + l)
		}
	}
	if len(d.Changed) > 0 {
		b.WriteString("\n【调整】")
		for _, ch := range d.Changed {
			fmt.Fprintf(&b, "\n* 第%d周 %s", ch.New.Week, describeSlot(ch.New))
			if ch.Old.Room != ch.New.Room {
				fmt.Fprintf(&b, "（教室 %s → %s）", ch.Old.Room, ch.New.Room)
			}
			if ch.Old.Teacher != ch.New.Teacher {
				fmt.Fprintf(&b, "（老师 %s → %s）", ch.Old.Teacher, ch.New.Teacher)
			}
		}
	}
	return b.String()
}
