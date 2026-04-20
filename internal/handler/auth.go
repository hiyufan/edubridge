package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"jww/internal/middleware"
	"jww/internal/model"
	"jww/internal/service"
	"jww/pkg/response"
)

type AuthHandler struct {
	jwtSecret        string
	jwtRefreshSecret string
	jwtExpires       time.Duration
	refreshExpires   time.Duration
}

func NewAuthHandler(jwtSecret, jwtRefreshSecret string) *AuthHandler {
	return &AuthHandler{
		jwtSecret:        jwtSecret,
		jwtRefreshSecret: jwtRefreshSecret,
		jwtExpires:       2 * time.Hour,
		refreshExpires:   30 * 24 * time.Hour,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "缺少必要参数")
		return
	}

	if req.Username == "" || req.Password == "" || req.Captcha == "" {
		response.Error(c, http.StatusBadRequest, "缺少必要参数")
		return
	}

	if req.SessionID == "" {
		response.Error(c, http.StatusBadRequest, "缺少 sessionId，请先获取验证码")
		return
	}

	jwSvc := service.GetJwService()
	if err := jwSvc.Login(req.SessionID, req.Username, req.Password, req.Captcha, req.LoginType); err != nil {
		response.ErrorWithCode(c, http.StatusUnauthorized, err.Error(), "")
		return
	}

	// 生成 Access Token
	accessToken, err := h.signToken(req.Username, req.SessionID, h.jwtSecret, h.jwtExpires)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	// 生成 Refresh Token 并设置 HttpOnly Cookie
	refreshToken, err := h.signToken(req.Username, req.SessionID, h.jwtRefreshSecret, h.refreshExpires)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	c.SetCookie("refreshToken", refreshToken, int(h.refreshExpires.Seconds()), "/", "", false, true)

	response.SuccessWithToken(c, accessToken, req.Username, int(h.jwtExpires.Seconds()))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshTokenStr, err := c.Cookie("refreshToken")
	if err != nil {
		response.ErrorWithInfo(c, "请重新登录")
		return
	}

	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(refreshTokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtRefreshSecret), nil
	})

	if err != nil || !token.Valid {
		response.ErrorWithInfo(c, "登录已过期，请重新登录")
		return
	}

	// 生成新的 Access Token
	newAccessToken, err := h.signToken(claims.UID, claims.SessionID, h.jwtSecret, h.jwtExpires)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    1,
		"token":     newAccessToken,
		"expiresIn": int(h.jwtExpires.Seconds()),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("refreshToken", "", -1, "/", "", false, true)
	response.SuccessWithInfo(c, "已退出登录")
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, _ := c.Get("uid")
	response.Success(c, gin.H{
		"uid": uid,
	})
}

func (h *AuthHandler) signToken(uid, sessionId, secret string, expire time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"uid":       uid,
		"sessionId": sessionId,
		"exp":       time.Now().Add(expire).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
