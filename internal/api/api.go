// Package api HTTP 层：路由、参数解析、统一的响应格式与错误处理。
// 不包含业务逻辑，只调用 academic / auth / notify / reminder。
//
// 响应格式：成功 {"status":1,"data":...}；失败 {"status":0,"info":"给用户看的说明","code":"可选错误码"}
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"jww/internal/academic"
	"jww/internal/auth"
	"jww/internal/monitor"
	"jww/internal/notify"
	"jww/internal/reminder"
)

// Deps API 依赖
type Deps struct {
	Academic      *academic.Service
	Auth          *auth.Service
	Notifier      *notify.Notifier
	Reminder      *reminder.Scheduler
	Monitor       *monitor.Monitor
	AllowedOrigin string
	SecureCookie  bool
}

// API HTTP 接口
type API struct {
	Deps
}

// New 创建 HTTP 接口
func New(d Deps) *API {
	return &API{Deps: d}
}

// Handler 构建路由
func (a *API) Handler() http.Handler {
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger(), a.cors())

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": 1}) })

	api := r.Group("/api")
	api.GET("/captcha", rateLimit(100, 5*time.Minute), a.captcha)
	api.POST("/auth/login", rateLimit(10, 5*time.Minute), a.login)
	api.POST("/auth/refresh", a.refresh)
	api.POST("/auth/logout", a.logout)
	api.GET("/schedule/ical/subscribe", a.subscribeICal)

	p := api.Group("", a.requireAuth())
	p.GET("/auth/me", a.me)

	p.GET("/schedule", a.weekSchedule)
	p.GET("/schedule/full", a.fullSchedule)
	p.GET("/schedule/conflicts", a.conflicts)
	p.GET("/schedule/diff", a.latestDiff)
	p.GET("/schedule/ical", a.ical)
	p.POST("/schedule/ical/token", a.createICalToken)
	p.GET("/schedule/ical/token-info", a.icalTokenInfo)

	p.GET("/score", a.scores)
	p.GET("/score/semesters", a.semesters)
	p.GET("/score/stats", a.scoreStats)

	p.POST("/webhook/register", a.registerWebhook)
	p.GET("/webhook/info", a.webhookInfo)
	p.POST("/webhook/trigger", a.triggerWebhook)
	p.GET("/notify/channels", a.channels)
	p.PUT("/notify/channels", a.saveChannels)
	p.POST("/notify/test", a.testNotify)
	p.GET("/notify/reminder", a.reminder)
	p.PUT("/notify/reminder", a.saveReminder)
	p.GET("/monitor/status", a.monitorStatus)
	return r
}

func (a *API) cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", a.AllowedOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
