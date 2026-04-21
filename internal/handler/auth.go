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

// getSessionID 安全提取 sessionId，带类型断言校验
func getSessionID(c *gin.Context) (string, bool) {
	val, exists := c.Get("sessionId")
	if !exists {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}

type AuthHandler struct {
	jwtSecret        string
	jwtRefreshSecret string
	jwtExpires       time.Duration
	refreshExpires   time.Duration
	secureCookie     bool // SEC-2 修复：从配置读取 Secure 标志
}

func NewAuthHandler(jwtSecret, jwtRefreshSecret string, secureCookie bool) *AuthHandler {
	return &AuthHandler{
		jwtSecret:        jwtSecret,
		jwtRefreshSecret: jwtRefreshSecret,
		jwtExpires:       2 * time.Hour,
		refreshExpires:   30 * 24 * time.Hour,
		secureCookie:     secureCookie,
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

	// 生成 Refresh Token 并设置 HttpOnly Cookie（SEC-2 修复：Secure 标志从配置读取）
	refreshToken, err := h.signToken(req.Username, req.SessionID, h.jwtRefreshSecret, h.refreshExpires)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	c.SetCookie("refreshToken", refreshToken, int(h.refreshExpires.Seconds()), "/", "", h.secureCookie, true)

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
		// SEC-1: 校验签名算法，防止 alg:none 攻击
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
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

	// BUG-9 修复：Refresh Token 滑动更新 - 续签后重新签发 Refresh Token 并更新 Cookie 过期时间
	newRefreshToken, err := h.signToken(claims.UID, claims.SessionID, h.jwtRefreshSecret, h.refreshExpires)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成 Token 失败")
		return
	}
	c.SetCookie("refreshToken", newRefreshToken, int(h.refreshExpires.Seconds()), "/", "", h.secureCookie, true)

	c.JSON(http.StatusOK, gin.H{
		"status":    1,
		"token":     newAccessToken,
		"expiresIn": int(h.jwtExpires.Seconds()),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("refreshToken", "", -1, "/", "", h.secureCookie, true)
	response.SuccessWithInfo(c, "已退出登录")
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, _ := c.Get("uid")
	response.Success(c, gin.H{
		"uid": uid,
	})
}

// signToken 使用 *middleware.Claims 结构体签发 Token（与解析方保持一致）
func (h *AuthHandler) signToken(uid, sessionId, secret string, expire time.Duration) (string, error) {
	claims := &middleware.Claims{
		UID:       uid,
		SessionID: sessionId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
