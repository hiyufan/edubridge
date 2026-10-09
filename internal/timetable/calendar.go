// Package timetable 课表领域逻辑：作息时间、教学周计算、课表变动比对、冲突检测、iCal 导出。
// 全部为纯函数，不做任何 IO。
package timetable

import (
	"math"
	"sort"
	"time"
	_ "time/tzdata" // 容器镜像（alpine）可能没有时区数据

	"jww/internal/model"
)

// CST 北京时间；所有“今天/第几周/几点上课”都按它计算
var CST = loadCST()

func loadCST() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// periodTimes 节次作息（与前端 utils/periods.js 保持一致）
var periodTimes = map[int][2]string{
	1: {"08:00", "08:45"}, 2: {"08:55", "09:40"}, 3: {"10:00", "10:45"}, 4: {"10:55", "11:40"},
	5: {"14:00", "14:45"}, 6: {"14:55", "15:40"}, 7: {"15:50", "16:35"}, 8: {"16:45", "17:30"},
	9: {"18:30", "19:15"}, 10: {"19:25", "20:10"}, 11: {"20:20", "21:05"}, 12: {"21:15", "22:00"},
}

// PeriodTimeRange 连续节次的起止时间（"08:00"）；未知节次返回空字符串
func PeriodTimeRange(periodStart, periods int) (start, end string) {
	if periods < 1 {
		periods = 1
	}
	start = periodTimes[periodStart][0]
	end = periodTimes[periodStart+periods-1][1]
	if end == "" {
		end = periodTimes[periodStart][1]
	}
	return start, end
}

// ClassStart 某天某节课的开始时刻；未知节次返回 false
func ClassStart(day time.Time, periodStart int) (time.Time, bool) {
	start, _ := PeriodTimeRange(periodStart, 1)
	t, err := time.Parse("15:04", start)
	if err != nil {
		return time.Time{}, false
	}
	d := day.In(CST)
	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), 0, 0, CST), true
}

// WeekOf 某天是第几教学周（学期开始前或学期起始日未知时返回 0）
func WeekOf(semesterStart string, day time.Time) int {
	start, err := time.ParseInLocation("2006-01-02", semesterStart, CST)
	if err != nil {
		return 0
	}
	d := day.In(CST)
	d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, CST)
	days := int(math.Round(d.Sub(start).Hours() / 24))
	if days < 0 {
		return 0
	}
	return days/7 + 1
}

// CurrentWeek 当前教学周，限制在 [1, maxWeek]
func CurrentWeek(semesterStart string, maxWeek int, now time.Time) int {
	w := WeekOf(semesterStart, now)
	if w < 1 {
		return 1
	}
	if w > maxWeek {
		return maxWeek
	}
	return w
}

// Weekday 1=周一 … 7=周日
func Weekday(day time.Time) int {
	wd := int(day.In(CST).Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

var weekdayNames = [...]string{"", "周一", "周二", "周三", "周四", "周五", "周六", "周日"}

// WeekdayName 1 -> "周一"
func WeekdayName(d int) string {
	if d < 1 || d > 7 {
		return ""
	}
	return weekdayNames[d]
}

// ClassesOn 某天要上的课，按节次排序
func ClassesOn(s *model.FullSchedule, day time.Time) []model.Course {
	week := WeekOf(s.SemesterStart, day)
	if week == 0 {
		return nil
	}
	wd := Weekday(day)
	var out []model.Course
	for _, c := range s.Courses {
		if c.DayOfWeek == wd && containsInt(c.Weeks, week) {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PeriodStart < out[j].PeriodStart })
	return out
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
