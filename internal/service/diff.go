package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"jww/internal/model"
)

// Occurrence 某一周的一次具体上课
type Occurrence struct {
	Week        int    `json:"week"`
	DayOfWeek   int    `json:"dayOfWeek"`
	PeriodStart int    `json:"periodStart"`
	Periods     int    `json:"periods"`
	Name        string `json:"name"`
	Teacher     string `json:"teacher"`
	Room        string `json:"room"`
}

// OccurrenceChange 同一节课的教室/老师/节数变化
type OccurrenceChange struct {
	Old Occurrence `json:"old"`
	New Occurrence `json:"new"`
}

// ScheduleDiff 课表变动（只比较当前周及以后）
type ScheduleDiff struct {
	Added      []Occurrence       `json:"added"`   // 加课
	Removed    []Occurrence       `json:"removed"` // 减课/停课
	Changed    []OccurrenceChange `json:"changed"` // 换教室、换老师等
	DetectedAt time.Time          `json:"detectedAt"`
}

// Empty 是否没有任何变动
func (d *ScheduleDiff) Empty() bool {
	return d == nil || len(d.Added)+len(d.Removed)+len(d.Changed) == 0
}

func expandOccurrences(s *model.FullSchedule, fromWeek int) map[string]Occurrence {
	out := make(map[string]Occurrence)
	for _, c := range s.Courses {
		for _, w := range c.Weeks {
			if w < fromWeek {
				continue
			}
			o := Occurrence{
				Week: w, DayOfWeek: c.DayOfWeek, PeriodStart: c.PeriodStart, Periods: c.Periods,
				Name: c.Name, Teacher: c.Teacher, Room: c.Room,
			}
			out[fmt.Sprintf("%d|%d|%d|%s", w, c.DayOfWeek, c.PeriodStart, c.Name)] = o
		}
	}
	return out
}

func lessOcc(a, b Occurrence) bool {
	if a.Week != b.Week {
		return a.Week < b.Week
	}
	if a.DayOfWeek != b.DayOfWeek {
		return a.DayOfWeek < b.DayOfWeek
	}
	return a.PeriodStart < b.PeriodStart
}

func sortOccurrences(list []Occurrence) {
	sort.Slice(list, func(i, j int) bool { return lessOcc(list[i], list[j]) })
}

// diffSchedules 比较两次完整课表，只看 fromWeek 及以后的课
func diffSchedules(old, new *model.FullSchedule, fromWeek int) *ScheduleDiff {
	oldOcc := expandOccurrences(old, fromWeek)
	newOcc := expandOccurrences(new, fromWeek)

	diff := &ScheduleDiff{DetectedAt: time.Now()}
	for k, o := range newOcc {
		prev, ok := oldOcc[k]
		switch {
		case !ok:
			diff.Added = append(diff.Added, o)
		case prev.Room != o.Room || prev.Teacher != o.Teacher || prev.Periods != o.Periods:
			diff.Changed = append(diff.Changed, OccurrenceChange{Old: prev, New: o})
		}
	}
	for k, o := range oldOcc {
		if _, ok := newOcc[k]; !ok {
			diff.Removed = append(diff.Removed, o)
		}
	}

	sortOccurrences(diff.Added)
	sortOccurrences(diff.Removed)
	sort.Slice(diff.Changed, func(i, j int) bool {
		return lessOcc(diff.Changed[i].New, diff.Changed[j].New)
	})
	return diff
}

var weekdayNames = []string{"", "周一", "周二", "周三", "周四", "周五", "周六", "周日"}

func describeSlot(o Occurrence) string {
	day := ""
	if o.DayOfWeek >= 1 && o.DayOfWeek <= 7 {
		day = weekdayNames[o.DayOfWeek]
	}
	periods := fmt.Sprintf("第%d节", o.PeriodStart)
	if o.Periods > 1 {
		periods = fmt.Sprintf("第%d-%d节", o.PeriodStart, o.PeriodStart+o.Periods-1)
	}
	s := fmt.Sprintf("%s %s %s", o.Name, day, periods)
	if o.Room != "" {
		s += " @" + o.Room
	}
	return s
}

// groupByWeeks 把同一时段同一门课的多周合并成一行，例如「高等数学 周三 第3-4节 @A101（第8、9周）」
func groupByWeeks(list []Occurrence) []string {
	type group struct {
		desc  string
		weeks []int
	}
	var order []string
	groups := make(map[string]*group)
	for _, o := range list {
		desc := describeSlot(o)
		g, ok := groups[desc]
		if !ok {
			g = &group{desc: desc}
			groups[desc] = g
			order = append(order, desc)
		}
		g.weeks = append(g.weeks, o.Week)
	}
	lines := make([]string, 0, len(order))
	for _, desc := range order {
		g := groups[desc]
		weeks := make([]string, len(g.weeks))
		for i, w := range g.weeks {
			weeks[i] = fmt.Sprint(w)
		}
		lines = append(lines, fmt.Sprintf("%s（第%s周）", g.desc, strings.Join(weeks, "、")))
	}
	return lines
}

// Summary 生成适合推送的中文文本
func (d *ScheduleDiff) Summary() string {
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
			b.WriteString(fmt.Sprintf("\n* 第%d周 %s", ch.New.Week, describeSlot(ch.New)))
			if ch.Old.Room != ch.New.Room {
				b.WriteString(fmt.Sprintf("（教室 %s → %s）", ch.Old.Room, ch.New.Room))
			}
			if ch.Old.Teacher != ch.New.Teacher {
				b.WriteString(fmt.Sprintf("（老师 %s → %s）", ch.Old.Teacher, ch.New.Teacher))
			}
		}
	}
	return b.String()
}
