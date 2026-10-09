package grades

import (
	"math"
	"strings"
	"testing"

	"jww/internal/model"
)

func sc(year, term, course string, grade any, gpa, credit float64) model.Score {
	return model.Score{Year: year, Semester: term, Course: course, Grade: grade, GPA: gpa, Credit: credit}
}

func TestTrack(t *testing.T) {
	known := &model.KnownScores{}

	// 第一次不完整：不可信
	if _, ready := Track(known, []model.Score{sc("2025-2026", "1", "数学", "80", 3, 4)}, false); ready {
		t.Fatal("baseline should not be ready")
	}
	// 第一次完整：仍然是建立基线
	changes, ready := Track(known, []model.Score{sc("2025-2026", "1", "数学", "80", 3, 4), sc("2025-2026", "1", "英语", "70", 2, 2)}, true)
	if ready || len(changes) != 1 {
		t.Fatalf("ready=%v changes=%+v", ready, changes)
	}
	// 之后的变化可信
	changes, ready = Track(known, []model.Score{
		sc("2025-2026", "1", "数学", "85", 3.5, 4),
		sc("2025-2026", "1", "英语", "70", 2, 2),
		sc("2025-2026", "2", "体育", "优秀", 4, 1),
		sc("2025-2026", "2", "物理", nil, 0, 3),
	}, true)
	if !ready || len(changes) != 2 || changes[0].Score.Course != "数学" || changes[0].OldGrade != "80" || changes[1].Score.Course != "体育" {
		t.Fatalf("ready=%v changes=%+v", ready, changes)
	}
	text := ChangesSummary(changes)
	if !strings.Contains(text, "《数学》85（绩点 3.5，学分 4） [由 80 修改]") || !strings.Contains(text, "《体育》优秀（绩点 4，学分 1）") {
		t.Fatalf("summary:\n%s", text)
	}
}

func TestStatsAndFilters(t *testing.T) {
	scores := []model.Score{
		sc("2024-2025", "2", "数学", float64(90), 4, 4),
		sc("2024-2025", "2", "英语", float64(55), 0, 2),
		sc("2025-2026", "1", "物理", float64(80), 3, 3),
		sc("2025-2026", "1", "体育", "优秀", 3.5, 1),
	}
	st := Stats(scores)
	if st.TotalCredits != 10 || st.TotalCourses != 4 || st.FailedCount != 1 || len(st.FailedCourses) != 1 {
		t.Fatalf("stats = %+v", st)
	}
	if math.Abs(st.WeightedGPA-(16+0+9+3.5)/10.0) > 1e-9 {
		t.Fatalf("weighted = %v", st.WeightedGPA)
	}
	if len(st.SemesterStats) != 2 || st.SemesterStats[0].Semester != "2025-2026-1" || st.SemesterStats[0].GPA != 12.5/4 {
		t.Fatalf("semesters = %+v", st.SemesterStats)
	}
	if st.HighestCourse.Course != "数学" || st.LowestCourse.Course != "物理" {
		t.Fatalf("highest/lowest = %v / %v", st.HighestCourse.Course, st.LowestCourse.Course)
	}
	if got := Semesters(scores); len(got) != 2 || got[0] != "2025-2026-1" {
		t.Fatalf("semesters = %v", got)
	}
	if got := Filter(scores, "2024-2025-2"); len(got) != 2 {
		t.Fatalf("filter = %v", got)
	}
	if empty := Stats(nil); empty.SemesterStats == nil || empty.TotalCourses != 0 {
		t.Fatalf("empty = %+v", empty)
	}
}
