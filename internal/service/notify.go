package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// 通知事件类型
const (
	EventScheduleDiff   = "schedule-diff"   // 课表变动
	EventSessionExpired = "session-expired" // 教务登录失效，需要重新登录
	EventTest           = "test"            // 手动测试
)

// NotifyPayload 推送给用户通知渠道的内容
type NotifyPayload struct {
	Event string      `json:"event"`
	Text  string      `json:"text"`
	Data  interface{} `json:"data,omitempty"`
	Time  time.Time   `json:"time"`
}

var notifyClient = &http.Client{Timeout: 10 * time.Second}

// notifyUser 异步向用户配置的所有通知渠道推送（目前为 Webhook）
func notifyUser(uid, event, text string, data interface{}) {
	payload := &NotifyPayload{Event: event, Text: text, Data: data, Time: time.Now()}
	go func() {
		if err := SendNotify(uid, payload); err != nil && !errors.Is(err, ErrNotFound) {
			slog.Warn("Notify failed", "uid", uid, "event", event, "err", err)
		}
	}()
}

// SendNotify 同步推送，返回第一个错误；用户没有配置任何渠道时返回 ErrNotFound
func SendNotify(uid string, payload *NotifyPayload) error {
	entry, err := GetWebhook(uid)
	if err != nil {
		return err
	}
	return sendWebhook(entry, payload)
}

func sendWebhook(entry *WebhookEntry, payload *NotifyPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", entry.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-JWW-Event", payload.Event)
	if entry.Secret != "" {
		mac := hmac.New(sha256.New, []byte(entry.Secret))
		mac.Write(body)
		req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := notifyClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook 返回 %d", resp.StatusCode)
	}
	return nil
}
