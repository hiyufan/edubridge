package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthMiddleware struct {
	jwtSecret       string
	jwtRefreshSecret string
}

func NewAuthMiddleware(jwtSecret, jwtRefreshSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret:       jwtSecret,
		jwtRefreshSecret: jwtRefreshSecret,
	}
}

// Claims JWT Claims
type Claims struct {
	UID       string `json:"uid"`
	SessionID string `json:"sessionId"`
	jwt.RegisteredClaims
}

// AuthRequired JWT 鉴权中间件
func (m *AuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": 0,
				"info":   "未授权",
			})
			c.Abort()
			return
		}

		tokenStr := auth[7:]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(m.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": 0,
				"info":   "Token 无效或已过期",
			})
			c.Abort()
			return
		}

		c.Set("uid", claims.UID)
		c.Set("sessionId", claims.SessionID)
		c.Next()
	}
}
