package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"jww/internal/model"
	"jww/internal/service"
	"jww/pkg/response"
)

// iCalTokenStore 内存存储 iCal token（token -> sessionID）
var (
	iCalTokenStore = make(map[string]*iCalTokenEntry)
	iCalMu         sync.RWMutex
)

type iCalTokenEntry struct {
	SessionID  string
	ExpireTime time.Time
}

// GenerateICalToken 生成 90 天订阅 token
func (h *ScheduleHandler) GenerateICalToken(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	// 生成随机 token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// 存储，90 天过期
	iCalMu.Lock()
	iCalTokenStore[token] = &iCalTokenEntry{
		SessionID:  sessionIDStr,
		ExpireTime: time.Now().Add(90 * 24 * time.Hour),
	}
	iCalMu.Unlock()

	// 返回订阅 URL
	subscribeURL := fmt.Sprintf("%s/api/schedule/ical/subscribe?token=%s", getBaseURL(c), token)
	response.Success(c, gin.H{
		"token":    token,
		"url":      subscribeURL,
		"webcal":   "webcal://" + strings.TrimPrefix(subscribeURL, "https://"),
		"expireAt": time.Now().Add(90 * 24 * time.Hour).Format("2006-01-02"),
	})
}

// SubscribeICal 通过 token 免登录订阅 iCal
func (h *ScheduleHandler) SubscribeICal(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(400, gin.H{"error": "缺少 token 参数"})
		return
	}

	iCalMu.RLock()
	entry, exists := iCalTokenStore[token]
	iCalMu.RUnlock()

	if !exists || time.Now().After(entry.ExpireTime) {
		c.JSON(401, gin.H{"error": "token 已过期或无效"})
		return
	}

	// 获取课表
	jwSvc := service.GetJwService()
	fullSchedule, err := jwSvc.GetFullSchedule(entry.SessionID, 20)
	if err != nil || fullSchedule == nil {
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
	if err != nil || fullSchedule == nil {
		response.Error(c, http.StatusInternalServerError, "获取课表失败")
		return
	}

	ical := generateICalContent(fullSchedule)

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"课程表-%s.ics\"", fullSchedule.StudentName))
	c.String(200, ical)
}

// GetICalTokenInfo 获取当前用户的 token 信息
func (h *ScheduleHandler) GetICalTokenInfo(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	iCalMu.RLock()
	defer iCalMu.RUnlock()

	var tokenInfo *struct {
		Token    string `json:"token"`
		ExpireAt string `json:"expireAt"`
	}

	for token, entry := range iCalTokenStore {
		if entry.SessionID == sessionIDStr && time.Now().Before(entry.ExpireTime) {
			tokenInfo = &struct {
				Token    string `json:"token"`
				ExpireAt string `json:"expireAt"`
			}{
				Token:    token,
				ExpireAt: entry.ExpireTime.Format("2006-01-02"),
			}
			break
		}
	}

	if tokenInfo == nil {
		response.Success(c, nil)
		return
	}
	response.Success(c, tokenInfo)
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

func getPeriodStartTime(periodStart int) string {
	// 前两节 8:00，后续每节 45 分钟
	times := []string{"080000", "081500", "090000", "091500", "100000", "101500", "110000", "111500",
		"140000", "141500", "150000", "151500", "160000", "161500", "170000", "171500",
		"190000", "191500", "200000", "201500", "210000", "211500", "220000", "221500"}
	idx := periodStart - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(times) {
		idx = len(times) - 1
	}
	return times[idx]
}

func getPeriodEndTime(periodStart, periods int) string {
	// 简单计算：每节课 45 分钟
	startMinutes := (periodStart - 1) * 45
	endMinutes := startMinutes + periods*45
	hour := 8 + endMinutes/60
	minute := endMinutes % 60
	return fmt.Sprintf("%02d%02d00", hour, minute)
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
