package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Status int         `json:"status"`
	Info   string     `json:"info,omitempty"`
	Data   interface{} `json:"data,omitempty"`
	Token  string     `json:"token,omitempty"`
	UID    string     `json:"uid,omitempty"`
	Code   string     `json:"code,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{
		"status": 1,
		"data":   data,
	})
}

func SuccessWithToken(c *gin.Context, token string, uid string, expiresIn int) {
	c.JSON(200, gin.H{
		"status":    1,
		"info":      "登录成功",
		"token":     token,
		"uid":       uid,
		"expiresIn": expiresIn,
	})
}

func SuccessWithInfo(c *gin.Context, info string) {
	c.JSON(200, gin.H{
		"status": 1,
		"info":   info,
	})
}

func Error(c *gin.Context, httpStatus int, info string) {
	c.JSON(httpStatus, gin.H{
		"status": 0,
		"info":   info,
	})
}

func ErrorWithCode(c *gin.Context, httpStatus int, info string, code string) {
	c.JSON(httpStatus, gin.H{
		"status": 0,
		"info":   info,
		"code":   code,
	})
}

func ErrorWithInfo(c *gin.Context, info string) {
	// 建议使用 Error(c, http.StatusUnauthorized, info) 代替
	c.JSON(http.StatusUnauthorized, gin.H{
		"status": 0,
		"info":   info,
	})
}
