package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jww/internal/model"
	"jww/internal/service"
	"jww/pkg/response"
)

// GenerateICalToken 生成 90 天订阅 token（持久化到 Redis，按学号关联，同一用户旧 token 失效）
func (h *ScheduleHandler) GenerateICalToken(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}
	sessionIDStr, _ := getSessionID(c)

	// 先拉一次课表，确保订阅时 Redis 里有课表快照
	if _, err := service.GetJwService().GetFullSchedule(sessionIDStr, 20); err != nil {
		serviceError(c, err)
		return
	}

	// 生成随机 token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		response.Error(c, http.StatusInternalServerError, "生成 token 失败")
		return
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	if err := service.SaveICalToken(uid, token); err != nil {
		response.Error(c, http.StatusInternalServerError, "保存 token 失败")
		return
	}

	// 返回订阅 URL
	subscribeURL := fmt.Sprintf("%s/api/schedule/ical/subscribe?token=%s", getBaseURL(c), token)
	response.Success(c, gin.H{
		"token":    token,
		"url":      subscribeURL,
		"webcal":   "webcal://" + strings.TrimPrefix(subscribeURL, "https://"),
		"expireAt": time.Now().Add(service.ICalTokenTTL).Format("2006-01-02"),
	})
}

// SubscribeICal 通过 token 免登录订阅 iCal（读取用户最近一次登录时的课表快照）
func (h *ScheduleHandler) SubscribeICal(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(400, gin.H{"error": "缺少 token 参数"})
		return
	}

	uid, err := service.LookupICalToken(token)
	if errors.Is(err, service.ErrNotFound) {
		c.JSON(401, gin.H{"error": "token 已过期或无效"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "读取 token 失败"})
		return
	}

	fullSchedule, err := service.GetScheduleSnapshot(uid)
	if err != nil {
		c.JSON(500, gin.H{"error": "获取课表失败"})
		return
	}

	// 生成 iCal
	ical := generateICalContent(fullSchedule)

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"课程表-%s.ics\"", fullSchedule.StudentName))
	c.String(200, ical)
}

// GetICal 获取当前用户的 iCal（需登录）
func (h *ScheduleHandler) GetICal(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	jwSvc := service.GetJwService()
	fullSchedule, err := jwSvc.GetFullSchedule(sessionIDStr, 20)
	if err != nil {
		serviceError(c, err)
		return
	}

	ical := generateICalContent(fullSchedule)

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"课程表-%s.ics\"", fullSchedule.StudentName))
	c.String(200, ical)
}

// GetICalTokenInfo 获取当前用户的 token 信息
func (h *ScheduleHandler) GetICalTokenInfo(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	token, expireAt, err := service.GetUserICalToken(uid)
	if errors.Is(err, service.ErrNotFound) {
		response.Success(c, nil)
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "读取 token 失败")
		return
	}

	response.Success(c, gin.H{
		"token":    token,
		"expireAt": expireAt.Format("2006-01-02"),
	})
}

// generateICalContent 生成 iCalendar 格式内容
func generateICalContent(schedule *model.FullSchedule) string {
	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//jww.p//Course Schedule//CN\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")
	sb.WriteString("X-WR-CALNAME:课程表\r\n")
	sb.WriteString("X-WR-TIMEZONE:Asia/Shanghai\r\n")

	// 计算学期第一天（如果已知）
	semesterStart := schedule.SemesterStart
	if semesterStart == "" {
		semesterStart = time.Now().Format("2006-01-02")
	}
	startDate, _ := time.Parse("2006-01-02", semesterStart)

	for _, course := range schedule.Courses {
		for _, week := range course.Weeks {
			// 计算这门课在这一周的日期
			courseDate := startDate.AddDate(0, 0, (week-1)*7+(course.DayOfWeek-1))
			startTime := getPeriodStartTime(course.PeriodStart)
			endTime := getPeriodEndTime(course.PeriodStart, course.Periods)

			dtStart := fmt.Sprintf("%sT%s", courseDate.Format("20060102"), startTime)
			dtEnd := fmt.Sprintf("%sT%s", courseDate.Format("20060102"), endTime)

			sb.WriteString("BEGIN:VEVENT\r\n")
			sb.WriteString(fmt.Sprintf("UID:%s-%d-%d@jwschedule\r\n", course.Name, week, course.DayOfWeek))
			sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", dtStart))
			sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", dtEnd))
			sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICalText(course.Name)))
			sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICalText(course.Room)))
			sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\\n教师: %s\\n第%d周/周%d/第%d节\r\n",
				escapeICalText(course.Name), escapeICalText(course.Teacher), week, course.DayOfWeek, course.PeriodStart))
			sb.WriteString(fmt.Sprintf("RRULE:FREQ=WEEKLY;COUNT=%d;BYDAY=%s\r\n",
				len(course.Weeks), getWeekdayICal(course.DayOfWeek)))
			sb.WriteString("END:VEVENT\r\n")
		}
	}

	sb.WriteString("END:VCALENDAR\r\n")
	return sb.String()
}

func getWeekdayICal(dayOfWeek int) string {
	days := []string{"MO", "TU", "WE", "TH", "FR", "SA", "SU"}
	if dayOfWeek >= 1 && dayOfWeek <= 7 {
		return days[dayOfWeek-1]
	}
	return "MO"
}

// getPeriodStartTime 节次开始时间（"080000"），作息表见 service.PeriodTimeRange
func getPeriodStartTime(periodStart int) string {
	start, _ := service.PeriodTimeRange(periodStart, 1)
	if start == "" {
		start = "08:00"
	}
	return strings.ReplaceAll(start, ":", "") + "00"
}

// getPeriodEndTime 连续节次的结束时间（"094000"）
func getPeriodEndTime(periodStart, periods int) string {
	_, end := service.PeriodTimeRange(periodStart, periods)
	if end == "" {
		end = "08:45"
	}
	return strings.ReplaceAll(end, ":", "") + "00"
}

func escapeICalText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func getBaseURL(c *gin.Context) string {
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + c.Request.Host
}
