package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
	"jww/internal/service"
	"jww/pkg/response"
)

// pushTokenStore 内存存储 push token（sessionID -> token）
var (
	pushTokenStore = make(map[string]string)
	pushMu         sync.RWMutex
)

// NotifyHandler 通知处理器
type NotifyHandler struct{}

// NewNotifyHandler 创建通知处理器
func NewNotifyHandler() *NotifyHandler {
	h := &NotifyHandler{}
	// 启动每日 7:00 提醒 goroutine
	go h.dailyRemindLoop()
	return h
}

// VAPIDKeys VAPID 公私钥（生产环境应从配置读取）
var vapidPrivateKey = "BCk0qAer5hNq6z1zLqM7M2Y3F4Y5Z6A7B8C9D0E1F2G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4"
var vapidPublicKey = "BCk0qAer5hNq6z1zLqM7M2Y3F4Y5Z6A7B8C9D0E1F2G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4=" // 示例公钥，实际需换有效的

// RegisterToken 注册推送 token
func (h *NotifyHandler) RegisterToken(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "缺少 token")
		return
	}

	// 同步到 JwService Session
	jwSvc := service.GetJwService()
	jwSvc.SetPushToken(sessionIDStr, req.Token)

	slog.Info("Push token registered", "sessionID", sessionIDStr)
	response.Success(c, gin.H{"message": "token 注册成功"})
}

// TestNotify 发送测试通知
func (h *NotifyHandler) TestNotify(c *gin.Context) {
	sessionIDStr, ok := getSessionID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "无效的会话")
		return
	}

	pushMu.RLock()
	token, exists := pushTokenStore[sessionIDStr]
	pushMu.RUnlock()

	if !exists || token == "" {
		response.Error(c, http.StatusBadRequest, "未注册推送 token，请先注册")
		return
	}

	// 发送测试通知
	notif := map[string]interface{}{
		"title": "📚 课程提醒测试",
		"body":  "您的课程提醒功能已开启，明天早上 7:00 将收到课程通知",
		"icon":  "/favicon.ico",
	}
	payload, _ := json.Marshal(notif)

	err := sendWebPush(token, payload)
	if err != nil {
		slog.Error("Test notify failed", "err", err)
		response.Error(c, http.StatusInternalServerError, "发送失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "测试通知已发送"})
}

// sendWebPush 发送 Web Push 通知
func sendWebPush(token string, payload []byte) error {
	sub := &webpush.Subscription{
		Endpoint: token,
		Keys: webpush.Keys{
			P256dh: "BNcRdreALrFX5O1yGqD2X5P4H8L6N7M2K9J1I3S0R5T7U6V4W2X1Y3Z5A7B8C9D0E1F2G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4=",
			Auth:    "bbzT2Z3qY9F7G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4=",
		},
	}

	resp, err := webpush.SendNotification(payload, sub, &webpush.Options{
		VAPIDPublicKey:  vapidPublicKey,
		VAPIDPrivateKey: vapidPrivateKey,
		TTL:             3600,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// dailyRemindLoop 每日 7:00 课程提醒循环
func (h *NotifyHandler) dailyRemindLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		now := time.Now()
		// 计算距离下一个 7:00 的时间
		next7 := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
		if now.After(next7) {
			next7 = next7.AddDate(0, 0, 1)
		}
		waitDuration := next7.Sub(now)

		// 等待到 7:00
		select {
		case <-time.After(waitDuration):
		case <-time.After(24 * time.Hour): // fallback
		}

		// 执行提醒
		h.sendDailyReminders()
	}
}

// sendDailyReminders 发送每日课程提醒
func (h *NotifyHandler) sendDailyReminders() {
	slog.Info("Sending daily course reminders")

	jwSvc := service.GetJwService()
	jwSvc.ForEachSession(func(sessionID string, token string) bool {
		if token == "" {
			return true
		}

		// 获取当天课表（简化：获取全量课表后过滤今天）
		fullSchedule, err := jwSvc.GetFullSchedule(sessionID, 20)
		if err != nil || fullSchedule == nil {
			return true
		}

		// 找到今天的课程
		todayWeekday := int(time.Now().Weekday())
		if todayWeekday == 0 {
			todayWeekday = 7 // 周日
		}

		var todayCourses []string
		_, currentWeek, _ := jwSvc.GetScheduleTimeInfo(sessionID)
		for _, c := range fullSchedule.Courses {
			if c.DayOfWeek == todayWeekday {
				// 检查当前周是否有这门课
				for _, w := range c.Weeks {
					if w == currentWeek {
						todayCourses = append(todayCourses, c.Name)
						break
					}
				}
			}
		}

		if len(todayCourses) == 0 {
			return true
		}

		// 构建通知内容
		notif := map[string]interface{}{
			"title": "📚 今日课程提醒",
			"body":  "今天有 " + string(rune('0'+len(todayCourses))) + " 节课：" + joinStrings(todayCourses, "、"),
			"icon":  "/favicon.ico",
		}
		payload, _ := json.Marshal(notif)

		err = sendWebPush(token, payload)
		if err != nil {
			slog.Warn("Failed to send daily reminder", "sessionID", sessionID, "err", err)
		} else {
			slog.Info("Daily reminder sent", "sessionID", sessionID, "courses", len(todayCourses))
		}
		return true
	})
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
