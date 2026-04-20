package handler

import (
	"log/slog"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"jww/internal/service"
	"jww/pkg/response"
)

type ScoreHandler struct{}

func NewScoreHandler() *ScoreHandler {
	return &ScoreHandler{}
}

func (h *ScoreHandler) GetScore(c *gin.Context) {
	sessionID, _ := c.Get("sessionId")
	sessionIDStr := sessionID.(string)

	semester := c.Query("semester")

	slog.Info("GetScore handler", "sessionID", sessionIDStr, "semester", semester)

	jwSvc := service.GetJwService()
	scores, err := jwSvc.GetScorePage(sessionIDStr, semester)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("GetScore handler 返回", "count", len(scores))

	// 按学期统计
	semMap := make(map[string]int)
	for _, s := range scores {
		key := s.Year + "-" + s.Semester
		semMap[key]++
	}
	slog.Info("各学期成绩数", "detail", semMap)

	response.Success(c, scores)
}

func (h *ScoreHandler) GetSemesters(c *gin.Context) {
	sessionID, _ := c.Get("sessionId")
	sessionIDStr := sessionID.(string)

	slog.Info("GetSemesters handler", "sessionID", sessionIDStr)

	jwSvc := service.GetJwService()
	scores, err := jwSvc.GetScorePage(sessionIDStr, "")
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("GetSemesters 返回成绩数", "count", len(scores))

	// 去重获取学期列表
	semesterSet := make(map[string]struct{})
	for _, s := range scores {
		semester := s.Year + "-" + s.Semester
		semesterSet[semester] = struct{}{}
	}

	semesters := make([]string, 0, len(semesterSet))
	for k := range semesterSet {
		semesters = append(semesters, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(semesters))) // 降序，最新学期在前

	slog.Info("GetSemesters 返回学期列表", "semesters", semesters)
	response.Success(c, semesters)
}
