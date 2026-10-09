package service

import (
	"strings"
	"testing"

	"jww/internal/model"
)

func TestDiffSchedulesIgnoresPastWeeksAndGroupsWeeks(t *testing.T) {
	old := &model.FullSchedule{Courses: []model.Course{
		{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 2, Weeks: []int{1, 2, 3, 4, 5, 6}},
		{Name: "大学英语", Teacher: "赵老师", Room: "C301", DayOfWeek: 2, PeriodStart: 3, Periods: 2, Weeks: []int{1, 2, 3, 4, 5, 6}},
	}}
	cur := &model.FullSchedule{Courses: []model.Course{
		// 第 1 周（已过去）和第 5、6 周停课
		{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 2, Weeks: []int{2, 3, 4}},
		// 第 4 周起换教室
		{Name: "大学英语", Teacher: "赵老师", Room: "C301", DayOfWeek: 2, PeriodStart: 3, Periods: 2, Weeks: []int{1, 2, 3}},
		{Name: "大学英语", Teacher: "赵老师", Room: "D101", DayOfWeek: 2, PeriodStart: 3, Periods: 2, Weeks: []int{4, 5, 6}},
		// 第 5 周补课
		{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 6, PeriodStart: 1, Periods: 2, Weeks: []int{5}},
	}}

	diff := diffSchedules(old, cur, 3)

	if len(diff.Removed) != 2 || diff.Removed[0].Week != 5 || diff.Removed[1].Week != 6 {
		t.Fatalf("removed = %+v", diff.Removed)
	}
	if len(diff.Added) != 1 || diff.Added[0].DayOfWeek != 6 {
		t.Fatalf("added = %+v", diff.Added)
	}
	if len(diff.Changed) != 3 || diff.Changed[0].New.Room != "D101" || diff.Changed[0].New.Week != 4 {
		t.Fatalf("changed = %+v", diff.Changed)
	}

	text := diff.Summary()
	for _, want := range []string{
		"+ 高等数学 周六 第1-2节 @A101（第5周）",
		"- 高等数学 周一 第1-2节 @A101（第5、6周）",
		"* 第4周 大学英语 周二 第3-4节 @D101（教室 C301 → D101）",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestDiffSchedulesNoChange(t *testing.T) {
	s := &model.FullSchedule{Courses: []model.Course{
		{Name: "高等数学", DayOfWeek: 1, PeriodStart: 1, Periods: 2, Weeks: []int{1, 2, 3}},
	}}
	if d := diffSchedules(s, s, 1); !d.Empty() {
		t.Fatalf("expected empty diff, got %+v", d)
	}
}
