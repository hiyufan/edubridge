package timetable

import (
	"strings"
	"testing"
	"time"

	"jww/internal/model"
)

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, CST)
	if err != nil {
		panic(err)
	}
	return t
}

func TestWeeks(t *testing.T) {
	const start = "2026-09-07" // 周一
	cases := []struct {
		day  string
		want int
	}{
		{"2026-09-06 23:59", 0}, {"2026-09-07 00:00", 1}, {"2026-09-13 23:59", 1}, {"2026-09-14 00:01", 2},
	}
	for _, c := range cases {
		if got := WeekOf(start, at(c.day)); got != c.want {
			t.Errorf("WeekOf(%s) = %d, want %d", c.day, got, c.want)
		}
	}
	// UTC 时间换算成北京时间：周日 16:30 UTC = 周一 00:30 北京时间
	if got := WeekOf(start, time.Date(2026, 9, 13, 16, 30, 0, 0, time.UTC)); got != 2 {
		t.Errorf("UTC conversion: %d", got)
	}
	if CurrentWeek(start, 20, at("2026-08-01 10:00")) != 1 || CurrentWeek(start, 20, at("2027-08-01 10:00")) != 20 {
		t.Error("CurrentWeek should clamp")
	}
	if Weekday(at("2026-09-13 10:00")) != 7 || WeekdayName(3) != "周三" {
		t.Error("weekday")
	}
}

func TestPeriodsAndClassesOn(t *testing.T) {
	if s, e := PeriodTimeRange(1, 2); s != "08:00" || e != "09:40" {
		t.Errorf("1-2 = %s-%s", s, e)
	}
	if s, e := PeriodTimeRange(12, 3); s != "21:15" || e != "22:00" {
		t.Errorf("12+ = %s-%s", s, e)
	}
	if _, ok := ClassStart(at("2026-09-07 00:00"), 99); ok {
		t.Error("unknown period")
	}

	s := &model.FullSchedule{SemesterStart: "2026-09-07", Courses: []model.Course{
		{Name: "英语", DayOfWeek: 1, PeriodStart: 3, Weeks: []int{1, 2}},
		{Name: "数学", DayOfWeek: 1, PeriodStart: 1, Weeks: []int{2}},
		{Name: "体育", DayOfWeek: 2, PeriodStart: 1, Weeks: []int{2}},
	}}
	got := ClassesOn(s, at("2026-09-14 12:00"))
	if len(got) != 2 || got[0].Name != "数学" || got[1].Name != "英语" {
		t.Fatalf("ClassesOn = %+v", got)
	}
	if len(ClassesOn(s, at("2026-09-01 12:00"))) != 0 {
		t.Error("before semester")
	}
}

func TestDiffIgnoresPastWeeksAndGroupsWeeks(t *testing.T) {
	old := &model.FullSchedule{Courses: []model.Course{
		{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 2, Weeks: []int{1, 2, 3, 4, 5, 6}},
		{Name: "大学英语", Teacher: "赵老师", Room: "C301", DayOfWeek: 2, PeriodStart: 3, Periods: 2, Weeks: []int{1, 2, 3, 4, 5, 6}},
	}}
	cur := &model.FullSchedule{Courses: []model.Course{
		{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 2, Weeks: []int{2, 3, 4}},
		{Name: "大学英语", Teacher: "赵老师", Room: "C301", DayOfWeek: 2, PeriodStart: 3, Periods: 2, Weeks: []int{1, 2, 3}},
		{Name: "大学英语", Teacher: "赵老师", Room: "D101", DayOfWeek: 2, PeriodStart: 3, Periods: 2, Weeks: []int{4, 5, 6}},
		{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 6, PeriodStart: 1, Periods: 2, Weeks: []int{5}},
	}}

	d := Diff(old, cur, 3, time.Now())
	if len(d.Removed) != 2 || d.Removed[0].Week != 5 || d.Removed[1].Week != 6 {
		t.Fatalf("removed = %+v", d.Removed)
	}
	if len(d.Added) != 1 || d.Added[0].DayOfWeek != 6 {
		t.Fatalf("added = %+v", d.Added)
	}
	if len(d.Changed) != 3 || d.Changed[0].New.Room != "D101" || d.Changed[0].New.Week != 4 {
		t.Fatalf("changed = %+v", d.Changed)
	}
	text := DiffSummary(d)
	for _, want := range []string{
		"+ 高等数学 周六 第1-2节 @A101（第5周）",
		"- 高等数学 周一 第1-2节 @A101（第5、6周）",
		"* 第4周 大学英语 周二 第3-4节 @D101（教室 C301 → D101）",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("summary missing %q:\n%s", want, text)
		}
	}
	if !Diff(cur, cur, 1, time.Now()).Empty() {
		t.Error("same schedule should have empty diff")
	}
}

func TestConflicts(t *testing.T) {
	got := Conflicts([]model.Course{
		{Name: "A", DayOfWeek: 1, PeriodStart: 1, Periods: 2, Weeks: []int{1, 2, 3}},
		{Name: "B", DayOfWeek: 1, PeriodStart: 2, Periods: 2, Weeks: []int{3, 4}},
		{Name: "C", DayOfWeek: 1, PeriodStart: 3, Periods: 1, Weeks: []int{1, 4}}, // 与 A 节次不重叠，与 B 第 4 周冲突
		{Name: "D", DayOfWeek: 2, PeriodStart: 1, Periods: 2, Weeks: []int{1}},
	})
	if len(got) != 2 || got[0].CourseB.Name != "B" || len(got[0].ConflictWeeks) != 1 || got[1].CourseA.Name != "B" || got[1].CourseB.Name != "C" {
		t.Fatalf("conflicts = %+v", got)
	}
}

func TestICal(t *testing.T) {
	s := &model.FullSchedule{SemesterStart: "2026-09-07", Courses: []model.Course{
		{Name: "高等数学", Teacher: "王老师", Room: "A101, 一楼", DayOfWeek: 3, PeriodStart: 3, Periods: 2, Weeks: []int{1, 3}},
	}}
	ics := ICal(s, at("2026-09-01 08:00"))
	if n := strings.Count(ics, "BEGIN:VEVENT"); n != 2 {
		t.Fatalf("events = %d", n)
	}
	for _, want := range []string{
		"DTSTART:20260909T100000\r\nDTEND:20260909T114000",
		"DTSTART:20260923T100000",
		`LOCATION:A101\, 一楼`,
	} {
		if !strings.Contains(ics, want) {
			t.Errorf("missing %q in\n%s", want, ics)
		}
	}
	if strings.Contains(ics, "RRULE") {
		t.Error("should not repeat events")
	}
}
