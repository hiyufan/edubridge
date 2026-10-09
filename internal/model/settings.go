package model

import "time"

// WebhookConfig 用户自定义 Webhook
type WebhookConfig struct {
	URL        string    `json:"url"`
	Secret     string    `json:"secret"`
	Registered time.Time `json:"registered"`
}

// NotifyChannels 用户的通知渠道（nil 表示未开启）
type NotifyChannels struct {
	PushPlus *PushPlusChannel `json:"pushplus"`
	Email    *EmailChannel    `json:"email"`
	Bot      *BotChannel      `json:"bot"`
}

// Any 是否至少开启了一个渠道
func (c *NotifyChannels) Any() bool {
	return c != nil && (c.PushPlus != nil || c.Email != nil || c.Bot != nil)
}

// PushPlusChannel 微信推送（https://www.pushplus.plus）
type PushPlusChannel struct {
	Token string `json:"token"`
}

// EmailChannel 邮件推送
type EmailChannel struct {
	To string `json:"to"`
}

// BotChannel 群机器人
type BotChannel struct {
	Type   string `json:"type"`   // wecom | dingtalk | feishu
	URL    string `json:"url"`    // 机器人 Webhook 地址
	Secret string `json:"secret"` // 钉钉/飞书“加签”密钥（可选）
}

// ReminderSettings 上课提醒设置
type ReminderSettings struct {
	Daily         bool   `json:"daily"`
	DailyTime     string `json:"dailyTime"`     // "07:00"
	DailyTomorrow bool   `json:"dailyTomorrow"` // true 时推送明天的课
	BeforeClass   bool   `json:"beforeClass"`
	BeforeMinutes int    `json:"beforeMinutes"`
}

// HistoryEntry 一条动态（课表变动、新成绩、登录失效）
type HistoryEntry struct {
	Event string    `json:"event"`
	Text  string    `json:"text"`
	Time  time.Time `json:"time"`
}

// MonitorStatus 后台监控运行情况
type MonitorStatus struct {
	LastCheck          *time.Time `json:"lastCheck"`
	LastCheckError     string     `json:"lastCheckError,omitempty"`
	LastKeepalive      *time.Time `json:"lastKeepalive"`
	LastKeepaliveError string     `json:"lastKeepaliveError,omitempty"`
}
