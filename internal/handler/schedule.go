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
		if err == nil {
			week = &w
		}
	}

	jwSvc := service.GetJwService()

	// 如果没有指定 week，先获取课表以提取学期起始日，计算真实当前周
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

		// 用真实当前周再次获取课表
		html, err = jwSvc.GetSchedulePage(sessionIDStr, &realCurrentWeek)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}

		schedule, err := jwSvc.ParseSchedule(html)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}

		schedule.CurrentWeek = realCurrentWeek
		response.Success(c, schedule)
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
		if m, err := strconv.Atoi(maxWeekStr); err == nil {
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
