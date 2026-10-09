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

// TestNotify 向用户配置的通知渠道发送一条测试消息
func (h *NotifyHandler) TestNotify(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	err := service.SendNotify(uid, &service.NotifyPayload{
		Event: service.EventTest,
		Text:  "这是一条测试通知，课表有加课、减课时会通过这里提醒你。",
		Time:  time.Now(),
	})
	if errors.Is(err, service.ErrNotFound) {
		response.Error(c, http.StatusBadRequest, "还没有配置通知渠道")
		return
	}
	if err != nil {
		response.Error(c, http.StatusBadGateway, "发送失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "测试通知已发送"})
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

	result := gin.H{
		"monitoring":    monitored,
		"notifyChannel": webhookErr == nil,
	}
	if diff, err := service.GetLatestDiff(uid); err == nil {
		result["lastDiff"] = diff
		result["lastDiffText"] = diff.Summary()
	}
	response.Success(c, result)
}
