package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
	"jww/internal/service"
	"jww/pkg/response"
)

type CaptchaHandler struct{}

func NewCaptchaHandler() *CaptchaHandler {
	return &CaptchaHandler{}
}

func (h *CaptchaHandler) GetCaptcha(c *gin.Context) {
	// 优先使用请求中的 PHPSESSID，保持会话一致
	sessionID, err := c.Cookie("PHPSESSID")
	if err != nil || sessionID == "" {
		// 生成新的 session ID
		sessionIDBytes := make([]byte, 16)
		if _, err := rand.Read(sessionIDBytes); err != nil {
			response.Error(c, 500, "生成 sessionID 失败")
			return
		}
		sessionID = hex.EncodeToString(sessionIDBytes)
	}

	jwSvc := service.GetJwService()
	image, contentType, err := jwSvc.FetchCaptcha(sessionID)
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}

	// 设置 PHPSESSID Cookie
	c.SetCookie("PHPSESSID", sessionID, 30*60, "/", "", false, true)

	// 转换 content-type 到 MIME 字符串（如 image/jpeg -> "jpeg"）
	mimeStr := strings.TrimPrefix(contentType, "image/")

	// 返回 Base64 编码的图片
	base64Str := "data:image/" + mimeStr + ";base64," + base64.StdEncoding.EncodeToString(image)

	c.JSON(200, gin.H{
		"status":    1,
		"data":      base64Str,
		"sessionId": sessionID,
	})
}
