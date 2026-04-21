package handler

import (
	"log/slog"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"jww/internal/model"
	"jww/internal/service"
	"jww/pkg/response"
)

type ScoreHandler struct{}

func NewScoreHandler() *ScoreHandler {
	return &ScoreHandler{}
}

func (h *ScoreHandler) GetScore(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	semester := c.Query("semester")

	slog.Info("GetScore handler", "sessionID", sessionIDStr, "semester", semester)

	jwSvc := service.GetJwService()
	scores, err := jwSvc.GetScorePage(sessionIDStr, semester)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, scores)
}

func (h *ScoreHandler) GetSemesters(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	jwSvc := service.GetJwService()

	// P4 修复：先查缓存，命中则直接提取学期列表，避免触发全量 HTTP 抓取
	semesters, err := jwSvc.GetCachedSemesters(sessionIDStr)
	if err == nil && len(semesters) > 0 {
		response.Success(c, semesters)
		return
	}

	// 缓存未命中，走原有逻辑
	scores, err := jwSvc.GetScorePage(sessionIDStr, "")
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 去重获取学期列表
	semesterSet := make(map[string]struct{})
	for _, s := range scores {
		semester := s.Year + "-" + s.Semester
		semesterSet[semester] = struct{}{}
	}

	semesters = make([]string, 0, len(semesterSet))
	for k := range semesterSet {
		semesters = append(semesters, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(semesters))) // 降序，最新学期在前

	response.Success(c, semesters)
}

// gradeToFloat converts any grade to float64 for GPA calculation
func gradeToFloat(g any) float64 {
	switch v := g.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		// 不合格/缓考/作弊 等返回 -1
		return -1
	default:
		return -1
	}
}

// isFailed checks if a grade represents failure (< 60 or non-pass string)
func isFailed(g any) bool {
	f := gradeToFloat(g)
	if f >= 0 {
		return f < 60
	}
	return false
}

// GetScoreStats 返回成绩统计（GPA/学分/逐学期趋势/挂科列表）
func (h *ScoreHandler) GetScoreStats(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	jwSvc := service.GetJwService()
	scores, err := jwSvc.GetScorePage(sessionIDStr, "")
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if len(scores) == 0 {
		response.Success(c, gin.H{
			"totalCredits":  0,
			"weightedGPA":   0,
			"simpleGPA":     0,
			"failedCount":   0,
			"totalCourses":  0,
			"semesterStats": []any{},
		})
		return
	}

	// 按学期分组
	type semKey struct{ year, term string }
	semMap := make(map[semKey][]model.Score)
	for _, sc := range scores {
		k := semKey{sc.Year, sc.Semester}
		semMap[k] = append(semMap[k], sc)
	}

	var totalCredits float64
	var totalWeighted float64
	var totalGPA float64
	var failedCount int
	var semesterStats []any

	// 学期排序（降序）
	keys := make([]semKey, 0, len(semMap))
	for k := range semMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].year != keys[j].year {
			return keys[i].year > keys[j].year
		}
		return keys[i].term > keys[j].term
	})

	var highestGPA, lowestGPA float64 = -1, 5
	var highestCourse, lowestCourse model.Score

	for _, k := range keys {
		courses := semMap[k]
		var semCredits float64
		var semWeighted, semGPA float64
		var semFailed int

		for _, sc := range courses {
			credit := sc.Credit
			gpa := sc.GPA

			semCredits += credit
			semWeighted += credit * gpa
			if gpa > 0 {
				semGPA += gpa
			}
			if isFailed(sc.Grade) {
				semFailed++
				failedCount++
			}

			// 最高/最低
			if gpa > 0 && (highestGPA < 0 || gpa > highestGPA) {
				highestGPA = gpa
				highestCourse = sc
			}
			if gpa > 0 && gpa < lowestGPA {
				lowestGPA = gpa
				lowestCourse = sc
			}
		}

		if semCredits > 0 {
			semGPA = semWeighted / semCredits
		}
		totalCredits += semCredits
		totalWeighted += semWeighted
		if len(courses) > 0 {
			totalGPA += semGPA
		}

		semName := k.year + "-" + k.term
		semesterStats = append(semesterStats, gin.H{
			"semester":    semName,
			"year":        k.year,
			"term":        k.term,
			"credits":     semCredits,
			"gpa":         semGPA,
			"courseCount": len(courses),
			"failedCount": semFailed,
		})
	}

	weightedGPA := 0.0
	if totalCredits > 0 {
		weightedGPA = totalWeighted / totalCredits
	}
	simpleGPA := 0.0
	if len(keys) > 0 {
		simpleGPA = totalGPA / float64(len(keys))
	}

	// 收集挂科课程
	var failedCourses []any
	for _, sc := range scores {
		if isFailed(sc.Grade) {
			failedCourses = append(failedCourses, sc)
		}
	}

	response.Success(c, gin.H{
		"totalCredits":   totalCredits,
		"weightedGPA":    weightedGPA,
		"simpleGPA":      simpleGPA,
		"failedCount":    failedCount,
		"totalCourses":   len(scores),
		"semesterStats":  semesterStats,
		"highestCourse":  highestCourse,
		"lowestCourse":   lowestCourse,
		"failedCourses":  failedCourses,
	})
}
