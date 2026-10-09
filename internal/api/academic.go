package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"jww/internal/academic"
	"jww/internal/store"
)

// intQuery 读取整数参数，不合法或超出范围时返回 def
func intQuery(c *gin.Context, key string, def, lo, hi int) int {
	v, err := strconv.Atoi(c.Query(key))
	if err != nil || v < lo || v > hi {
		return def
	}
	return v
}

func (a *API) weekSchedule(c *gin.Context) {
	week := intQuery(c, "week", 0, 1, 30) // 0 表示当前周
	s, err := a.Academic.WeekSchedule(c.Request.Context(), uidOf(c), week)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, s)
}

func (a *API) fullSchedule(c *gin.Context) {
	maxWeek := academic.MaxWeek
	if v, err := strconv.Atoi(c.Query("maxWeek")); err == nil && v > 0 && v < maxWeek {
		maxWeek = v
	}
	s, err := a.Academic.FullSchedule(c.Request.Context(), uidOf(c), maxWeek)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, s)
}

func (a *API) conflicts(c *gin.Context) {
	list, err := a.Academic.Conflicts(c.Request.Context(), uidOf(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, list)
}

func (a *API) latestDiff(c *gin.Context) {
	d, err := a.Academic.LatestDiff(c.Request.Context(), uidOf(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, d)
}

func (a *API) scores(c *gin.Context) {
	list, err := a.Academic.Scores(c.Request.Context(), uidOf(c), c.Query("semester"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, list)
}

func (a *API) semesters(c *gin.Context) {
	list, err := a.Academic.Semesters(c.Request.Context(), uidOf(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, list)
}

func (a *API) scoreStats(c *gin.Context) {
	st, err := a.Academic.ScoreStats(c.Request.Context(), uidOf(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, st)
}

// ---- iCal ----

func sendICal(c *gin.Context, ics, studentName string) {
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape("课程表-"+studentName+".ics")))
	c.Data(http.StatusOK, "text/calendar; charset=utf-8", []byte(ics))
}

func (a *API) ical(c *gin.Context) {
	ics, s, err := a.Academic.ICal(c.Request.Context(), uidOf(c))
	if err != nil {
		fail(c, err)
		return
	}
	sendICal(c, ics, s.StudentName)
}

// subscribeICal 日历应用凭 token 免登录订阅
func (a *API) subscribeICal(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		badRequest(c, "缺少 token 参数")
		return
	}
	ics, s, err := a.Academic.ICalByToken(c.Request.Context(), token)
	if errors.Is(err, store.ErrNotFound) {
		failWith(c, http.StatusUnauthorized, "订阅链接已过期或无效，请在应用中重新生成", "")
		return
	}
	if err != nil {
		fail(c, err)
		return
	}
	sendICal(c, ics, s.StudentName)
}

func baseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}

func (a *API) createICalToken(c *gin.Context) {
	token, expireAt, err := a.Academic.CreateICalToken(c.Request.Context(), uidOf(c))
	if err != nil {
		fail(c, err)
		return
	}
	link := baseURL(c) + "/api/schedule/ical/subscribe?token=" + url.QueryEscape(token)
	ok(c, gin.H{
		"token":    token,
		"url":      link,
		"webcal":   "webcal://" + strings.TrimPrefix(strings.TrimPrefix(link, "https://"), "http://"),
		"expireAt": expireAt.Format("2006-01-02"),
	})
}

func (a *API) icalTokenInfo(c *gin.Context) {
	token, expireAt, err := a.Academic.ICalToken(c.Request.Context(), uidOf(c))
	if errors.Is(err, store.ErrNotFound) {
		ok(c, nil)
		return
	}
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"token": token, "expireAt": expireAt.Format("2006-01-02")})
}
