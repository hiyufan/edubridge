package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jww/internal/service"
	"jww/pkg/response"
)

// RegisterWebhook 注册 Webhook（写入 service 层的统一存储）
func (h *ScheduleHandler) RegisterWebhook(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	var req struct {
		URL    string `json:"url" binding:"required,url"`
		Secret string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "缺少 url 参数")
		return
	}

	jwSvc := service.GetJwService()
	jwSvc.RegisterWebhookURL(sessionIDStr, req.URL, req.Secret)

	response.Success(c, gin.H{"message": "webhook 注册成功"})
}

// TriggerWebhook 手动触发一次 Webhook 推送
func (h *ScheduleHandler) TriggerWebhook(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	jwSvc := service.GetJwService()
	diff := jwSvc.GetLatestScheduleDiff(sessionIDStr)

	summary := summarizeDiff(diff)
	entry := jwSvc.GetWebhookEntry(sessionIDStr)
	if entry == nil {
		response.Error(c, http.StatusBadRequest, "未注册 webhook")
		return
	}

	ok_ := sendWebhookPost(entry.URL, entry.Secret, diff)

	if ok_ {
		response.Success(c, gin.H{"message": "推送成功", "summary": summary})
	} else {
		response.Error(c, http.StatusInternalServerError, "推送失败: "+summary)
	}
}

// GetWebhookInfo 获取当前用户的 webhook 注册信息
func (h *ScheduleHandler) GetWebhookInfo(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	jwSvc := service.GetJwService()
	entry := jwSvc.GetWebhookEntry(sessionIDStr)

	if entry == nil {
		response.Success(c, gin.H{"registered": false})
		return
	}

	showSecret := ""
	if len(entry.Secret) > 8 {
		showSecret = entry.Secret[:8] + "***"
	}

	response.Success(c, gin.H{
		"registered":    true,
		"url":           entry.URL,
		"secret":        showSecret,
		"registeredAt":  entry.Registered.Format("2006-01-02 15:04"),
	})
}

// GetScheduleDiff 获取最近一次课表变动差分
func (h *ScheduleHandler) GetScheduleDiff(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	jwSvc := service.GetJwService()
	diff := jwSvc.GetLatestScheduleDiff(sessionIDStr)

	response.Success(c, diff)
}

func sendWebhookPost(url, secret string, diff *service.ScheduleDiff) bool {
	body, _ := json.Marshal(diff)

	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Hub-Signature-256", "sha256="+sig)
	}
	req.Header.Set("X-JWW-Event", "schedule-diff")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func summarizeDiff(diff *service.ScheduleDiff) string {
	if diff == nil {
		return "无变动"
	}
	parts := []string{}
	if len(diff.Added) > 0 {
		parts = append(parts, fmt.Sprintf("新增%d门", len(diff.Added)))
	}
	if len(diff.Removed) > 0 {
		parts = append(parts, fmt.Sprintf("删除%d门", len(diff.Removed)))
	}
	if len(diff.Changed) > 0 {
		parts = append(parts, fmt.Sprintf("变更%d门", len(diff.Changed)))
	}
	if len(parts) == 0 {
		return "无变动"
	}
	return strings.Join(parts, "，")
}

// GenerateWebhookSecret 生成随机 webhook secret
func generateWebhookSecret() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
