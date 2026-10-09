// Package grades 成绩领域逻辑：发现新成绩、GPA 统计、学期筛选。全部为纯函数。
package grades

import (
	"fmt"
	"sort"
	"strings"

	"jww/internal/model"
)

// Key 同一门课的唯一标识
func Key(s model.Score) string {
	return s.Year + "|" + s.Semester + "|" + s.Course
}

// SemesterOf 学期标识，例如 "2025-2026-1"
func SemesterOf(s model.Score) string {
	return s.Year + "-" + s.Semester
}

// GradeText 成绩的文字形式（数字或“优秀”等）；未录入为空
func GradeText(g any) string {
	if g == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(g))
}

// Track 把本次拉取的成绩并入 known，返回新出或被修改的成绩。
// 只有 known 已经有过完整基线时返回的变化才可信，调用方应据此决定是否通知。
func Track(known *model.KnownScores, scores []model.Score, complete bool) (changes []model.ScoreChange, baselineReady bool) {
	if known.Grades == nil {
		known.Grades = make(map[string]string)
	}
	baselineReady = known.Complete
	for _, s := range scores {
		grade := GradeText(s.Grade)
		if grade == "" {
			continue // 尚未录入
		}
		key := Key(s)
		old, ok := known.Grades[key]
		switch {
		case !ok:
			changes = append(changes, model.ScoreChange{Score: s})
		case old != grade:
			changes = append(changes, model.ScoreChange{Score: s, OldGrade: old})
		}
		known.Grades[key] = grade
	}
	if complete {
		known.Complete = true
	}
	sort.Slice(changes, func(i, j int) bool { return Key(changes[i].Score) < Key(changes[j].Score) })
	return changes, baselineReady
}

func formatNumber(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}

// ChangesSummary 适合推送的中文描述
func ChangesSummary(changes []model.ScoreChange) string {
	var b strings.Builder
	b.WriteString("出成绩了：")
	for _, ch := range changes {
		s := ch.Score
		fmt.Fprintf(&b, "\n《%s》%s", s.Course, GradeText(s.Grade))
		var extra []string
		if s.GPA > 0 {
			extra = append(extra, "绩点 "+formatNumber(s.GPA))
		}
		if s.Credit > 0 {
			extra = append(extra, "学分 "+formatNumber(s.Credit))
		}
		if len(extra) > 0 {
			b.WriteString("（" + strings.Join(extra, "，") + "）")
		}
		if ch.OldGrade != "" {
			fmt.Fprintf(&b, " [由 %s 修改]", ch.OldGrade)
		}
	}
	return b.String()
}

// Filter 只保留某学期（"2025-2026-1"）的成绩；semester 为空时原样返回
func Filter(scores []model.Score, semester string) []model.Score {
	if semester == "" {
		return scores
	}
	out := []model.Score{}
	for _, s := range scores {
		if SemesterOf(s) == semester {
			out = append(out, s)
		}
	}
	return out
}

// Semesters 出现过的学期，最新的在前
func Semesters(scores []model.Score) []string {
	set := make(map[string]bool)
	for _, s := range scores {
		set[SemesterOf(s)] = true
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out
}

// isFailed 数字成绩低于 60 视为挂科（文字成绩不判定）
func isFailed(g any) bool {
	switch v := g.(type) {
	case float64:
		return v < 60
	case int:
		return v < 60
	case int64:
		return v < 60
	}
	return false
}

// Stats 学分、加权 GPA、逐学期统计、最高/最低、挂科列表
func Stats(scores []model.Score) model.ScoreStats {
	stats := model.ScoreStats{
		TotalCourses:  len(scores),
		SemesterStats: []model.SemesterStat{},
		FailedCourses: []model.Score{},
	}
	if len(scores) == 0 {
		return stats
	}

	type term struct{ year, term string }
	byTerm := make(map[term][]model.Score)
	for _, s := range scores {
		k := term{s.Year, s.Semester}
		byTerm[k] = append(byTerm[k], s)
	}
	terms := make([]term, 0, len(byTerm))
	for k := range byTerm {
		terms = append(terms, k)
	}
	sort.Slice(terms, func(i, j int) bool {
		if terms[i].year != terms[j].year {
			return terms[i].year > terms[j].year
		}
		return terms[i].term > terms[j].term
	})

	var totalWeighted, sumTermGPA float64
	for _, k := range terms {
		st := model.SemesterStat{Semester: k.year + "-" + k.term, Year: k.year, Term: k.term, CourseCount: len(byTerm[k])}
		var weighted float64
		for _, s := range byTerm[k] {
			st.Credits += s.Credit
			weighted += s.Credit * s.GPA
			if isFailed(s.Grade) {
				st.FailedCount++
				stats.FailedCourses = append(stats.FailedCourses, s)
			}
			if s.GPA > 0 {
				if stats.HighestCourse == nil || s.GPA > stats.HighestCourse.GPA {
					c := s
					stats.HighestCourse = &c
				}
				if stats.LowestCourse == nil || s.GPA < stats.LowestCourse.GPA {
					c := s
					stats.LowestCourse = &c
				}
			}
		}
		if st.Credits > 0 {
			st.GPA = weighted / st.Credits
		}
		stats.TotalCredits += st.Credits
		stats.FailedCount += st.FailedCount
		totalWeighted += weighted
		sumTermGPA += st.GPA
		stats.SemesterStats = append(stats.SemesterStats, st)
	}
	if stats.TotalCredits > 0 {
		stats.WeightedGPA = totalWeighted / stats.TotalCredits
	}
	stats.SimpleGPA = sumTermGPA / float64(len(terms))
	return stats
}
