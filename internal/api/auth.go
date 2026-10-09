package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"jww/internal/auth"
)

const (
	refreshCookie = "refreshToken"
	captchaCookie = "PHPSESSID" // 前端获取验证码时的会话标识
	ctxUID        = "uid"
)

func uidOf(c *gin.Context) string { return c.GetString(ctxUID) }

// requireAuth 校验 Authorization: Bearer <access token>，把学号放进上下文
func (a *API) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !found || token == "" {
			failWith(c, http.StatusUnauthorized, "未登录", "")
			return
		}
		uid, err := a.Auth.Verify(token)
		if err != nil {
			failWith(c, http.StatusUnauthorized, "Token 无效或已过期", "")
			return
		}
		c.Set(ctxUID, uid)
		c.Next()
	}
}

// captcha 获取验证码；sessionId 需在登录时带回，以使用同一个教务会话
func (a *API) captcha(c *gin.Context) {
	sessionID, err := c.Cookie(captchaCookie)
	if err != nil || sessionID == "" {
		b := make([]byte, 16)
		rand.Read(b)
		sessionID = hex.EncodeToString(b)
	}
	img, contentType, err := a.Academic.Captcha(c.Request.Context(), sessionID)
	if err != nil {
		fail(c, err)
		return
	}
	c.SetCookie(captchaCookie, sessionID, 30*60, "/", "", a.SecureCookie, true)
	c.JSON(http.StatusOK, gin.H{
		"status":    1,
		"data":      "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(img),
		"sessionId": sessionID,
	})
}

type loginRequest struct {
	SessionID string `json:"sessionId"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Captcha   string `json:"captcha"`
	LoginType string `json:"loginType"`
}

func (a *API) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" || req.Captcha == "" {
		badRequest(c, "请填写学号、密码和验证码")
		return
	}
	if req.SessionID == "" {
		badRequest(c, "请先获取验证码")
		return
	}
	ctx := c.Request.Context()
	if err := a.Academic.Login(ctx, req.SessionID, req.Username, req.Password, req.Captcha, req.LoginType); err != nil {
		fail(c, err)
		return
	}
	access, refresh, err := a.Auth.Issue(ctx, req.Username)
	if err != nil {
		fail(c, err)
		return
	}
	a.setRefreshCookie(c, refresh)
	c.JSON(http.StatusOK, gin.H{
		"status": 1, "info": "登录成功", "token": access, "uid": req.Username,
		"expiresIn": int(auth.AccessTTL.Seconds()),
	})
}

func (a *API) setRefreshCookie(c *gin.Context, token string) {
	maxAge := int(auth.RefreshTTL.Seconds())
	if token == "" {
		maxAge = -1
	}
	c.SetCookie(refreshCookie, token, maxAge, "/", "", a.SecureCookie, true)
}

func (a *API) refresh(c *gin.Context) {
	token, err := c.Cookie(refreshCookie)
	if err != nil || token == "" {
		failWith(c, http.StatusUnauthorized, "请重新登录", "")
		return
	}
	_, access, refresh, err := a.Auth.Refresh(c.Request.Context(), token)
	if err != nil {
		fail(c, err)
		return
	}
	a.setRefreshCookie(c, refresh)
	c.JSON(http.StatusOK, gin.H{"status": 1, "token": access, "expiresIn": int(auth.AccessTTL.Seconds())})
}

// logout 作废凭证、清理教务登录态并停止后台监控（不要求 access token 仍有效）
func (a *API) logout(c *gin.Context) {
	ctx := c.Request.Context()
	if token, err := c.Cookie(refreshCookie); err == nil && token != "" {
		uid, err := a.Auth.Revoke(ctx, token)
		if err != nil {
			slog.Warn("revoke refresh token failed", "err", err)
		}
		if uid != "" {
			if err := a.Academic.Logout(ctx, uid); err != nil {
				slog.Warn("logout failed", "uid", uid, "err", err)
			}
		}
	}
	a.setRefreshCookie(c, "")
	c.JSON(http.StatusOK, gin.H{"status": 1, "info": "已退出登录"})
}

func (a *API) me(c *gin.Context) {
	ok(c, gin.H{"uid": uidOf(c)})
}
