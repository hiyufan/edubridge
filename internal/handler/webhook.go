package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"jww/internal/service"
	"jww/pkg/response"
)

// RegisterWebhook 注册 Webhook（持久化到 Redis，按学号关联）
func (h *ScheduleHandler) RegisterWebhook(c *gin.Context) {
	var req struct {
		URL    string `json:"url" binding:"required,url"`
		Secret string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "缺少 url 参数")
		return
	}

	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}
	if err := service.SaveWebhook(uid, &service.WebhookEntry{
		URL:        req.URL,
		Secret:     req.Secret,
		Registered: time.Now(),
	}); err != nil {
		response.Error(c, http.StatusInternalServerError, "webhook 保存失败")
		return
	}

	response.Success(c, gin.H{"message": "webhook 注册成功"})
}

// TriggerWebhook 手动把最近一次课表变动推送到 Webhook
func (h *ScheduleHandler) TriggerWebhook(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	diff, err := service.GetLatestDiff(uid)
	if err != nil && !errors.Is(err, service.ErrNotFound) {
		response.Error(c, http.StatusInternalServerError, "读取课表变动失败")
		return
	}
	summary := diff.Summary()

	entry, err := service.GetWebhook(uid)
	if errors.Is(err, service.ErrNotFound) {
		response.Error(c, http.StatusBadRequest, "未注册 webhook")
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "读取 webhook 失败")
		return
	}
	err = service.SendWebhook(entry, &service.NotifyPayload{
		Event: service.EventScheduleDiff,
		Text:  summary,
		Data:  diff,
		Time:  time.Now(),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "推送失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "推送成功", "summary": summary})
}

// GetWebhookInfo 获取当前用户的 webhook 注册信息
func (h *ScheduleHandler) GetWebhookInfo(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	entry, err := service.GetWebhook(uid)
	if errors.Is(err, service.ErrNotFound) {
		response.Success(c, gin.H{"registered": false})
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "读取 webhook 失败")
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

// GetScheduleDiff 获取最近一次检测到的课表变动（没有则返回 null）
func (h *ScheduleHandler) GetScheduleDiff(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	diff, err := service.GetLatestDiff(uid)
	if errors.Is(err, service.ErrNotFound) {
		response.Success(c, nil)
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "读取课表变动失败")
		return
	}
	response.Success(c, diff)
}
