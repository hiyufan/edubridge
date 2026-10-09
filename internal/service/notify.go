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

// notifyUser 异步向用户开启的所有通知渠道推送
func notifyUser(uid, event, text string, data interface{}) {
	payload := &NotifyPayload{Event: event, Text: text, Data: data, Time: time.Now()}
	go func() {
		if _, err := SendNotify(uid, payload); err != nil && !errors.Is(err, ErrNotFound) {
			slog.Warn("Notify failed", "uid", uid, "event", event, "err", err)
		}
	}()
}

// ChannelResult 单个渠道的发送结果
type ChannelResult struct {
	Channel string `json:"channel"`
	Error   string `json:"error,omitempty"`
}

// SendNotify 同步向用户开启的所有渠道推送；用户没有开启任何渠道时返回 ErrNotFound，
// 任一渠道失败时返回汇总错误（其它渠道照常发送）
func SendNotify(uid string, payload *NotifyPayload) ([]ChannelResult, error) {
	type sender struct {
		name string
		send func() error
	}
	var senders []sender

	if entry, err := GetWebhook(uid); err == nil {
		senders = append(senders, sender{"webhook", func() error { return SendWebhook(entry, payload) }})
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	channels, err := GetNotifyChannels(uid)
	if err != nil {
		return nil, err
	}
	if ch := channels.PushPlus; ch != nil {
		senders = append(senders, sender{"pushplus", func() error { return sendPushPlus(ch, payload) }})
	}
	if ch := channels.Email; ch != nil {
		senders = append(senders, sender{"email", func() error { return sendEmail(ch, payload) }})
	}
	if ch := channels.Bot; ch != nil {
		senders = append(senders, sender{"bot", func() error { return sendBot(ch, payload) }})
	}
	if len(senders) == 0 {
		return nil, ErrNotFound
	}

	results := make([]ChannelResult, len(senders))
	var errs []error
	for i, s := range senders {
		results[i].Channel = s.name
		if err := s.send(); err != nil {
			results[i].Error = err.Error()
			errs = append(errs, fmt.Errorf("%s: %w", s.name, err))
		}
	}
	return results, errors.Join(errs...)
}

// SendWebhook 只向 Webhook 推送
func SendWebhook(entry *WebhookEntry, payload *NotifyPayload) error {
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
