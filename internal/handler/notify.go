package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"jww/internal/service"
	"jww/pkg/response"
)

// NotifyHandler 通知与课表监控
type NotifyHandler struct{}

// NewNotifyHandler 创建通知处理器
func NewNotifyHandler() *NotifyHandler {
	return &NotifyHandler{}
}

// TestNotify 向用户开启的所有通知渠道发送一条测试消息，返回各渠道结果
func (h *NotifyHandler) TestNotify(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	results, err := service.SendNotify(uid, &service.NotifyPayload{
		Event: service.EventTest,
		Text:  "这是一条测试通知，课表有加课、减课时会通过这里提醒你。",
		Time:  time.Now(),
	})
	if errors.Is(err, service.ErrNotFound) {
		response.Error(c, http.StatusBadRequest, "还没有开启任何通知方式")
		return
	}
	if err != nil && results == nil {
		response.Error(c, http.StatusInternalServerError, "读取通知配置失败")
		return
	}
	// 部分渠道失败也返回 200，由前端按渠道展示结果
	response.Success(c, gin.H{"results": results})
}

// GetChannels 获取用户的通知渠道配置
func (h *NotifyHandler) GetChannels(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}
	channels, err := service.GetNotifyChannels(uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "读取通知配置失败")
		return
	}
	response.Success(c, gin.H{
		"channels":       channels,
		"emailAvailable": service.EmailAvailable(),
	})
}

// SaveChannels 保存用户的通知渠道配置（整体覆盖，未提供的渠道视为关闭）
func (h *NotifyHandler) SaveChannels(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}
	var channels service.NotifyChannels
	if err := c.ShouldBindJSON(&channels); err != nil {
		response.Error(c, http.StatusBadRequest, "参数格式不正确")
		return
	}
	if err := channels.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := service.SaveNotifyChannels(uid, &channels); err != nil {
		response.Error(c, http.StatusInternalServerError, "保存通知配置失败")
		return
	}
	response.Success(c, gin.H{"channels": channels})
}

// MonitorStatus 课表监控状态及最近一次检测到的变动
func (h *NotifyHandler) MonitorStatus(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	monitored, err := service.IsMonitored(uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "读取监控状态失败")
		return
	}
	_, webhookErr := service.GetWebhook(uid)
	channels, err := service.GetNotifyChannels(uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "读取通知配置失败")
		return
	}

	result := gin.H{
		"monitoring":    monitored,
		"notifyChannel": webhookErr == nil || channels.Any(),
	}
	if diff, err := service.GetLatestDiff(uid); err == nil {
		result["lastDiff"] = diff
		result["lastDiffText"] = diff.Summary()
	}
	response.Success(c, result)
}
