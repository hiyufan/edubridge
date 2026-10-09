package store

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"jww/internal/model"
)

const historyMax = 30

// SaveWebhook 保存（覆盖）用户的 Webhook
func (s *Store) SaveWebhook(ctx context.Context, uid string, w *model.WebhookConfig) error {
	return s.setJSON(ctx, keyWebhook+uid, w, 0)
}

// Webhook 用户的 Webhook
func (s *Store) Webhook(ctx context.Context, uid string) (*model.WebhookConfig, error) {
	var w model.WebhookConfig
	if err := s.getJSON(ctx, keyWebhook+uid, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

// SaveChannels 保存通知渠道
func (s *Store) SaveChannels(ctx context.Context, uid string, c *model.NotifyChannels) error {
	return s.setJSON(ctx, keyChannels+uid, c, 0)
}

// Channels 通知渠道；未配置时返回空配置
func (s *Store) Channels(ctx context.Context, uid string) (*model.NotifyChannels, error) {
	c := &model.NotifyChannels{}
	if err := s.getJSON(ctx, keyChannels+uid, c); err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return c, nil
}

// PushHistory 记录一条动态，只保留最近 30 条
func (s *Store) PushHistory(ctx context.Context, uid string, e *model.HistoryEntry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.LPush(ctx, keyHistory+uid, data)
		p.LTrim(ctx, keyHistory+uid, 0, historyMax-1)
		return nil
	})
	return err
}

// History 最近的动态（新的在前）
func (s *Store) History(ctx context.Context, uid string, limit int) ([]model.HistoryEntry, error) {
	items, err := s.rdb.LRange(ctx, keyHistory+uid, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]model.HistoryEntry, 0, len(items))
	for _, it := range items {
		var e model.HistoryEntry
		if json.Unmarshal([]byte(it), &e) == nil {
			out = append(out, e)
		}
	}
	return out, nil
}

// 监控结果类型
const (
	MonitorCheck     = "lastCheck"
	MonitorKeepalive = "lastKeepalive"
)

// RecordMonitorResult 记录一次检查/保活的时间和结果
func (s *Store) RecordMonitorResult(ctx context.Context, uid, kind string, result error) error {
	msg := ""
	if result != nil {
		msg = result.Error()
	}
	return s.rdb.HSet(ctx, keyMonitorStatus+uid, kind, time.Now().Unix(), kind+"Error", msg).Err()
}

// MonitorStatus 监控运行情况
func (s *Store) MonitorStatus(ctx context.Context, uid string) (*model.MonitorStatus, error) {
	m, err := s.rdb.HGetAll(ctx, keyMonitorStatus+uid).Result()
	if err != nil {
		return nil, err
	}
	parse := func(k string) *time.Time {
		sec, err := strconv.ParseInt(m[k], 10, 64)
		if err != nil {
			return nil
		}
		t := time.Unix(sec, 0)
		return &t
	}
	return &model.MonitorStatus{
		LastCheck:          parse(MonitorCheck),
		LastCheckError:     m[MonitorCheck+"Error"],
		LastKeepalive:      parse(MonitorKeepalive),
		LastKeepaliveError: m[MonitorKeepalive+"Error"],
	}, nil
}
