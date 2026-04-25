package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"jww/internal/middleware"
	"jww/internal/model"
	"jww/internal/service"
	"jww/pkg/response"
)

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
	secureCookie     bool
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

	accessToken, err := h.signToken(req.Username, req.SessionID, h.jwtSecret, h.jwtExpires)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	refreshTokenID := fmt.Sprintf("%s_%d", req.Username, time.Now().UnixNano())
	refreshToken, err := h.signTokenWithID(req.Username, req.SessionID, refreshTokenID, h.jwtRefreshSecret, h.refreshExpires)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	tokenSvc := service.GetTokenService()
	expiresAt := time.Now().Add(h.refreshExpires)
	if err := tokenSvc.StoreRefreshToken(req.Username, refreshTokenID, "web", c.Request.UserAgent(), expiresAt); err != nil {
		log.Printf("[Login] StoreRefreshToken error: %v", err)
		response.Error(c, http.StatusInternalServerError, "存储 Token 失败")
		return
	}

	c.SetCookie("refreshToken", refreshToken, int(h.refreshExpires.Seconds()), "/", "", h.secureCookie, true)
	log.Printf("[Login] Cookie set, refreshTokenID: %s", refreshTokenID)

	response.SuccessWithToken(c, accessToken, req.Username, int(h.jwtExpires.Seconds()))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshTokenStr, err := c.Cookie("refreshToken")
	log.Printf("[Refresh] Cookie error: %v", err)
	if err != nil {
		log.Printf("[Refresh] No cookie found: %v", err)
		response.ErrorWithInfo(c, "请重新登录")
		return
	}
	log.Printf("[Refresh] Cookie found, length: %d", len(refreshTokenStr))

	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(refreshTokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtRefreshSecret), nil
	})

	if err != nil {
		log.Printf("[Refresh] JWT parse error: %v", err)
		response.ErrorWithInfo(c, "登录已过期，请重新登录")
		return
	}

	if !token.Valid {
		log.Printf("[Refresh] Token invalid")
		response.ErrorWithInfo(c, "登录已过期，请重新登录")
		return
	}

	log.Printf("[Refresh] Token parsed, TokenID: %s, UID: %s", claims.TokenID, claims.UID)

	tokenSvc := service.GetTokenService()
	userID, err := tokenSvc.ValidateRefreshToken(claims.TokenID)
	if err != nil {
		log.Printf("[Refresh] ValidateRefreshToken error: %v", err)
		response.ErrorWithInfo(c, "登录已过期，请重新登录")
		return
	}

	if userID != claims.UID {
		log.Printf("[Refresh] UserID mismatch: %s != %s", userID, claims.UID)
		response.ErrorWithInfo(c, "登录已过期，请重新登录")
		return
	}

	log.Printf("[Refresh] Validation success, userID: %s", userID)

	newAccessToken, err := h.signToken(claims.UID, claims.SessionID, h.jwtSecret, h.jwtExpires)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	newRefreshTokenID, err := tokenSvc.RotateRefreshToken(claims.TokenID, claims.UID, "web", c.Request.UserAgent(), time.Now().Add(h.refreshExpires))
	if err != nil {
		log.Printf("[Refresh] RotateRefreshToken error: %v", err)
		response.Error(c, http.StatusInternalServerError, "刷新 Token 失败")
		return
	}

	newRefreshToken, err := h.signTokenWithID(claims.UID, claims.SessionID, newRefreshTokenID, h.jwtRefreshSecret, h.refreshExpires)
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
	refreshTokenStr, err := c.Cookie("refreshToken")
	if err == nil && refreshTokenStr != "" {
		claims := &middleware.Claims{}
		_, err := jwt.ParseWithClaims(refreshTokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(h.jwtRefreshSecret), nil
		})
		if err == nil {
			tokenSvc := service.GetTokenService()
			tokenSvc.RevokeRefreshToken(claims.TokenID, claims.UID)
		}
	}

	c.SetCookie("refreshToken", "", -1, "/", "", h.secureCookie, true)
	response.SuccessWithInfo(c, "已退出登录")
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, _ := c.Get("uid")
	response.Success(c, gin.H{
		"uid": uid,
	})
}

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

func (h *AuthHandler) signTokenWithID(uid, sessionId, tokenID, secret string, expire time.Duration) (string, error) {
	claims := &middleware.Claims{
		UID:       uid,
		SessionID: sessionId,
		TokenID:   tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
