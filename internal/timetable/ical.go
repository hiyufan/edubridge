package timetable

import (
	"fmt"
	"strings"
	"time"

	"jww/internal/model"
)

func escapeICal(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\n", `\n`).Replace(s)
}

func icalTime(day time.Time, hhmm string) string {
	return day.Format("20060102") + "T" + strings.ReplaceAll(hhmm, ":", "") + "00"
}

// ICal 生成 iCalendar：每门课每个上课周一个事件（不使用 RRULE，周次常常不连续）
func ICal(s *model.FullSchedule, now time.Time) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//jww//Course Schedule//CN\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\nX-WR-CALNAME:课程表\r\nX-WR-TIMEZONE:Asia/Shanghai\r\n")

	start, err := time.ParseInLocation("2006-01-02", s.SemesterStart, CST)
	if err != nil {
		start = now.In(CST)
	}
	for _, c := range s.Courses {
		from, to := PeriodTimeRange(c.PeriodStart, c.Periods)
		if from == "" {
			from, to = "08:00", "08:45"
		}
		for _, week := range c.Weeks {
			day := start.AddDate(0, 0, (week-1)*7+(c.DayOfWeek-1))
			b.WriteString("BEGIN:VEVENT\r\n")
			fmt.Fprintf(&b, "UID:%s-%d-%d-%d@jwschedule\r\n", escapeICal(c.Name), week, c.DayOfWeek, c.PeriodStart)
			fmt.Fprintf(&b, "DTSTAMP:%s\r\n", now.UTC().Format("20060102T150405Z"))
			fmt.Fprintf(&b, "DTSTART:%s\r\nDTEND:%s\r\n", icalTime(day, from), icalTime(day, to))
			fmt.Fprintf(&b, "SUMMARY:%s\r\nLOCATION:%s\r\n", escapeICal(c.Name), escapeICal(c.Room))
			fmt.Fprintf(&b, "DESCRIPTION:教师: %s\\n第%d周 %s 第%d节\r\n",
				escapeICal(c.Teacher), week, WeekdayName(c.DayOfWeek), c.PeriodStart)
			b.WriteString("END:VEVENT\r\n")
		}
	}
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}
