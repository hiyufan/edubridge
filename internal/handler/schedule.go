package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"jww/internal/model"
	"jww/internal/service"
	"jww/pkg/response"
)

type ScheduleHandler struct{}

func NewScheduleHandler() *ScheduleHandler {
	return &ScheduleHandler{}
}

func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	weekStr := c.Query("week")
	var week *int
	if weekStr != "" {
		w, err := strconv.Atoi(weekStr)
		// B11 修复：week 参数范围校验，防止 0 或负数下发给教务系统
		if err == nil && w >= 1 && w <= 30 {
			week = &w
		}
	}

	jwSvc := service.GetJwService()

	var html string
	var parsed *model.Schedule

	// BUG FIX：无 week 参数时，始终请求真实当前周的课表
	// 入口页可能是任意周，不能信任它返回的课程数据
	// 必须先用入口页获取 semesterStart，再反推真实当前周，最后请求该周课表
	if week == nil {
		// Step 1: 获取入口页以提取 semesterStart
		var err error
		html, err = jwSvc.GetSchedulePage(sessionIDStr, nil)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}

		parsed, err = jwSvc.ParseSchedule(html)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}

		// Step 2: 根据 semesterStart 计算真实当前周
		realCurrentWeek := 1
		if parsed.SemesterStart != "" {
			realCurrentWeek = service.CalcRealCurrentWeek(parsed.SemesterStart, 20)
		}

		// Step 3: 如果入口页不是当前周，重新请求真实当前周课表
		if parsed.Week != realCurrentWeek || len(parsed.Courses) == 0 {
			html, err = jwSvc.GetSchedulePage(sessionIDStr, &realCurrentWeek)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, err.Error())
				return
			}
			parsed, err = jwSvc.ParseSchedule(html)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, err.Error())
				return
			}
		}

		parsed.CurrentWeek = realCurrentWeek
		response.Success(c, parsed)
		return
	}

	html, err := jwSvc.GetSchedulePage(sessionIDStr, week)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	schedule, err := jwSvc.ParseSchedule(html)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 计算真实当前周
	if schedule.SemesterStart != "" {
		schedule.CurrentWeek = service.CalcRealCurrentWeek(schedule.SemesterStart, 20)
	}

	response.Success(c, schedule)
}

func (h *ScheduleHandler) GetFullSchedule(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	maxWeek := 20
	if maxWeekStr := c.Query("maxWeek"); maxWeekStr != "" {
		if m, err := strconv.Atoi(maxWeekStr); err == nil && m > 0 {
			if m > 20 {
				m = 20
			}
			maxWeek = m
		}
	}

	jwSvc := service.GetJwService()
	schedule, err := jwSvc.GetFullSchedule(sessionIDStr, maxWeek)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, schedule)
}
