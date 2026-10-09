package notify

import (
	"context"
	"time"

	"jww/internal/model"
)

// Webhook 用户的 Webhook（未注册时返回 store.ErrNotFound）
func (n *Notifier) Webhook(ctx context.Context, uid string) (*model.WebhookConfig, error) {
	return n.store.Webhook(ctx, uid)
}

// SaveWebhook 注册（覆盖）用户的 Webhook
func (n *Notifier) SaveWebhook(ctx context.Context, uid, url, secret string) error {
	return n.store.SaveWebhook(ctx, uid, &model.WebhookConfig{URL: url, Secret: secret, Registered: time.Now()})
}

// Channels 用户的通知渠道
func (n *Notifier) Channels(ctx context.Context, uid string) (*model.NotifyChannels, error) {
	return n.store.Channels(ctx, uid)
}

// SaveChannels 校验并保存通知渠道（整体覆盖，未提供的渠道视为关闭）
func (n *Notifier) SaveChannels(ctx context.Context, uid string, c *model.NotifyChannels) error {
	if err := n.ValidateChannels(c); err != nil {
		return &ValidationError{err.Error()}
	}
	return n.store.SaveChannels(ctx, uid, c)
}

// HasChannel 用户是否开启了任一通知渠道（含 Webhook）
func (n *Notifier) HasChannel(ctx context.Context, uid string) (bool, error) {
	if _, err := n.store.Webhook(ctx, uid); err == nil {
		return true, nil
	}
	c, err := n.store.Channels(ctx, uid)
	if err != nil {
		return false, err
	}
	return c.Any(), nil
}

// History 最近的动态
func (n *Notifier) History(ctx context.Context, uid string, limit int) ([]model.HistoryEntry, error) {
	return n.store.History(ctx, uid, limit)
}

// ValidationError 用户提交的配置不合法，信息可直接展示
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }
