package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

	// P1 修复：week == nil 时复用第一次 GetSchedulePage 的结果，不再二次请求
	if week == nil {
		// 先获取当前页面（可能是入口页或当前周的课表）来获取 semesterStart
		html, err := jwSvc.GetSchedulePage(sessionIDStr, nil)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}

		parsed, err := jwSvc.ParseSchedule(html)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}

		// 计算真实当前周
		realCurrentWeek := 1
		if parsed.SemesterStart != "" {
			realCurrentWeek = service.CalcRealCurrentWeek(parsed.SemesterStart, 20)
		}

		// 如果入口页直接是课表（EntryHtml），GetSchedulePage 已直接返回，无需二次请求
		// 否则复用第一次拿到的 html，只在周数不同时才二次请求
		if parsed.EntryHtml == "" && realCurrentWeek != parsed.DQZ {
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
