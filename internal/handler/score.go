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
