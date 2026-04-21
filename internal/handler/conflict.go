package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"jww/internal/model"
	"jww/internal/service"
	"jww/pkg/response"
)

// ConflictPair 课程冲突对
type ConflictPair struct {
	CourseA       model.Course `json:"courseA"`
	CourseB       model.Course `json:"courseB"`
	ConflictWeeks []int       `json:"conflictWeeks"`
}

// GetConflicts 检测课程冲突
func (h *ScheduleHandler) GetConflicts(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	// 获取全学期课表
	jwSvc := service.GetJwService()
	fullSchedule, err := jwSvc.GetFullSchedule(sessionIDStr, 20)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// O(n²) 冲突检测
	conflicts := detectConflicts(fullSchedule.Courses)
	response.Success(c, conflicts)
}

// detectConflicts 检测课程冲突
// 冲突条件：同 dayOfWeek + period 区间重叠 + weeks[] 交集非空
func detectConflicts(courses []model.Course) []ConflictPair {
	var result []ConflictPair

	for i := 0; i < len(courses); i++ {
		for j := i + 1; j < len(courses); j++ {
			a, b := courses[i], courses[j]

			// 不同星期，肯定不冲突
			if a.DayOfWeek != b.DayOfWeek {
				continue
			}

			// period 区间重叠检测
			aEnd := a.PeriodStart + a.Periods - 1
			bEnd := b.PeriodStart + b.Periods - 1
			if aEnd < b.PeriodStart || bEnd < a.PeriodStart {
				continue
			}

			// weeks[] 交集
			intersect := intersectIntSlices(a.Weeks, b.Weeks)
			if len(intersect) == 0 {
				continue
			}

			result = append(result, ConflictPair{
				CourseA:       a,
				CourseB:       b,
				ConflictWeeks: intersect,
			})
		}
	}

	return result
}

// intersectIntSlices 计算两个 int 切片的交集
func intersectIntSlices(a, b []int) []int {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}

	bMap := make(map[int]bool, len(b))
	for _, v := range b {
		bMap[v] = true
	}

	var intersect []int
	for _, v := range a {
		if bMap[v] {
			intersect = append(intersect, v)
		}
	}
	return intersect
}
