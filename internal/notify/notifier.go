// Package notify 负责把事件推送到用户开启的通知渠道（Webhook、微信 PushPlus、邮箱、群机器人），
// 并记录“最近动态”。
package notify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"jww/internal/model"
	"jww/internal/store"
)

// 事件类型
const (
	EventScheduleDiff   = "schedule-diff"   // 课表变动
	EventScoreNew       = "score-new"       // 新出成绩
	EventReminder       = "reminder"        // 上课提醒
	EventSessionExpired = "session-expired" // 教务登录失效
	EventTest           = "test"            // 手动测试
)

// ErrNoChannel 用户没有开启任何通知渠道
var ErrNoChannel = errors.New("还没有开启任何通知方式")

// Payload 推送内容
type Payload struct {
	Event string    `json:"event"`
	Text  string    `json:"text"`
	Data  any       `json:"data,omitempty"`
	Time  time.Time `json:"time"`
}

// ChannelResult 单个渠道的发送结果
type ChannelResult struct {
	Channel string `json:"channel"`
	Error   string `json:"error,omitempty"`
}

// Options 通知配置
type Options struct {
	SMTP        SMTPConfig   // Host 为空表示不支持邮件
	PushPlusURL string       // 默认 https://www.pushplus.plus/send
	HTTPClient  *http.Client // 默认 10 秒超时
}

// Notifier 通知发送器
type Notifier struct {
	store       *store.Store
	smtp        *SMTPConfig
	pushPlusURL string
	client      *http.Client
	wg          sync.WaitGroup
}

// New 创建通知发送器
func New(st *store.Store, opts Options) *Notifier {
	n := &Notifier{store: st, pushPlusURL: opts.PushPlusURL, client: opts.HTTPClient}
	if n.pushPlusURL == "" {
		n.pushPlusURL = "https://www.pushplus.plus/send"
	}
	if n.client == nil {
		n.client = &http.Client{Timeout: 10 * time.Second}
	}
	if opts.SMTP.Host != "" && opts.SMTP.User != "" {
		cfg := opts.SMTP
		if cfg.From == "" {
			cfg.From = cfg.User
		}
		n.smtp = &cfg
	}
	return n
}

// EmailAvailable 服务器是否配置了发信邮箱
func (n *Notifier) EmailAvailable() bool { return n.smtp != nil }

// Notify 记录动态并异步推送到用户的所有渠道（失败只记日志）
func (n *Notifier) Notify(uid, event, text string, data any) {
	p := &Payload{Event: event, Text: text, Data: data, Time: time.Now()}
	if event != EventTest && event != EventReminder {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := n.store.PushHistory(ctx, uid, &model.HistoryEntry{Event: event, Text: text, Time: p.Time}); err != nil {
			slog.Warn("record history failed", "uid", uid, "err", err)
		}
		cancel()
	}

	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := n.Send(ctx, uid, p); err != nil && !errors.Is(err, ErrNoChannel) {
			slog.Warn("notify failed", "uid", uid, "event", event, "err", err)
		}
	}()
}

// Wait 等待所有异步推送完成（退出前调用）
func (n *Notifier) Wait() { n.wg.Wait() }

// Send 同步推送到用户的所有渠道；单个渠道失败不影响其它渠道，返回汇总错误
func (n *Notifier) Send(ctx context.Context, uid string, p *Payload) ([]ChannelResult, error) {
	type sender struct {
		name string
		send func() error
	}
	var senders []sender

	if w, err := n.store.Webhook(ctx, uid); err == nil {
		senders = append(senders, sender{"webhook", func() error { return n.SendWebhook(ctx, w, p) }})
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	ch, err := n.store.Channels(ctx, uid)
	if err != nil {
		return nil, err
	}
	if c := ch.PushPlus; c != nil {
		senders = append(senders, sender{"pushplus", func() error { return n.sendPushPlus(ctx, c, p) }})
	}
	if c := ch.Email; c != nil {
		senders = append(senders, sender{"email", func() error { return n.sendEmail(c, p) }})
	}
	if c := ch.Bot; c != nil {
		senders = append(senders, sender{"bot", func() error { return n.sendBot(ctx, c, p) }})
	}
	if len(senders) == 0 {
		return nil, ErrNoChannel
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

// Title 事件标题
func Title(event string) string {
	switch event {
	case EventScheduleDiff:
		return "课表变动提醒"
	case EventScoreNew:
		return "成绩发布提醒"
	case EventReminder:
		return "上课提醒"
	case EventSessionExpired:
		return "教务登录已失效"
	case EventTest:
		return "课表监控测试通知"
	}
	return "课表监控通知"
}
