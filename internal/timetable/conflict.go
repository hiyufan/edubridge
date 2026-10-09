package timetable

import "jww/internal/model"

// Conflicts 找出同一天、节次重叠且上课周有交集的课程对
func Conflicts(courses []model.Course) []model.ConflictPair {
	var result []model.ConflictPair
	for i := 0; i < len(courses); i++ {
		for j := i + 1; j < len(courses); j++ {
			a, b := courses[i], courses[j]
			if a.DayOfWeek != b.DayOfWeek {
				continue
			}
			if a.PeriodStart+a.Periods-1 < b.PeriodStart || b.PeriodStart+b.Periods-1 < a.PeriodStart {
				continue
			}
			if weeks := intersect(a.Weeks, b.Weeks); len(weeks) > 0 {
				result = append(result, model.ConflictPair{CourseA: a, CourseB: b, ConflictWeeks: weeks})
			}
		}
	}
	return result
}

func intersect(a, b []int) []int {
	set := make(map[int]bool, len(b))
	for _, v := range b {
		set[v] = true
	}
	var out []int
	for _, v := range a {
		if set[v] {
			out = append(out, v)
		}
	}
	return out
}
