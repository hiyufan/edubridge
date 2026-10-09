package api

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"jww/internal/auth"
	"jww/internal/jwclient"
	"jww/internal/notify"
	"jww/internal/reminder"
	"jww/internal/store"
)

// 错误码（前端据此做特殊处理）
const (
	codeJWSessionExpired = "JW_SESSION_EXPIRED" // 教务登录失效，需要重新输入验证码登录
)

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"status": 1, "data": data})
}

func failWith(c *gin.Context, status int, info, code string) {
	body := gin.H{"status": 0, "info": info}
	if code != "" {
		body["code"] = code
	}
	c.AbortWithStatusJSON(status, body)
}

func badRequest(c *gin.Context, info string) {
	failWith(c, http.StatusBadRequest, info, "")
}

// fail 把错误映射成给用户看的响应；未预期的错误只记日志，不把内部信息返回给前端
func fail(c *gin.Context, err error) {
	var (
		loginErr    *jwclient.LoginError
		invalidErr  *notify.ValidationError
		reminderErr *reminder.ValidationError
	)
	switch {
	case errors.As(err, &loginErr):
		badRequest(c, loginErr.Info)
	case errors.As(err, &invalidErr):
		badRequest(c, invalidErr.Msg)
	case errors.As(err, &reminderErr):
		badRequest(c, reminderErr.Msg)
	case errors.Is(err, jwclient.ErrSessionExpired):
		failWith(c, http.StatusUnauthorized, jwclient.ErrSessionExpired.Error(), codeJWSessionExpired)
	case errors.Is(err, auth.ErrInvalidToken):
		failWith(c, http.StatusUnauthorized, err.Error(), "")
	case errors.Is(err, jwclient.ErrUnavailable):
		slog.Warn("jw unavailable", "path", c.FullPath(), "err", err)
		failWith(c, http.StatusBadGateway, jwclient.ErrUnavailable.Error(), "")
	case errors.Is(err, notify.ErrNoChannel):
		badRequest(c, err.Error())
	case errors.Is(err, store.ErrNotFound):
		failWith(c, http.StatusNotFound, "数据不存在或已过期", "")
	default:
		slog.Error("request failed", "path", c.FullPath(), "err", err)
		failWith(c, http.StatusInternalServerError, "服务器内部错误，请稍后再试", "")
	}
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http", "method", c.Request.Method, "path", c.FullPath(), "status", c.Writer.Status(), "duration", time.Since(start))
	}
}
